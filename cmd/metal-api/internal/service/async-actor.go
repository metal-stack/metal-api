package service

import (
	"errors"
	"fmt"

	ipamer "github.com/metal-stack/go-ipam"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/ipam"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-lib/bus"
	"go.uber.org/zap"
)

type asyncActor struct {
	*zap.Logger
	ipam.IPAMer
	*datastore.RethinkStore
	machineReleaser bus.Func
	ipReleaser      bus.Func
}

func newAsyncActor(l *zap.Logger, ep *bus.Endpoints, ds *datastore.RethinkStore, ip ipam.IPAMer) (*asyncActor, error) {
	actor := &asyncActor{
		Logger:       l,
		IPAMer:       ip,
		RethinkStore: ds,
	}
	var err error
	actor.machineReleaser, err = ep.Function("releaseMachineNetworks", actor.releaseMachineNetworks)
	if err != nil {
		return nil, fmt.Errorf("cannot create async bus function for machine releasing: %w", err)
	}
	actor.ipReleaser, err = ep.Function("releaseIP", actor.releaseIP)
	if err != nil {
		return nil, fmt.Errorf("cannot create bus function for ip releasing: %w", err)
	}
	return actor, nil
}

func (a *asyncActor) freeMachine(pub bus.Publisher, m *metal.Machine) error {
	if m.State.Value == metal.LockedState {
		return fmt.Errorf("machine is locked")
	}

	err := deleteVRFSwitches(a.RethinkStore, m, a.Logger)
	if err != nil {
		return err
	}

	err = publishDeleteEvent(pub, m, a.Logger)
	if err != nil {
		return err
	}

	// call the releaser async
	err = a.machineReleaser(m)
	if err != nil {
		a.Error("cannot call async machine cleanup", zap.Error(err))
	}

	old := *m

	m.Allocation = nil
	m.Tags = nil

	err = a.UpdateMachine(&old, m)
	if err != nil {
		return err
	}
	a.Infow("freed machine", "machineID", m.ID)

	return nil
}

func (a *asyncActor) releaseMachineNetworks(machine *metal.Machine) error {
	if machine.Allocation == nil {
		return nil
	}
	for _, machineNetwork := range machine.Allocation.MachineNetworks {
		for _, ipString := range machineNetwork.IPs {
			ip, err := a.FindIPByID(ipString)
			if err != nil {
				return err
			}
			// ignore ips that were associated with the machine for allocation but the association is not present anymore at the ip
			if !ip.HasMachineId(machine.GetID()) {
				continue
			}
			// disassociate machine from ip
			new := *ip
			new.RemoveMachineId(machine.GetID())
			err = a.UpdateIP(ip, &new)
			if err != nil {
				return err
			}
			// static ips should not be released automatically
			if ip.Type == metal.Static {
				continue
			}
			// ips that are associated to other machines will should not be released automatically
			if len(new.GetMachineIds()) > 0 {
				continue
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
		}
	}
	return nil
}

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
		s := dbip.GetScope()
		if s != metal.ScopeProject && ip.Type == metal.Static {
			a.Info("do not delete static IP", zap.String("scope", string(s)))
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
