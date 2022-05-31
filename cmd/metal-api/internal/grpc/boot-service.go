package grpc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/avast/retry-go/v4"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"github.com/metal-stack/metal-lib/bus"
	"go.uber.org/zap"
)

type BootService struct {
	log          *zap.SugaredLogger
	ds           Datasource
	pwdFile      string
	publisher    bus.Publisher
	eventService *EventService
}

func NewBootService(cfg *ServerConfig, eventService *EventService) *BootService {
	return &BootService{
		ds:           cfg.Datasource,
		log:          cfg.Logger.Named("boot-service"),
		publisher:    cfg.Publisher,
		pwdFile:      cfg.BMCSuperUserPasswordFile,
		eventService: eventService,
	}
}
func (b *BootService) Dhcp(ctx context.Context, req *v1.BootServiceDhcpRequest) (*v1.BootServiceDhcpResponse, error) {
	b.log.Infow("dhcp", "req", req)

	_, err := b.eventService.Send(ctx, &v1.EventServiceSendRequest{Events: map[string]*v1.MachineProvisioningEvent{
		req.Uuid: {
			Event:   string(metal.ProvisioningEventPXEBooting),
			Message: "machine sent extended dhcp request",
		},
	}})
	if err != nil {
		return nil, err
	}
	return &v1.BootServiceDhcpResponse{}, nil
}

func (b *BootService) Boot(ctx context.Context, req *v1.BootServiceBootRequest) (*v1.BootServiceBootResponse, error) {
	b.log.Infow("boot", "req", req)
	if req == nil {
		return nil, fmt.Errorf("req is nil")
	}

	var m *metal.Machine
	err := b.ds.FindMachine(&datastore.MachineSearchQuery{
		NicsMacAddresses: []string{req.Mac},
		PartitionID:      &req.PartitionId,
	}, m)
	if err != nil && !metal.IsNotFound(err) {
		return nil, err
	}

	if m != nil && m.PartitionID != req.PartitionId {
		return nil, fmt.Errorf("partitionID:%q of machine with mac does not match partitionID:%q", m.PartitionID, req.PartitionId)
	}
	p, err := b.ds.FindPartition(req.PartitionId)
	if err != nil || p == nil {
		return nil, fmt.Errorf("no partition with id:%q found %w", req.PartitionId, err)
	}

	resp := &v1.BootServiceBootResponse{
		Kernel:       p.BootConfiguration.KernelURL,
		InitRamDisks: []string{p.BootConfiguration.ImageURL},
		Cmdline:      &p.BootConfiguration.CommandLine,
	}
	b.log.Infow("boot", "resp", resp)
	return resp, nil
}

