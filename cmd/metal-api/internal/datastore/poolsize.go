package datastore

import (
	e "github.com/metal-stack/metal-api/cmd/metal-api/internal/eventbus"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-lib/bus"
	"go.uber.org/zap"
)

type manager struct {
	rs          *RethinkStore
	publisher   bus.Publisher
	partitionid string
	sizeid      string
}

func (m *manager) AllMachines() (metal.Machines, error) {
	q := MachineSearchQuery{
		PartitionID: &m.partitionid,
		SizeID:      &m.sizeid,
	}

	allMachines := metal.Machines{}
	err := m.rs.SearchMachines(&q, &allMachines)
	if err != nil {
		return nil, err
	}

	return allMachines, nil
}

func (m *manager) WaitingMachines() (metal.Machines, error) {
	stateValue := string(metal.AvailableState)
	notAllocated := true
	q := MachineSearchQuery{
		PartitionID:  &m.partitionid,
		SizeID:       &m.sizeid,
		StateValue:   &stateValue,
		NotAllocated: &notAllocated,
	}

	waitingMachines := metal.Machines{}
	err := m.rs.SearchMachines(&q, &waitingMachines)
	if err != nil {
		return nil, err
	}

	return waitingMachines, nil
}

func (m *manager) ShutdownMachines() (metal.Machines, error) {
	sleeping := true
	q := MachineSearchQuery{
		PartitionID: &m.partitionid,
		SizeID:      &m.sizeid,
		Sleeping:    &sleeping,
	}

	shutdownMachines := metal.Machines{}
	err := m.rs.SearchMachines(&q, &shutdownMachines)
	if err != nil {
		return nil, err
	}

	return shutdownMachines, nil
}

func (m *manager) Shutdown(machine *metal.Machine) error {
	state := metal.MachineState{
		Sleeping:    true,
		Description: "shut down as exceeding maximum partition poolsize",
	}

	err := m.rs.publishCommandAndUpdate(m.rs.log, machine, m.publisher, metal.MachineOffCmd, state)
	if err != nil {
		return err
	}
	return nil
}

func (m *manager) PowerOn(machine *metal.Machine) error {
	state := metal.MachineState{Sleeping: false}

	err := m.rs.publishCommandAndUpdate(m.rs.log, machine, m.publisher, metal.MachineOnCmd, state)
	if err != nil {
		return err
	}
	return nil
}

func (rs *RethinkStore) publishCommandAndUpdate(logger *zap.SugaredLogger, m *metal.Machine, publisher bus.Publisher, cmd metal.MachineCommand, state metal.MachineState) error {
	newMachine := *m
	newMachine.State = state
	err := rs.UpdateMachine(m, &newMachine)
	if err != nil {
		return err
	}

	err = e.PublishMachineCmd(logger, m, publisher, cmd)
	if err != nil {
		return err
	}

	return nil
}
