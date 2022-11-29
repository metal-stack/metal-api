package scaler

import (
	"testing"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"go.uber.org/zap/zaptest"
)

func TestPoolScaler_waitingMachinesMatchPoolSize(t *testing.T) {
	tests := []struct {
		name        string
		actual      int
		required    string
		inPercent   bool
		partitionid string
		sizeid      string
		mockFn      func(mock *MockMachineManager)
		want        int
	}{
		{
			name:        "less waiting machines than required",
			actual:      10,
			required:    "15",
			inPercent:   false,
			partitionid: "",
			sizeid:      "",
			want:        -5,
		},
		{
			name:        "more waiting machines than required",
			actual:      20,
			required:    "15",
			inPercent:   false,
			partitionid: "",
			sizeid:      "",
			want:        5,
		},
		{
			name:        "less waiting machines than required in percent",
			actual:      20,
			required:    "30%",
			inPercent:   true,
			partitionid: "",
			sizeid:      "",
			mockFn: func(mock *MockMachineManager) {
				mock.On("AllMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 150)...), nil)
			},
			want: -25,
		},
		{
			name:        "more waiting machines than required in percent",
			actual:      24,
			required:    "15%",
			inPercent:   true,
			partitionid: "",
			sizeid:      "",
			mockFn: func(mock *MockMachineManager) {
				mock.On("AllMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 150)...), nil)
			},
			want: 1,
		},
		{
			name:        "waiting machines match pool size",
			actual:      20,
			required:    "20",
			inPercent:   false,
			partitionid: "",
			sizeid:      "",
			want:        0,
		},
		{
			name:        "waiting machines match pool size in percent",
			actual:      30,
			required:    "20%",
			inPercent:   true,
			partitionid: "",
			sizeid:      "",
			mockFn: func(mock *MockMachineManager) {
				mock.On("AllMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 150)...), nil)
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewMockMachineManager(t)

			if tt.mockFn != nil {
				tt.mockFn(manager)
			}

			p := &PoolScaler{
				log:     zaptest.NewLogger(t).Sugar(),
				manager: manager,
			}
			if got := p.waitingMachinesMatchPoolSize(tt.actual, tt.required, tt.inPercent, tt.partitionid, tt.sizeid); got != tt.want {
				t.Errorf("PoolScaler.waitingMachinesMatchPoolSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPoolScaler_AdjustNumberOfWaitingMachines(t *testing.T) {
	tests := []struct {
		name      string
		m         *metal.Machine
		partition *metal.Partition
		mockFn    func(mock *MockMachineManager)
		wantErr   bool
	}{
		{
			name:      "waiting machines match pool size; nothing to do",
			m:         &metal.Machine{SizeID: ""},
			partition: &metal.Partition{WaitingPoolSize: "15"},
			mockFn: func(mock *MockMachineManager) {
				mock.On("WaitingMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 15)...), nil)
			},
			wantErr: false,
		},
		{
			name:      "waiting machines match pool size in percent; nothing to do",
			m:         &metal.Machine{SizeID: ""},
			partition: &metal.Partition{WaitingPoolSize: "30%"},
			mockFn: func(mock *MockMachineManager) {
				mock.On("AllMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 150)...), nil)
				mock.On("WaitingMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 45)...), nil)
			},
			wantErr: false,
		},
		{
			name:      "more waiting machines needed; power on 5 machines",
			m:         &metal.Machine{SizeID: ""},
			partition: &metal.Partition{WaitingPoolSize: "15"},
			mockFn: func(mock *MockMachineManager) {
				mock.On("WaitingMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 10)...), nil)
				mock.On("ShutdownMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 5)...), nil)
				mock.On("PowerOn", &metal.Machine{}).Return(nil).Times(5)
			},
			wantErr: false,
		},
		{
			name:      "more waiting machines needed in percent; power on 10 machines",
			m:         &metal.Machine{SizeID: ""},
			partition: &metal.Partition{WaitingPoolSize: "30%"},
			mockFn: func(mock *MockMachineManager) {
				mock.On("AllMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 150)...), nil)
				mock.On("WaitingMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 35)...), nil)
				mock.On("ShutdownMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 10)...), nil)
				mock.On("PowerOn", &metal.Machine{}).Return(nil).Times(10)
			},
			wantErr: false,
		},
		{
			name:      "pool size exceeded; power off 5 machines",
			m:         &metal.Machine{SizeID: ""},
			partition: &metal.Partition{WaitingPoolSize: "15"},
			mockFn: func(mock *MockMachineManager) {
				mock.On("WaitingMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 20)...), nil)
				mock.On("Shutdown", &metal.Machine{}).Return(nil).Times(5)
			},
			wantErr: false,
		},
		{
			name:      "pool size exceeded in percent; power off 5 machines",
			m:         &metal.Machine{SizeID: ""},
			partition: &metal.Partition{WaitingPoolSize: "15%"},
			mockFn: func(mock *MockMachineManager) {
				mock.On("AllMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 150)...), nil)
				mock.On("WaitingMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 28)...), nil)
				mock.On("Shutdown", &metal.Machine{}).Return(nil).Times(5)
			},
			wantErr: false,
		},
		{
			name:      "more machines needed than available; power on all remaining",
			m:         &metal.Machine{SizeID: ""},
			partition: &metal.Partition{WaitingPoolSize: "30"},
			mockFn: func(mock *MockMachineManager) {
				mock.On("ShutdownMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 10)...), nil)
				mock.On("WaitingMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 15)...), nil)
				mock.On("PowerOn", &metal.Machine{}).Return(nil).Times(10)
			},
			wantErr: false,
		},
		{
			name:      "no more machines left; do nothing",
			m:         &metal.Machine{SizeID: ""},
			partition: &metal.Partition{WaitingPoolSize: "15%"},
			mockFn: func(mock *MockMachineManager) {
				mock.On("AllMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 150)...), nil)
				mock.On("ShutdownMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 0)...), nil)
				mock.On("WaitingMachines").Once().Return(append(metal.Machines{}, make([]metal.Machine, 22)...), nil)
			},
			wantErr: false,
		},
		{
			name:      "pool scaling disabled; do nothing",
			m:         &metal.Machine{SizeID: ""},
			partition: &metal.Partition{WaitingPoolSize: "0"},
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewMockMachineManager(t)

			if tt.mockFn != nil {
				tt.mockFn(manager)
			}

			p := &PoolScaler{
				log:     zaptest.NewLogger(t).Sugar(),
				manager: manager,
			}
			if err := p.AdjustNumberOfWaitingMachines(tt.m, tt.partition); (err != nil) != tt.wantErr {
				t.Errorf("PoolScaler.AdjustNumberOfWaitingMachines() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