func (b *BootService) Register(ctx context.Context, req *v1.BootServiceRegisterRequest) (*v1.BootServiceRegisterResponse, error) {
	b.log.Infow("register", "req", req)
	if req.Uuid == "" {
		return nil, errors.New("uuid is empty")
	}
	if req.Hardware == nil {
		return nil, errors.New("hardware is nil")
	}

	disks := []metal.BlockDevice{}
	for i := range req.Hardware.Disks {
		d := req.Hardware.Disks[i]
		disks = append(disks, metal.BlockDevice{
			Name: d.Name,
			Size: d.Size,
		})
	}
	nics := metal.Nics{}

	for i := range req.Hardware.Nics {
		nic := req.Hardware.Nics[i]
		neighs := metal.Nics{}
		for j := range nic.Neighbors {
			neigh := nic.Neighbors[j]
			neighs = append(neighs, metal.Nic{
				Name:       neigh.Name,
				MacAddress: metal.MacAddress(neigh.Mac),
			})
		}
		nics = append(nics, metal.Nic{
			Name:       nic.Name,
			MacAddress: metal.MacAddress(nic.Mac),
			Neighbors:  neighs,
		})
	}

	machineHardware := metal.MachineHardware{
		Memory:   req.Hardware.Memory,
		CPUCores: int(req.Hardware.CpuCores),
		Disks:    disks,
		Nics:     nics,
	}
	size, _, err := b.ds.FromHardware(machineHardware)
	if err != nil {
		size = metal.UnknownSize
		b.log.Errorw("no size found for hardware, defaulting to unknown size", "hardware", machineHardware, "error", err)
	}

	m, err := b.ds.FindMachineByID(req.Uuid)
	if err != nil && !metal.IsNotFound(err) {
		return nil, err
	}

	var ipmi metal.IPMI
	if req.Ipmi != nil {
		i := req.Ipmi

		ipmi = metal.IPMI{
			Address:    i.Address,
			MacAddress: i.Mac,
			User:       i.User,
			Password:   i.Password,
			Interface:  i.Interface,
			BMCVersion: i.BmcVersion,
			PowerState: i.PowerState,
		}
		if i.Fru != nil {
			f := i.Fru
			fru := metal.Fru{}
			if f.ChassisPartNumber != nil {
				fru.ChassisPartNumber = *f.ChassisPartNumber
			}
			if f.ChassisPartSerial != nil {
				fru.ChassisPartSerial = *f.ChassisPartSerial
			}
			if f.BoardMfg != nil {
				fru.BoardMfg = *f.BoardMfg
			}
			if f.BoardMfgSerial != nil {
				fru.BoardMfgSerial = *f.BoardMfgSerial
			}
			if f.BoardPartNumber != nil {
				fru.BoardPartNumber = *f.BoardPartNumber
			}
			if f.ProductManufacturer != nil {
				fru.ProductManufacturer = *f.ProductManufacturer
			}
			if f.ProductPartNumber != nil {
				fru.ProductPartNumber = *f.ProductPartNumber
			}
			if f.ProductSerial != nil {
				fru.ProductSerial = *f.ProductSerial
			}
			ipmi.Fru = fru
		}

	}

	var registerState v1.RegisterState
	if m == nil {
		// machine is not in the database, create it
		m = &metal.Machine{
			Base: metal.Base{
				ID: req.Uuid,
			},
			Allocation: nil,
			SizeID:     size.ID,
			Hardware:   machineHardware,
			BIOS: metal.BIOS{
				Version: req.Bios.Version,
				Vendor:  req.Bios.Vendor,
				Date:    req.Bios.Date,
			},
			State: metal.MachineState{
				Value: metal.AvailableState,
			},
			LEDState: metal.ChassisIdentifyLEDState{
				Value:       metal.LEDStateOff,
				Description: "Machine registered",
			},
			Tags: req.Tags,
			IPMI: ipmi,
		}

		err = b.ds.CreateMachine(m)
		if err != nil {
			return nil, err
		}

		registerState = v1.RegisterState_REGISTER_STATE_CREATED
	} else {
		// machine has already registered, update it
		old := *m

		m.SizeID = size.ID
		m.Hardware = machineHardware
		m.BIOS.Version = req.Bios.Version
		m.BIOS.Vendor = req.Bios.Vendor
		m.BIOS.Date = req.Bios.Date
		m.IPMI = ipmi

		err = b.ds.UpdateMachine(&old, m)
		if err != nil {
			return nil, err
		}
		registerState = v1.RegisterState_REGISTER_STATE_UPDATED
	}

	ec, err := b.ds.FindProvisioningEventContainer(m.ID)
	if err != nil && !metal.IsNotFound(err) {
		return nil, err
	}

	if ec == nil {
		err = b.ds.CreateProvisioningEventContainer(&metal.ProvisioningEventContainer{
			Base:                         metal.Base{ID: m.ID},
			Liveliness:                   metal.MachineLivelinessAlive,
			IncompleteProvisioningCycles: "0",
		},
		)
		if err != nil {
			return nil, err
		}
	}

	old := *m
	err = retry.Do(
		func() error {
			// RackID and PartitionID is set
			err := b.ds.ConnectMachineWithSwitches(m)
			if err != nil {
				return err
			}
			return b.ds.UpdateMachine(&old, m)
		},
		retry.Attempts(10),
		retry.RetryIf(func(err error) bool {
			return strings.Contains(err.Error(), datastore.EntityAlreadyModifiedErrorMessage)
		}),
		retry.DelayType(retry.CombineDelay(retry.BackOffDelay, retry.RandomDelay)),
		retry.LastErrorOnly(true),
	)

	if err != nil {
		return nil, err
	}

	return &v1.BootServiceRegisterResponse{
		Uuid:          req.Uuid,
		Size:          size.ID,
		PartitionId:   m.PartitionID,
		RegisterState: registerState,
	}, nil
}

func (b *BootService) FetchSuperUserPassword(ctx context.Context, req *v1.SuperUserPasswordRequest) (*v1.SuperUserPasswordResponse, error) {
	defer ctx.Done()

	resp := &v1.SuperUserPasswordResponse{}
	if b.pwdFile == "" {
		resp.FeatureDisabled = true
		return resp, nil
	}

	bb, err := os.ReadFile(b.pwdFile)
	if err != nil {
		b.log.Errorw("failed to lookup BMC superuser password", "password file", b.pwdFile, "error", err)
		return nil, err
	}
	resp.FeatureDisabled = false
	resp.SuperUserPassword = strings.TrimSpace(string(bb))
	return resp, nil
}

