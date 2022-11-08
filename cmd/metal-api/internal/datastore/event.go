package datastore

import (
	"crypto/rand"
	"math"
	"math/big"
	mathrand "math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/fsm/states"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-lib/bus"
	"go.uber.org/zap"
)

// ListProvisioningEventContainers returns all machine provisioning event containers.
func (rs *RethinkStore) ListProvisioningEventContainers() (metal.ProvisioningEventContainers, error) {
	es := make(metal.ProvisioningEventContainers, 0)
	err := rs.listEntities(rs.eventTable(), &es)
	return es, err
}

// FindProvisioningEventContainer finds a provisioning event container to a given machine id.
func (rs *RethinkStore) FindProvisioningEventContainer(id string) (*metal.ProvisioningEventContainer, error) {
	var e metal.ProvisioningEventContainer
	err := rs.findEntityByID(rs.eventTable(), &e, id)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// UpdateProvisioningEventContainer updates a provisioning event container.
func (rs *RethinkStore) UpdateProvisioningEventContainer(old *metal.ProvisioningEventContainer, new *metal.ProvisioningEventContainer) error {
	return rs.updateEntity(rs.eventTable(), new, old)
}

// CreateProvisioningEventContainer creates a new provisioning event container.
func (rs *RethinkStore) CreateProvisioningEventContainer(ec *metal.ProvisioningEventContainer) error {
	return rs.createEntity(rs.eventTable(), ec)
}

// UpsertProvisioningEventContainer inserts a machine's event container.
func (rs *RethinkStore) UpsertProvisioningEventContainer(ec *metal.ProvisioningEventContainer) error {
	return rs.upsertEntity(rs.eventTable(), ec)
}
func (rs *RethinkStore) ProvisioningEventForMachine(log *zap.SugaredLogger, publisher bus.Publisher, event *metal.ProvisioningEvent, machine *metal.Machine) (*metal.ProvisioningEventContainer, error) {
	ec, err := rs.FindProvisioningEventContainer(machine.ID)
	if err != nil && !metal.IsNotFound(err) {
		return nil, err
	}

	if ec == nil {
		ec = &metal.ProvisioningEventContainer{
			Base: metal.Base{
				ID: machine.ID,
			},
			Liveliness: metal.MachineLivelinessAlive,
		}
	}

	config := states.StateConfig{
		Log:       log,
		Container: ec,
		Event:     event,
		Message:   states.FSMMessageEmpty,
	}

	newEC, err := fsm.HandleProvisioningEvent(&config)
	if err != nil {
		return nil, err
	}

	if err = newEC.Validate(); err != nil {
		return nil, err
	}

	newEC.TrimEvents(100)
	err = rs.UpsertProvisioningEventContainer(newEC)
	if err != nil {
		return nil, err
	}

	if config.Message == states.FSMMessageEnterWaitingState || config.Message == states.FSMMessageLeaveWaitingState {
		err = rs.AdjustNumberOfWaitingMachines(log, publisher, machine)
	}

	return newEC, err
}

func (rs *RethinkStore) AdjustNumberOfWaitingMachines(log *zap.SugaredLogger, publisher bus.Publisher, machine *metal.Machine) error {
	p, err := rs.FindPartition(machine.PartitionID)
	if err != nil {
		return err
	}

	waitingMachines, err := rs.listWaitingMachines(machine.PartitionID, machine.SizeID)
	if err != nil {
		return err
	}

	poolSizeExceeded := 0
	if strings.Contains(p.WaitingPoolSize, "%") {
		poolSizeExceeded = rs.waitingMachinesMatchPoolSize(len(waitingMachines), p.WaitingPoolSize, true, machine)
	} else {
		poolSizeExceeded = rs.waitingMachinesMatchPoolSize(len(waitingMachines), p.WaitingPoolSize, false, machine)
	}

	if poolSizeExceeded == 0 {
		return nil
	}

	if poolSizeExceeded > 0 {
		m := randomMachine(waitingMachines)
		err = rs.machineCommand(log, &m, publisher, metal.MachineOffCmd, metal.ShutdownState)
		if err != nil {
			return err
		}
	} else {
		shutdownMachines, err := rs.listShutdownMachines(machine.PartitionID, machine.SizeID)
		if err != nil {
			return err
		}

		m := randomMachine(shutdownMachines)
		err = rs.machineCommand(log, &m, publisher, metal.MachineOnCmd, metal.AvailableState)
		if err != nil {
			return err
		}
	}

	return nil
}

func (rs *RethinkStore) waitingMachinesMatchPoolSize(actual int, required string, inPercent bool, m *metal.Machine) int {
	if !inPercent {
		r, err := strconv.ParseInt(required, 10, 32)
		if err != nil {
			return 0
		}
		return actual - int(r)
	}

	p := strings.Split(required, "%")[0]
	r, err := strconv.ParseInt(p, 10, 32)
	if err != nil {
		return 0
	}

	allMachines, err := rs.listMachinesInPartition(m.PartitionID, m.SizeID)
	if err != nil {
		return 0
	}

	return actual - int(math.Round(float64(len(allMachines))*float64(r)/100))
}

func randomMachine(machines metal.Machines) metal.Machine {
	var idx int
	b, err := rand.Int(rand.Reader, big.NewInt(int64(len(machines)))) //nolint:gosec
	if err != nil {
		idx = int(b.Uint64())
	} else {
		mathrand.Seed(time.Now().UnixNano())
		// nolint
		idx = mathrand.Intn(len(machines))
	}

	return machines[idx]
}

func (rs *RethinkStore) machineCommand(logger *zap.SugaredLogger, m *metal.Machine, publisher bus.Publisher, cmd metal.MachineCommand, state metal.MState) error {
	evt := metal.MachineEvent{
		Type: metal.COMMAND,
		Cmd: &metal.MachineExecCommand{
			Command:         cmd,
			TargetMachineID: m.ID,
			IPMI:            &m.IPMI,
		},
	}

	newMachine := *m
	newMachine.State = metal.MachineState{Value: state}
	err := rs.UpdateMachine(m, &newMachine)
	if err != nil {
		return err
	}

	logger.Infow("publish event", "event", evt, "command", *evt.Cmd)
	err = publisher.Publish(metal.TopicMachine.GetFQN(m.PartitionID), evt)
	if err != nil {
		return err
	}

	return nil
}
