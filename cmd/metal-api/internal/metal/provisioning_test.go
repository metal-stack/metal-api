package metal

import (
	"testing"

	"git.f-i-ts.de/cloud-native/metallib/zapup"
)

var (
	SuccessfulEventCycle = ProvisioningEvents{
		ProvisioningEvent{
			Event: ProvisioningEventBootingNewKernel,
		},
		ProvisioningEvent{
			Event: ProvisioningEventInstalling,
		},

		ProvisioningEvent{
			Event: ProvisioningEventWaiting,
		},
		ProvisioningEvent{
			Event: ProvisioningEventRegistering,
		},
		ProvisioningEvent{
			Event: ProvisioningEventPreparing,
		},
	}
	CrashEventCycle = ProvisioningEvents{
		ProvisioningEvent{
			Event: ProvisioningEventPreparing,
		},
		ProvisioningEvent{
			Event: ProvisioningEventRegistering,
		},
		ProvisioningEvent{
			Event: ProvisioningEventPreparing,
		},
		ProvisioningEvent{
			Event: ProvisioningEventRegistering,
		},
		ProvisioningEvent{
			Event: ProvisioningEventPreparing,
		},
	}
	CycleWithReset = ProvisioningEvents{
		ProvisioningEvent{
			Event: ProvisioningEventWaiting,
		},
		ProvisioningEvent{
			Event: ProvisioningEventRegistering,
		},
		ProvisioningEvent{
			Event: ProvisioningEventPreparing,
		},
		ProvisioningEvent{
			Event: ProvisioningEventPlannedReboot,
		},
		ProvisioningEvent{
			Event: ProvisioningEventRegistering,
		},
		ProvisioningEvent{
			Event: ProvisioningEventPreparing,
		},
	}
	CycleWithResetAndError = ProvisioningEvents{
		ProvisioningEvent{
			Event: ProvisioningEventPreparing,
		},
		ProvisioningEvent{
			Event: ProvisioningEventInstalling,
		},
		ProvisioningEvent{
			Event: ProvisioningEventWaiting,
		},
		ProvisioningEvent{
			Event: ProvisioningEventRegistering,
		},
		ProvisioningEvent{
			Event: ProvisioningEventPreparing,
		},
		ProvisioningEvent{
			Event: ProvisioningEventPlannedReboot,
		},
		ProvisioningEvent{
			Event: ProvisioningEventRegistering,
		},
		ProvisioningEvent{
			Event: ProvisioningEventPreparing,
		},
	}
	CycleWithResetAndImmediateError = ProvisioningEvents{
		ProvisioningEvent{
			Event: ProvisioningEventPreparing,
		},
		ProvisioningEvent{
			Event: ProvisioningEventInstalling,
		},
		ProvisioningEvent{
			Event: ProvisioningEventWaiting,
		},
		ProvisioningEvent{
			Event: ProvisioningEventRegistering,
		},
		ProvisioningEvent{
			Event: ProvisioningEventPlannedReboot,
		},
		ProvisioningEvent{
			Event: ProvisioningEventRegistering,
		},
		ProvisioningEvent{
			Event: ProvisioningEventPreparing,
		},
	}
)

func TestProvisioning_IncompleteCycles(t *testing.T) {

	tests := []struct {
		name           string
		eventContainer ProvisioningEventContainer
		want           string
	}{
		{
			name: "TestProvisioning_IncompleteCycles Test 1",
			eventContainer: ProvisioningEventContainer{
				Events: SuccessfulEventCycle,
			},
			want: "0",
		},
		{
			name: "TestProvisioning_IncompleteCycles Test 2",
			eventContainer: ProvisioningEventContainer{
				Events: CrashEventCycle,
			},
			want: "2",
		},
		{
			name: "TestProvisioning_IncompleteCycles Test 3",
			eventContainer: ProvisioningEventContainer{
				Events: CycleWithReset,
			},
			want: "0",
		},
		{
			name: "TestProvisioning_IncompleteCycles Test 4",
			eventContainer: ProvisioningEventContainer{
				Events: CycleWithResetAndError,
			},
			want: "1",
		},
		{
			name: "TestProvisioning_IncompleteCycles Test 5",
			eventContainer: ProvisioningEventContainer{
				Events: CycleWithResetAndImmediateError,
			},
			want: "2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.eventContainer.CalculateIncompleteCycles(zapup.MustRootLogger().Sugar()); got != tt.want {
				t.Errorf("CalculateIncompleteCycles() = %v, want %v", got, tt.want)
			}
		})
	}
}
