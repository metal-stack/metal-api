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
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.eventContainer.CalculateIncompleteCycles(zapup.MustRootLogger().Sugar()); got != tt.want {
				t.Errorf("CalculateIncompleteCycles() = %v, want %v", got, tt.want)
			}
		})
	}
}

// FSM Tests
var (
	SuccessfulCycle = ProvisioningEvents{
		ProvisioningEvent{
			Event: ProvisioningEventPXEBooting,
		},
		ProvisioningEvent{
			Event: ProvisioningEventPreparing,
		},
		ProvisioningEvent{
			Event: ProvisioningEventRegistering,
		},
		ProvisioningEvent{
			Event: ProvisioningEventWaiting,
		},
		ProvisioningEvent{
			Event: ProvisioningEventInstalling,
		},
		ProvisioningEvent{
			Event: ProvisioningEventBootingNewKernel,
		},
	}
	CrashCycle1 = ProvisioningEvents{
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
	PlannedRebootCycle = ProvisioningEvents{
		ProvisioningEvent{
			Event: ProvisioningEventPreparing,
		},
		ProvisioningEvent{
			Event: ProvisioningEventRegistering,
		},
		ProvisioningEvent{
			Event: ProvisioningEventPlannedReboot,
		},
		ProvisioningEvent{
			Event: ProvisioningEventPreparing,
		},
		ProvisioningEvent{
			Event: ProvisioningEventRegistering,
		},
		ProvisioningEvent{
			Event: ProvisioningEventWaiting,
		},
	}
	PlannedRebootAndError = ProvisioningEvents{
		ProvisioningEvent{
			Event: ProvisioningEventPreparing,
		},
		ProvisioningEvent{
			Event: ProvisioningEventRegistering,
		},
		ProvisioningEvent{
			Event: ProvisioningEventPlannedReboot,
		},
		ProvisioningEvent{
			Event: ProvisioningEventPreparing,
		},
		ProvisioningEvent{
			Event: ProvisioningEventRegistering,
		},
		ProvisioningEvent{
			Event: ProvisioningEventWaiting,
		},
		ProvisioningEvent{
			Event: ProvisioningEventPreparing,
		},
	}
	CrashCycle2 = ProvisioningEvents{
		ProvisioningEvent{
			Event: ProvisioningEventWaiting,
		},
		ProvisioningEvent{
			Event: ProvisioningEventInstalling,
		},
		ProvisioningEvent{
			Event: ProvisioningEventCrashed,
		},
		ProvisioningEvent{
			Event: ProvisioningEventPXEBooting,
		},
	}
	ResetCycle = ProvisioningEvents{
		ProvisioningEvent{
			Event: ProvisioningEventPreparing,
		},
		ProvisioningEvent{
			Event: ProvisioningEventRegistering,
		},
		ProvisioningEvent{
			Event: ProvisioningEventCrashed,
		},
		ProvisioningEvent{
			Event: ProvisioningEventPreparing,
		},
		ProvisioningEvent{
			Event: ProvisioningEventRegistering,
		},
		ProvisioningEvent{
			Event: ProvisioningEventWaiting,
		},
		ProvisioningEvent{
			Event: ProvisioningEventResetFailCount,
		},
	}
	SuccessfulCycleWithBadHistory = ProvisioningEvents{
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
			Event: ProvisioningEventWaiting,
		},
		ProvisioningEvent{
			Event: ProvisioningEventInstalling,
		},
		ProvisioningEvent{
			Event: ProvisioningEventBootingNewKernel,
		},
		ProvisioningEvent{
			Event: ProvisioningEventPhonedHome,
		},
	}
	MultipleTimesPXEBootingIsOK = ProvisioningEvents{
		ProvisioningEvent{
			Event: ProvisioningEventPXEBooting,
		},
		ProvisioningEvent{
			Event: ProvisioningEventPXEBooting,
		},
		ProvisioningEvent{
			Event: ProvisioningEventPreparing,
		},
		ProvisioningEvent{
			Event: ProvisioningEventRegistering,
		},
		ProvisioningEvent{
			Event: ProvisioningEventWaiting,
		},
	}
)

func TestIncompleteCycles(t *testing.T) {
	tests := []struct {
		name           string
		eventContainer ProvisioningEventContainer
		want           string
	}{
		{
			name: "TestFsm_CrashCycles Test 1",
			eventContainer: ProvisioningEventContainer{
				Events: SuccessfulCycle,
			},
			want: "0",
		},
		{
			name: "TestFsm_CrashCycles Test 2",
			eventContainer: ProvisioningEventContainer{
				Events: CrashCycle1,
			},
			want: "2",
		},
		{
			name: "TestFsm_CrashCycles Test 3",
			eventContainer: ProvisioningEventContainer{
				Events: PlannedRebootCycle,
			},
			want: "0",
		},
		{
			name: "TestFsm_CrashCycles Test 4",
			eventContainer: ProvisioningEventContainer{
				Events: PlannedRebootAndError,
			},
			want: "1",
		},
		{
			name: "TestFsm_CrashCycles Test 5",
			eventContainer: ProvisioningEventContainer{
				Events: CrashCycle2,
			},
			want: "1",
		},
		{
			name: "TestFsm_CrashCycles Test 6",
			eventContainer: ProvisioningEventContainer{
				Events: ResetCycle,
			},
			want: "0",
		}, {
			name: "TestFsm_CrashCycles Test 7",
			eventContainer: ProvisioningEventContainer{
				Events: SuccessfulCycleWithBadHistory,
			},
			want: "0",
		},
		{
			name: "TestFsm_CrashCycles Test 8",
			eventContainer: ProvisioningEventContainer{
				Events: MultipleTimesPXEBootingIsOK,
			},
			want: "0",
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.eventContainer.CrashCycles(zapup.MustRootLogger().Sugar()); got != tt.want {
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
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.Is(tt.event); got != tt.want {
				t.Errorf("ProvisioningEventType.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}
