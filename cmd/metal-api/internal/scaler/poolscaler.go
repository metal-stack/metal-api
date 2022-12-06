package scaler

import (
	"math"
	"math/rand"
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"go.uber.org/zap"
)

type (
	PoolScaler struct {
		log       *zap.SugaredLogger
		manager   MachineManager
		partition metal.Partition
	}
	PoolScalerConfig struct {
		Log       *zap.SugaredLogger
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
	if p.partition.WaitingPoolRange.IsDisabled() {
		return nil
	}

	waitingMachines, err := p.manager.WaitingMachines()
	if err != nil {
		return err
	}

	var poolSizeExcess int
	poolSizeExcess = p.calculatePoolSizeExcess(len(waitingMachines), p.partition.WaitingPoolRange)

	if poolSizeExcess == 0 {
		return nil
	}

	if poolSizeExcess > 0 {
		p.log.Infow("shut down spare machines", "number", poolSizeExcess)
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
			p.log.Infow("not enough machines to meet waiting pool size; power on all remaining", "number", len(shutdownMachines))
			err = p.powerOnMachines(shutdownMachines, len(shutdownMachines))
			if err != nil {
				return err
			}
			return nil
		}

		p.log.Infow("power on missing machines", "number", poolSizeExcess)
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
func (p *PoolScaler) calculatePoolSizeExcess(actual int, scalerRange metal.ScalerRange) int {
	rangeMin, err, inPercent := scalerRange.GetMin()
	if err != nil {
		return 0
	}

	rangeMax, err, _ := scalerRange.GetMax()
	if err != nil {
		return 0
	}

	min := rangeMin
	max := rangeMax
	average := (float64(rangeMax) + float64(rangeMin)) / 2

	if inPercent {
		allMachines, err := p.manager.AllMachines()
		if err != nil {
			return 0
		}

		min = int64(math.Round(float64(len(allMachines)) * float64(rangeMin) / 100))
		max = int64(math.Round(float64(len(allMachines)) * float64(rangeMax) / 100))
		average = float64(len(allMachines)) * average / 100
	}

	if int64(actual) >= min && int64(actual) <= max {
		return 0
	}

	return actual - int(math.Round(average))
}

func randomIndices(n, k int) []int {
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}

	rand.Seed(time.Now().UnixNano())
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
