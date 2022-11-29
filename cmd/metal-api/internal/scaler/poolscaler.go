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
func (p *PoolScaler) AdjustNumberOfWaitingMachines(m *metal.Machine, partition *metal.Partition) error {
	// WaitingPoolSize == 0 means scaling is disabled
	if partition.WaitingPoolSize == "0" {
		return nil
	}

	waitingMachines, err := p.manager.WaitingMachines()
	if err != nil {
		return err
	}

	poolSizeExcess := 0
	if strings.Contains(partition.WaitingPoolSize, "%") {
		poolSizeExcess = p.waitingMachinesMatchPoolSize(len(waitingMachines), partition.WaitingPoolSize, true, partition.ID, m.SizeID)
	} else {
		poolSizeExcess = p.waitingMachinesMatchPoolSize(len(waitingMachines), partition.WaitingPoolSize, false, partition.ID, m.SizeID)
	}

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

func (p *PoolScaler) waitingMachinesMatchPoolSize(actual int, required string, inPercent bool, partitionid, sizeid string) int {
	if !inPercent {
		r, err := strconv.ParseInt(required, 10, 64)
		if err != nil {
			return 0
		}
		return actual - int(r)
	}

	percent := strings.Split(required, "%")[0]
	r, err := strconv.ParseInt(percent, 10, 64)
	if err != nil {
		return 0
	}

	allMachines, err := p.manager.AllMachines()
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
