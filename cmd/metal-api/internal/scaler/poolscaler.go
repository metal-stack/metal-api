package scaler

import (
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"go.uber.org/zap"
)

type (
	PoolScaler struct {
		log     *zap.SugaredLogger
		manager MachineManager
	}
	PoolScalerConfig struct {
		Log     *zap.SugaredLogger
		Manager MachineManager
	}
	MachineManager interface {
		AllMachines() (metal.Machines, error)
		WaitingMachines() (metal.Machines, error)
		ShutdownMachines() (metal.Machines, error)

		Shutdown(m *metal.Machine) error
		PowerOn(m *metal.Machine) error
	}
)

func NewPoolScaler(log *zap.SugaredLogger, manager MachineManager) *PoolScaler {
	return &PoolScaler{
		log:     log,
		manager: manager,
	}
}

// AdjustNumberOfWaitingMachines compares the number of waiting machines to the required pool size of the partition and shuts down or powers on machines accordingly
func (p *PoolScaler) AdjustNumberOfWaitingMachines(partition *metal.Partition) error {
	waitingMachines, err := p.manager.WaitingMachines()
	if err != nil {
		return err
	}

	poolSizeExcess := 0
	poolSizeExcess = p.calculatePoolSizeExcess(len(waitingMachines), partition.WaitingPoolMinSize, partition.WaitingPoolMaxSize)

	if poolSizeExcess == 0 {
		return nil
	}

	if poolSizeExcess > 0 {
		p.log.Infow("shut down spare machines", "number", poolSizeExcess, "partition", p)
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
			p.log.Infow("not enough machines to meet waiting pool size; power on all remaining", "number", len(shutdownMachines), "partition", p)
			err = p.powerOnMachines(shutdownMachines, len(shutdownMachines))
			if err != nil {
				return err
			}
			return nil
		}

		p.log.Infow("power on missing machines", "number", poolSizeExcess, "partition", p)
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
func (p *PoolScaler) calculatePoolSizeExcess(actual int, minRequired, maxRequired string) int {
	if strings.Contains(minRequired, "%") != strings.Contains(maxRequired, "%") {
		return 0
	}

	if !strings.Contains(minRequired, "%") && !strings.Contains(maxRequired, "%") {
		min, err := strconv.ParseInt(minRequired, 10, 64)
		if err != nil {
			return 0
		}

		max, err := strconv.ParseInt(maxRequired, 10, 64)
		if err != nil {
			return 0
		}

		if int64(actual) >= min && int64(actual) <= max {
			return 0
		}

		average := int(math.Round((float64(max) + float64(min)) / 2))
		return actual - average
	}

	minString := strings.Split(minRequired, "%")[0]
	minPercent, err := strconv.ParseInt(minString, 10, 64)
	if err != nil {
		return 0
	}

	maxString := strings.Split(maxRequired, "%")[0]
	maxPercent, err := strconv.ParseInt(maxString, 10, 64)
	if err != nil {
		return 0
	}

	allMachines, err := p.manager.AllMachines()
	if err != nil {
		return 0
	}

	min := int(math.Round(float64(len(allMachines)) * float64(minPercent) / 100))
	max := int(math.Round(float64(len(allMachines)) * float64(maxPercent) / 100))

	if actual >= min && actual <= max {
		return 0
	}

	average := int(math.Round((float64(max) + float64(min)) / 2))
	return actual - average
}

func randomIndices(n, k int) []int {
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}

	for i := 0; i < (n - k); i++ {
		rand.Seed(time.Now().UnixNano()) //nolint:gosec
		r := rand.Intn(len(indices))
		indices = append(indices[:r], indices[r+1:]...)
	}

	return indices
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