func (b *BootService) Report(ctx context.Context, req *v1.BootServiceReportRequest) (*v1.BootServiceReportResponse, error) {
	b.log.Infow("report", "req", req)

	// FIXME implement success handling

	m, err := b.ds.FindMachineByID(req.Uuid)
	if err != nil {
		return nil, err
	}
	if m.Allocation == nil {
		return nil, fmt.Errorf("the machine %q is not allocated", req.Uuid)
	}
	if req.BootInfo == nil {
		return nil, fmt.Errorf("the machine %q bootinfo is nil", req.Uuid)
	}

	old := *m

	bootInfo := req.BootInfo

	m.Allocation.ConsolePassword = req.ConsolePassword
	m.Allocation.MachineSetup = &metal.MachineSetup{
		ImageID:      m.Allocation.ImageID,
		PrimaryDisk:  bootInfo.PrimaryDisk,
		OSPartition:  bootInfo.OsPartition,
		Initrd:       bootInfo.Initrd,
		Cmdline:      bootInfo.Cmdline,
		Kernel:       bootInfo.Kernel,
		BootloaderID: bootInfo.BootloaderId,
	}
	m.Allocation.Reinstall = false

	err = b.ds.UpdateMachine(&old, m)
	if err != nil {
		return nil, err
	}

	vrf := ""
	if m.IsFirewall() {
		// if a machine has multiple networks, it serves as firewall, so it can not be enslaved into the tenant vrf
		vrf = "default"
	} else {
		for _, mn := range m.Allocation.MachineNetworks {
			if mn.Private {
				vrf = fmt.Sprintf("vrf%d", mn.Vrf)
				break
			}
		}
	}

	err = retry.Do(
		func() error {
			_, err := b.ds.SetVrfAtSwitches(m, vrf)
			return err
		},
		retry.Attempts(10),
		retry.RetryIf(func(err error) bool {
			return strings.Contains(err.Error(), datastore.EntityAlreadyModifiedErrorMessage)
		}),
		retry.DelayType(retry.CombineDelay(retry.BackOffDelay, retry.RandomDelay)),
		retry.LastErrorOnly(true),
	)
	if err != nil {
		return nil, fmt.Errorf("the machine %q could not be enslaved into the vrf %s, error: %w", req.Uuid, vrf, err)
	}

	b.setBootOrderDisk(m.ID, m.PartitionID)
	return &v1.BootServiceReportResponse{}, nil
}
func (b *BootService) AbortReinstall(ctx context.Context, req *v1.BootServiceAbortReinstallRequest) (*v1.BootServiceAbortReinstallResponse, error) {
	b.log.Infow("abortreinstall", "req", req)
	m, err := b.ds.FindMachineByID(req.Uuid)
	if err != nil {
		return nil, err
	}

	var bootInfo *v1.BootInfo

	if m.Allocation != nil && !req.PrimaryDiskWiped {
		old := *m

		m.Allocation.Reinstall = false
		if m.Allocation.MachineSetup != nil {
			m.Allocation.ImageID = m.Allocation.MachineSetup.ImageID
		}

		err = b.ds.UpdateMachine(&old, m)
		if err != nil {
			return nil, err
		}
		b.log.Infow("removed reinstall mark", "machineID", m.ID)

		if m.Allocation.MachineSetup != nil {
			bootInfo = &v1.BootInfo{
				ImageId:      m.Allocation.MachineSetup.ImageID,
				PrimaryDisk:  m.Allocation.MachineSetup.PrimaryDisk,
				OsPartition:  m.Allocation.MachineSetup.OSPartition,
				Initrd:       m.Allocation.MachineSetup.Initrd,
				Cmdline:      m.Allocation.MachineSetup.Cmdline,
				Kernel:       m.Allocation.MachineSetup.Kernel,
				BootloaderId: m.Allocation.MachineSetup.BootloaderID,
			}
		}
	}
	b.setBootOrderDisk(m.ID, m.PartitionID)
	// FIXME what to do in the else case ?

	return &v1.BootServiceAbortReinstallResponse{BootInfo: bootInfo}, nil
}

func (b *BootService) setBootOrderDisk(machineID, partitionID string) {
	evt := metal.MachineEvent{
		Type: metal.COMMAND,
		Cmd: &metal.MachineExecCommand{
			Command:         metal.MachineDiskCmd,
			Params:          nil,
			TargetMachineID: machineID,
		},
	}

	b.log.Infow("publish event", "event", evt, "command", *evt.Cmd)
	err := b.publisher.Publish(metal.TopicMachine.GetFQN(partitionID), evt)
	if err != nil {
		b.log.Errorw("unable to send boot via hd, continue anyway", "error", err)
	}
}
