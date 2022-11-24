package datastore

import (
	e "github.com/metal-stack/metal-api/cmd/metal-api/internal/eventbus"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	s "github.com/metal-stack/metal-api/cmd/metal-api/internal/scaler"
	"github.com/metal-stack/metal-lib/bus"
	"go.uber.org/zap"
)

type Scaler struct {
	rs        *RethinkStore
	publisher bus.Publisher
}

func (s *Scaler) AllMachines(partitionid, sizeid string) (metal.Machines, error) {
	q := MachineSearchQuery{
		PartitionID: &partitionid,
		SizeID:      &sizeid,
	}

	allMachines := metal.Machines{}
	err := s.rs.SearchMachines(&q, &allMachines)
	if err != nil {
		return nil, err
	}

	return allMachines, nil
}

func (s *Scaler) WaitingMachines(partitionid, sizeid string) (machines metal.Machines, err error) {
	stateValue := string(metal.AvailableState)
	waiting := true
	q := MachineSearchQuery{
		PartitionID: &partitionid,
		SizeID:      &sizeid,
		StateValue:  &stateValue,
		Waiting:     &waiting,
	}

	waitingMachines := metal.Machines{}
	err = s.rs.SearchMachines(&q, &waitingMachines)
	if err != nil {
		return nil, err
	}

	return waitingMachines, nil
}

func (s *Scaler) ShutdownMachines(partitionid, sizeid string) (metal.Machines, error) {
	stateValue := string(metal.ShutdownState)
	q := MachineSearchQuery{
		PartitionID: &partitionid,
		SizeID:      &sizeid,
		StateValue:  &stateValue,
	}

	shutdownMachines := metal.Machines{}
	err := s.rs.SearchMachines(&q, &shutdownMachines)
	if err != nil {
		return nil, err
	}

	return shutdownMachines, nil
}

func (s *Scaler) Shutdown(m *metal.Machine) error {
	state := metal.MachineState{
		Value:       metal.ShutdownState,
		Description: "shut down as exceeding maximum partition poolsize",
	}

	err := s.rs.publishCommandAndUpdate(s.rs.log, m, s.publisher, metal.MachineOnCmd, state)
	if err != nil {
		return err
	}
	return nil
}

func (s *Scaler) PowerOn(m *metal.Machine) error {
	state := metal.MachineState{Value: metal.AvailableState}

	err := s.rs.publishCommandAndUpdate(s.rs.log, m, s.publisher, metal.MachineOnCmd, state)
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

	err = e.PublishMachineCmd(logger, m, publisher, metal.MachineOnCmd)
	if err != nil {
		return err
	}

	return nil
}

func (rs *RethinkStore) adjustNumberOfWaitingMachines(log *zap.SugaredLogger, publisher bus.Publisher, machine *metal.Machine) error {
	p, err := rs.FindPartition(machine.PartitionID)
	if err != nil {
		return err
	}

	manager := &Scaler{
		rs:        rs,
		publisher: publisher,
	}
	scaler := s.NewPoolScaler(log, manager)

	err = scaler.AdjustNumberOfWaitingMachines(machine, p)
	if err != nil {
		return err
	}

	return nil
}
