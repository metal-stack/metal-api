package scaler

import (
	"log/slog"
	"math"
	"math/rand"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

type (
	PoolScaler struct {
		log       *slog.Logger
		manager   MachineManager
		partition metal.Partition
	}
	PoolScalerConfig struct {
		Log       *slog.Logger
		Manager   MachineManager
		Partition metal.Partition
	}
	MachineManager interface {
		AllMachines() (metal.Machines, error)
		WaitingMachines() (metal.Machines, error)
		ShutdownMachines() (metal.Machines, error)

		Shutdown(m *metal.Machine) error
		PowerOn(m *metal.Machine) error
	}
)

func NewPoolScaler(c *PoolScalerConfig) *PoolScaler {
	return &PoolScaler{
		log:       c.Log.With("partition", c.Partition.ID),
		manager:   c.Manager,
		partition: c.Partition,
	}
}

// AdjustNumberOfWaitingMachines compares the number of waiting machines to the required pool size of the partition and shuts down or powers on machines accordingly
func (p *PoolScaler) AdjustNumberOfWaitingMachines() error {
	waitingPoolRange, err := metal.NewScalerRange(p.partition.WaitingPoolMinSize, p.partition.WaitingPoolMaxSize)
	if err != nil {
		return err
	}

	if waitingPoolRange.IsDisabled() {
		shutdownMachines, err := p.manager.ShutdownMachines()
		if err != nil {
			return err
		}

		if len(shutdownMachines) < 1 {
			return nil
		}

		p.log.Info("power on all shutdown machines as automatic pool size scaling is disabled", "number of machines to power on", len(shutdownMachines))
		err = p.powerOnMachines(shutdownMachines, len(shutdownMachines))
		if err != nil {
			return err
		}

		return nil
	}

	waitingMachines, err := p.manager.WaitingMachines()
	if err != nil {
		return err
	}

	var poolSizeExcess int
	poolSizeExcess = p.calculatePoolSizeExcess(len(waitingMachines), *waitingPoolRange)

	if poolSizeExcess == 0 {
		p.log.Info("pool size condition met; doing nothing")
		return nil
	}

	if poolSizeExcess > 0 {
		p.log.Info("shut down spare machines", "number", poolSizeExcess)
		err := p.shutdownMachines(waitingMachines, poolSizeExcess)
		if err != nil {
			return err
		}
	} else {
		shutdownMachines, err := p.manager.ShutdownMachines()
		if err != nil {
			return err
		}

		poolSizeExcess = int(math.Abs(float64(poolSizeExcess)))
		if len(shutdownMachines) < poolSizeExcess {
			p.log.Info("not enough machines to meet waiting pool size; power on all remaining", "number", len(shutdownMachines))
			err = p.powerOnMachines(shutdownMachines, len(shutdownMachines))
			if err != nil {
				return err
			}
			return nil
		}

		p.log.Info("power on missing machines", "number", poolSizeExcess)
		err = p.powerOnMachines(shutdownMachines, poolSizeExcess)
		if err != nil {
			return err
		}
	}

	return nil
}

// calculatePoolSizeExcess checks if there are less waiting machines than minRequired or more than maxRequired
// if yes, it returns the difference between the actual amount of waiting machines and the average of minRequired and maxRequired
// if no, it returns 0
func (p *PoolScaler) calculatePoolSizeExcess(current int, scalerRange metal.ScalerRange) int {
	allMachines, err := p.manager.AllMachines()
	if err != nil {
		return 0
	}

	min := scalerRange.Min(len(allMachines))
	max := scalerRange.Max(len(allMachines))
	average := (float64(max) + float64(min)) / 2

	p.log.Info("checking pool size condition", "minimum pool size", min, "maximum pool size", max, "current pool size", current)

	if current >= min && current <= max {
		return 0
	}

	return current - int(math.Round(average))
}

func randomIndices(n, k int) []int {
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}

	rand.Shuffle(len(indices), func(i, j int) { indices[i], indices[j] = indices[j], indices[i] })

	return indices[:k]
}

func (p *PoolScaler) shutdownMachines(machines metal.Machines, number int) error {
	indices := randomIndices(len(machines), number)
	for i := range indices {
		idx := indices[i]
		err := p.manager.Shutdown(&machines[idx])
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *PoolScaler) powerOnMachines(machines metal.Machines, number int) error {
	indices := randomIndices(len(machines), number)
	for i := range indices {
		idx := indices[i]
		err := p.manager.PowerOn(&machines[idx])
		if err != nil {
			return err
		}
	}
	return nil
}
