package service

import (
	"errors"
	"fmt"

	ipamer "github.com/metal-stack/go-ipam"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/ipam"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-lib/bus"
	"github.com/metal-stack/metal-lib/pkg/tag"
	"go.uber.org/zap"
)

type asyncActor struct {
	*zap.Logger
	ipam.IPAMer
	pub bus.Publisher
	*datastore.RethinkStore
	machineReleaser        bus.Func
	machineNetworkReleaser bus.Func
	ipReleaser             bus.Func
}

func newAsyncActor(l *zap.Logger, ep *bus.Endpoints, ds *datastore.RethinkStore, ip ipam.IPAMer, pub bus.Publisher) (*asyncActor, error) {
	actor := &asyncActor{
		Logger:       l,
		IPAMer:       ip,
		RethinkStore: ds,
		pub:          pub,
	}
	var err error
	_, actor.machineReleaser, err = ep.Function("releaseMachine", actor.releaseMachine)
	if err != nil {
		return nil, fmt.Errorf("cannot create async bus function for machine releasing: %w", err)
	}
	_, actor.machineNetworkReleaser, err = ep.Function("releaseMachineNetworks", actor.releaseMachineNetworks)
	if err != nil {
		return nil, fmt.Errorf("cannot create async bus function for machine network releasing: %w", err)
	}
	_, actor.ipReleaser, err = ep.Function("releaseIP", actor.releaseIP)
	if err != nil {
		return nil, fmt.Errorf("cannot create async bus function for ip releasing: %w", err)
	}
	return actor, nil
}

func (a *asyncActor) freeMachine(m *metal.Machine) error {
	if m.State.Value == metal.LockedState {
		return fmt.Errorf("machine is locked")
	}

	err := a.machineReleaser(m)
	if err != nil {
		return err
	}

	old := *m

	m.Allocation = nil
	m.Tags = nil

	err = a.UpdateMachine(&old, m)
	if err != nil {
		return err
	}
	a.Info("freed machine", zap.String("machineID", m.ID))

	return nil
}

func (a *asyncActor) releaseMachine(machine *metal.Machine) error {
	err := deleteVRFSwitches(a.RethinkStore, machine, a.Logger)
	if err != nil {
		return err
	}

	err = publishDeleteEvent(a.pub, machine, a.Logger)
	if err != nil {
		return err
	}

	err = a.machineReleaser(machine)
	if err != nil {
		// what should we do here? this error shows a problem with the nsq-bus system
		a.Error("error calling machine network releaser", zap.Error(err))
	}

	return nil
}

func (a *asyncActor) releaseMachineNetworks(machine *metal.Machine) error {
	if machine.Allocation == nil {
		return nil
	}
	var asn uint32
	for _, machineNetwork := range machine.Allocation.MachineNetworks {
		if machineNetwork.IPs == nil {
			continue
		}
		for _, ipString := range machineNetwork.IPs {
			ip, err := a.FindIPByID(ipString)
			if err != nil {
				if metal.IsNotFound(err) {
					// if we do not skip here we will always fail releasing the next ip addresses
					// after the first ip was released
					continue
				}
				return err
			}

			err = a.disassociateIP(ip, machine)
			if err != nil {
				return err
			}
		}
		// all machineNetworks have the same ASN, must only be released once
		// in the old way asn was in the range of 4200000000 + offset from last two octets of private ip
		// but we must only release asn which are acquired from 4210000000 and above from the ASNIntegerPool
		if machineNetwork.ASN >= ASNBase {
			asn = machineNetwork.ASN
		}
	}
	if asn >= ASNBase {
		err := releaseASN(a.RethinkStore, asn)
		if err != nil {
			return err
		}
	}

	// it can happen that an IP gets properly allocated for a machine but
	// the machine was not added to the machine network. We call these
	// IPs "dangling".
	var danglingIPs metal.IPs
	err := a.SearchIPs(&datastore.IPSearchQuery{
		Tags: []string{metal.IpTag(tag.MachineID, machine.ID)},
	}, &danglingIPs)
	if err != nil {
		return err
	}
	for _, ip := range danglingIPs {
		err = a.disassociateIP(&ip, machine)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *asyncActor) disassociateIP(ip *metal.IP, machine *metal.Machine) error {
	// ignore ips that were associated with the machine for allocation but the association is not present anymore at the ip
	if !ip.HasMachineId(machine.GetID()) {
		return nil
	}

	// disassociate machine from ip
	new := *ip
	new.RemoveMachineId(machine.GetID())
	err := a.UpdateIP(ip, &new)
	if err != nil {
		return err
	}
	// static ips should not be released automatically
	if ip.Type == metal.Static {
		return nil
	}
	// ips that are associated to other machines will should not be released automatically
	if len(new.GetMachineIds()) > 0 {
		return nil
	}

	// at this point the machine is removed from the IP, so the release of the
	// IP must not fail, because this loop will never come to this point again because the
	// begin of this loop checks if the IP contains the machine.
	// so we fork a new job to delete the IP. if this fails .... well then ("houston we have a problem")
	// we do not report the error to the caller, because this whole function cannot be re-do'ed.
	a.Infow("async release IP", "ip", *ip)
	if err := a.ipReleaser(*ip); err != nil {
		// what should we do here? this error shows a problem with the nsq-bus system
		a.Error("cannot call ip releaser", zap.Error(err))
	}

	return nil
}

// releaseIP releases the given IP. This function only checks if the given IP is connected
// to a machine and only deletes the IP if this is not the case. It is the duty of the caller
// to implement more validations: when a machine is deleted the caller has to check if the IP
// is static or not. If only an IP is freed, the caller has to check if the IP has machine scope.
func (a *asyncActor) releaseIP(ip metal.IP) error {
	a.Info("release IP", zap.Any("ip", ip))

	dbip, err := a.FindIPByID(ip.IPAddress)
	if err != nil && !metal.IsNotFound(err) {
		// some unknown error, we will let nsq resend the command
		a.Error("cannot find IP", zap.Any("ip", ip), zap.Error(err))
		return err
	}

	if err == nil {
		// if someone calls freeMachine for the same machine multiple times at the same
		// moment it can happen that we already deleted and released the IP in the ipam
		// so make sure that this IP is not already connected to a new machine
		if len(dbip.GetMachineIds()) > 0 {
			a.Info("do not delete IP, it is connected to a machine", zap.Any("ip", ip))
			return nil
		}

		// the ip is in our database and is not connected to a machine so cleanup
		err = a.DeleteIP(&ip)
		if err != nil {
			a.Error("cannot delete IP in datastore", zap.Any("ip", ip), zap.Error(err))
			return err
		}
	}

	// now the IP should not exist any more in our datastore
	// so cleanup the ipam

	err = a.ReleaseIP(ip)
	if err != nil {
		if errors.Is(err, ipamer.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("cannot release IP %q: %w", ip, err)
	}
	return nil
}
