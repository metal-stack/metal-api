package datastore

import (
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-lib/bus"
	"go.uber.org/zap"
)

func (rs *RethinkStore) adjustNumberOfWaitingMachines(log *zap.SugaredLogger, publisher bus.Publisher, machine *metal.Machine) error {
	p, err := rs.FindPartition(machine.PartitionID)
	if err != nil {
		return err
	}

	stateValue := string(metal.AvailableState)
	waiting := true
	q := MachineSearchQuery{
		PartitionID: &p.ID,
		SizeID:      &machine.SizeID,
		StateValue:  &stateValue,
		Waiting:     &waiting,
	}

	waitingMachines := metal.Machines{}
	err = rs.SearchMachines(&q, &waitingMachines)
	if err != nil {
		return err
	}

	poolSizeExcess := 0
	if strings.Contains(p.WaitingPoolSize, "%") {
		poolSizeExcess = rs.waitingMachinesMatchPoolSize(len(waitingMachines), p.WaitingPoolSize, true, machine)
	} else {
		poolSizeExcess = rs.waitingMachinesMatchPoolSize(len(waitingMachines), p.WaitingPoolSize, false, machine)
	}

	if poolSizeExcess == 0 {
		return nil
	}

	if poolSizeExcess > 0 {
		log.Infow("shut down spare machines", "number", poolSizeExcess, "partition", p)
		err := rs.shutdownMachines(log, publisher, waitingMachines, poolSizeExcess)
		if err != nil {
			return err
		}
	} else {
		stateValue = string(metal.ShutdownState)
		q = MachineSearchQuery{
			PartitionID: &p.ID,
			SizeID:      &machine.SizeID,
			StateValue:  &stateValue,
		}

		shutdownMachines := metal.Machines{}
		err = rs.SearchMachines(&q, &shutdownMachines)
		if err != nil {
			return err
		}

		poolSizeExcess = int(math.Abs(float64(poolSizeExcess)))
		if len(shutdownMachines) < poolSizeExcess {
			log.Infow("not enough machines to meet waiting pool size; power on all remaining", "number", len(shutdownMachines), "partition", p)
			err = rs.powerOnMachines(log, publisher, shutdownMachines, len(shutdownMachines))
			if err != nil {
				return err
			}
		}

		log.Infow("power on missing machines", "number", poolSizeExcess, "partition", p)
		err = rs.powerOnMachines(log, publisher, shutdownMachines, poolSizeExcess)
		if err != nil {
			return err
		}
	}

	return nil
}

func (rs *RethinkStore) waitingMachinesMatchPoolSize(actual int, required string, inPercent bool, m *metal.Machine) int {
	if !inPercent {
		r, err := strconv.ParseInt(required, 10, 64)
		if err != nil {
			return 0
		}
		return actual - int(r)
	}

	p := strings.Split(required, "%")[0]
	r, err := strconv.ParseInt(p, 10, 64)
	if err != nil {
		return 0
	}

	q := MachineSearchQuery{
		PartitionID: &m.PartitionID,
		SizeID:      &m.SizeID,
	}

	allMachines := metal.Machines{}
	err = rs.SearchMachines(&q, &allMachines)
	if err != nil {
		return 0
	}

	return actual - int(math.Round(float64(len(allMachines))*float64(r)/100))
}

func randomIndices(n, k int) []int {
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}

	for i := 0; i < (n - k); i++ {
		rand.Seed(time.Now().UnixNano())
		r := rand.Intn(len(indices))
		indices = append(indices[:r], indices[r+1:]...)
	}

	return indices
}

func (rs *RethinkStore) shutdownMachines(logger *zap.SugaredLogger, publisher bus.Publisher, machines metal.Machines, number int) error {
	indices := randomIndices(len(machines), number)
	for i := range indices {
		state := metal.MachineState{
			Value:       metal.ShutdownState,
			Description: "shut down as exceeding maximum partition poolsize",
		}
		idx := indices[i]

		err := rs.machineCommand(logger, &machines[idx], publisher, metal.MachineOffCmd, state)
		if err != nil {
			return err
		}
	}
	return nil
}

func (rs *RethinkStore) powerOnMachines(logger *zap.SugaredLogger, publisher bus.Publisher, machines metal.Machines, number int) error {
	indices := randomIndices(len(machines), number)
	for i := range indices {
		state := metal.MachineState{Value: metal.AvailableState}
		idx := indices[i]

		err := rs.machineCommand(logger, &machines[idx], publisher, metal.MachineOnCmd, state)
		if err != nil {
			return err
		}
	}
	return nil
}

func (rs *RethinkStore) machineCommand(logger *zap.SugaredLogger, m *metal.Machine, publisher bus.Publisher, cmd metal.MachineCommand, state metal.MachineState) error {
	evt := metal.MachineEvent{
		Type: metal.COMMAND,
		Cmd: &metal.MachineExecCommand{
			Command:         cmd,
			TargetMachineID: m.ID,
			IPMI:            &m.IPMI,
		},
	}

	newMachine := *m
	newMachine.State = state
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
