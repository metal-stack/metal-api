package metal

import (
	"testing"

	"github.com/metal-stack/metal-lib/zapup"
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
		ProvisioningEvent{
			Event: ProvisioningEventPXEBooting,
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
	CycleWithPlannedReboot = ProvisioningEvents{
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
	CycleWithPlannedRebootAndError = ProvisioningEvents{
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
	CycleWithPlannedRebootAndImmediateError = ProvisioningEvents{
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
	CycleWithACrash = ProvisioningEvents{
		ProvisioningEvent{
			Event: ProvisioningEventPreparing,
		},
		ProvisioningEvent{
			Event: ProvisioningEventCrashed,
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
			Event: ProvisioningEventResetFailCount,
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
			Event: ProvisioningEventCrashed,
		},
		ProvisioningEvent{
			Event: ProvisioningEventRegistering,
		},
		ProvisioningEvent{
			Event: ProvisioningEventPreparing,
		},
	}
	SuccessfulEventCycleWithBadHistory = ProvisioningEvents{
		ProvisioningEvent{
			Event: ProvisioningEventPhonedHome,
		},
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
		ProvisioningEvent{
			Event: ProvisioningEventRegistering,
		},
	}
	MultipleTimesPXEBootingIsNoIncompleteCycle = ProvisioningEvents{
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
			Event: ProvisioningEventPXEBooting,
		},
		ProvisioningEvent{
			Event: ProvisioningEventPXEBooting,
		},
	}
	SuccessfulReinstallAbortionEventCycle = ProvisioningEvents{
		ProvisioningEvent{
			Event: ProvisioningEventPhonedHome,
		},
		ProvisioningEvent{
			Event: ProvisioningEventReinstallAborted,
		},
		ProvisioningEvent{
			Event: ProvisioningEventPhonedHome,
		},
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
		ProvisioningEvent{
			Event: ProvisioningEventPXEBooting,
		},
	}
	IncompleteReinstallAbortionEventCycle = ProvisioningEvents{
		ProvisioningEvent{
			Event: ProvisioningEventPXEBooting,
		},
		ProvisioningEvent{
			Event: ProvisioningEventReinstallAborted,
		},
		ProvisioningEvent{
			Event: ProvisioningEventPhonedHome,
		},
		ProvisioningEvent{
			Event: ProvisioningEventBootingNewKernel,
		},
		ProvisioningEvent{
			Event: ProvisioningEventInstalling,
		},
		ProvisioningEvent{
			Event: ProvisioningEventWaiting,
		},
	}
	ErroneousReinstallAbortionEventCycle = ProvisioningEvents{
		ProvisioningEvent{
			Event: ProvisioningEventBootingNewKernel,
		},
		ProvisioningEvent{
			Event: ProvisioningEventReinstallAborted,
		},
		ProvisioningEvent{
			Event: ProvisioningEventPhonedHome,
		},
		ProvisioningEvent{
			Event: ProvisioningEventBootingNewKernel,
		},
		ProvisioningEvent{
			Event: ProvisioningEventInstalling,
		},
		ProvisioningEvent{
			Event: ProvisioningEventWaiting,
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
				Events: CycleWithPlannedReboot,
			},
			want: "0",
		},
		{
			name: "TestProvisioning_IncompleteCycles Test 4",
			eventContainer: ProvisioningEventContainer{
				Events: CycleWithPlannedRebootAndError,
			},
			want: "1",
		},
		{
			name: "TestProvisioning_IncompleteCycles Test 5",
			eventContainer: ProvisioningEventContainer{
				Events: CycleWithPlannedRebootAndImmediateError,
			},
			want: "1",
		},
		{
			name: "TestProvisioning_IncompleteCycles Test 6",
			eventContainer: ProvisioningEventContainer{
				Events: CycleWithACrash,
			},
			want: "1",
		},
		{
			name: "TestProvisioning_IncompleteCycles Test 7",
			eventContainer: ProvisioningEventContainer{
				Events: CycleWithReset,
			},
			want: "0",
		},
		{
			name: "TestProvisioning_IncompleteCycles Test 8",
			eventContainer: ProvisioningEventContainer{
				Events: SuccessfulEventCycleWithBadHistory,
			},
			want: "0",
		},
		{
			name: "TestProvisioning_IncompleteCycles Test 9",
			eventContainer: ProvisioningEventContainer{
				Events: MultipleTimesPXEBootingIsNoIncompleteCycle,
			},
			want: "0",
		},
		{
			name: "TestProvisioning_IncompleteCycles Test 10",
			eventContainer: ProvisioningEventContainer{
				Events: SuccessfulReinstallAbortionEventCycle,
			},
			want: "0",
		},
		{
			name: "TestProvisioning_IncompleteCycles Test 11",
			eventContainer: ProvisioningEventContainer{
				Events: IncompleteReinstallAbortionEventCycle,
			},
			want: "1",
		},
		{
			name: "TestProvisioning_IncompleteCycles Test 12",
			eventContainer: ProvisioningEventContainer{
				Events: ErroneousReinstallAbortionEventCycle,
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

func TestProvisioningEventType_Is(t *testing.T) {
	tests := []struct {
		name  string
		event string
		p     ProvisioningEventType
		want  bool
	}{
		{
			name:  "simple",
			event: "Waiting",
			p:     ProvisioningEventWaiting,
			want:  true,
		},
		{
			name:  "simple",
			event: "Waiting",
			p:     ProvisioningEventInstalling,
			want:  false,
		},
		{
			name:  "simple",
			event: "Alive",
			p:     ProvisioningEventAlive,
			want:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.Is(tt.event); got != tt.want {
				t.Errorf("ProvisioningEventType.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}
