package scaler

import (
	"testing"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"go.uber.org/zap/zaptest"
)

func TestPoolScaler_AdjustNumberOfWaitingMachines(t *testing.T) {
	tests := []struct {
		name      string
		partition *metal.Partition
		mockFn    func(mock *MockMachineManager)
		wantErr   bool
	}{
		{
			name: "waiting machines match pool size; do nothing",
			partition: &metal.Partition{
				WaitingPoolMinSize: "10",
				WaitingPoolMaxSize: "20",
			},
			mockFn: func(mock *MockMachineManager) {
				mock.On("WaitingMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 10)...), nil)
				mock.On("AllMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 150)...), nil)
			},
			wantErr: false,
		},
		{
			name: "waiting machines match pool size in percent; do nothing",
			partition: &metal.Partition{
				WaitingPoolMinSize: "25%",
				WaitingPoolMaxSize: "30%",
			},
			mockFn: func(mock *MockMachineManager) {
				mock.On("AllMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 150)...), nil)
				mock.On("WaitingMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 45)...), nil)
			},
			wantErr: false,
		},
		{
			name: "more waiting machines needed; power on 8 machines",
			partition: &metal.Partition{
				WaitingPoolMinSize: "15",
				WaitingPoolMaxSize: "20",
			},
			mockFn: func(mock *MockMachineManager) {
				mock.On("AllMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 150)...), nil)
				mock.On("WaitingMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 10)...), nil)
				mock.On("ShutdownMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 10)...), nil)
				mock.On("PowerOn", &metal.Machine{}).Return(nil).Times(8)
			},
			wantErr: false,
		},
		{
			name: "more waiting machines needed in percent; power on 10 machines",
			partition: &metal.Partition{
				WaitingPoolMinSize: "30%",
				WaitingPoolMaxSize: "40%",
			},
			mockFn: func(mock *MockMachineManager) {
				mock.On("AllMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 150)...), nil)
				mock.On("WaitingMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 35)...), nil)
				mock.On("ShutdownMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 20)...), nil)
				mock.On("PowerOn", &metal.Machine{}).Return(nil).Times(18)
			},
			wantErr: false,
		},
		{
			name: "pool size exceeded; power off 7 machines",
			partition: &metal.Partition{
				WaitingPoolMinSize: "10",
				WaitingPoolMaxSize: "15",
			},
			mockFn: func(mock *MockMachineManager) {
				mock.On("WaitingMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 20)...), nil)
				mock.On("AllMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 150)...), nil)
				mock.On("Shutdown", &metal.Machine{}).Return(nil).Times(7)
			},
			wantErr: false,
		},
		{
			name: "pool size exceeded in percent; power off 5 machines",
			partition: &metal.Partition{
				WaitingPoolMinSize: "15%",
				WaitingPoolMaxSize: "20%",
			},
			mockFn: func(mock *MockMachineManager) {
				mock.On("AllMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 150)...), nil)
				mock.On("WaitingMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 31)...), nil)
				mock.On("Shutdown", &metal.Machine{}).Return(nil).Times(4)
			},
			wantErr: false,
		},
		{
			name: "more machines needed than available; power on all remaining",
			partition: &metal.Partition{
				WaitingPoolMinSize: "30",
				WaitingPoolMaxSize: "40",
			},
			mockFn: func(mock *MockMachineManager) {
				mock.On("ShutdownMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 10)...), nil)
				mock.On("WaitingMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 15)...), nil)
				mock.On("AllMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 150)...), nil)
				mock.On("PowerOn", &metal.Machine{}).Return(nil).Times(10)
			},
			wantErr: false,
		},
		{
			name: "no more machines left; do nothing",
			partition: &metal.Partition{
				WaitingPoolMinSize: "15%",
				WaitingPoolMaxSize: "20%",
			},
			mockFn: func(mock *MockMachineManager) {
				mock.On("AllMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 150)...), nil)
				mock.On("ShutdownMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 0)...), nil)
				mock.On("WaitingMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 22)...), nil)
			},
			wantErr: false,
		},
		{
			name:      "pool scaling disabled; do nothing",
			partition: &metal.Partition{},
			wantErr:   false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			manager := NewMockMachineManager(t)

			if tt.mockFn != nil {
				tt.mockFn(manager)
			}

			p := &PoolScaler{
				log:       zaptest.NewLogger(t).Sugar(),
				manager:   manager,
				partition: *tt.partition,
			}
			if err := p.AdjustNumberOfWaitingMachines(); (err != nil) != tt.wantErr {
				t.Errorf("PoolScaler.AdjustNumberOfWaitingMachines() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
