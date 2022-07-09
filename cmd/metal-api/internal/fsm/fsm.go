package fsm

import (
	"errors"
	"fmt"
	"time"

	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"go.uber.org/zap"
)

const (
	// failedMachineReclaimThreshold is the duration after which the machine reclaim is assumed to have failed.
	failedMachineReclaimThreshold = 5 * time.Minute
	// FIXME define appropriate
	timeOutAfterMachineReclaim = 5 * time.Minute
)

type provisioningFSM struct {
	fsm       *fsm.FSM[metal.ProvisioningEventType, metal.ProvisioningEventType]
	container *metal.ProvisioningEventContainer
	event     *metal.ProvisioningEvent
}

var events = fsm.Transitions[metal.ProvisioningEventType, metal.ProvisioningEventType]{
	{
		Event: metal.ProvisioningEventPXEBooting,
		Src: []metal.ProvisioningEventType{
			metal.ProvisioningEventMachineReclaim,
		},
		Dst: metal.ProvisioningEventPXEBooting,
	},
	{
		Event: metal.ProvisioningEventPXEBooting,
		Src: []metal.ProvisioningEventType{
			metal.ProvisioningEventPXEBooting,
		},
	},
	{
		Event: metal.ProvisioningEventPreparing,
		Src: []metal.ProvisioningEventType{
			metal.ProvisioningEventPXEBooting,
			metal.ProvisioningEventMachineReclaim, // MachineReclaim is a valid src for Preparing because some machines might be incapable of sending PXEBoot events
		},
		Dst: metal.ProvisioningEventPreparing,
	},
	{
		Event: metal.ProvisioningEventRegistering,
		Src: []metal.ProvisioningEventType{
			metal.ProvisioningEventPreparing,
		},
		Dst: metal.ProvisioningEventRegistering,
	},
	{
		Event: metal.ProvisioningEventWaiting,
		Src: []metal.ProvisioningEventType{
			metal.ProvisioningEventRegistering,
		},
		Dst: metal.ProvisioningEventWaiting,
	},
	{
		Event: metal.ProvisioningEventInstalling,
		Src: []metal.ProvisioningEventType{
			metal.ProvisioningEventWaiting,
		},
		Dst: metal.ProvisioningEventInstalling,
	},
	{
		Event: metal.ProvisioningEventBootingNewKernel,
		Src: []metal.ProvisioningEventType{
			metal.ProvisioningEventInstalling,
		},
		Dst: metal.ProvisioningEventBootingNewKernel,
	},
	{
		Event: metal.ProvisioningEventPhonedHome,
		Src: []metal.ProvisioningEventType{
			metal.ProvisioningEventBootingNewKernel,
			metal.ProvisioningEventMachineReclaim,
		},
		Dst: metal.ProvisioningEventPhonedHome,
	},
	{
		Event: metal.ProvisioningEventPhonedHome,
		Src: []metal.ProvisioningEventType{
			metal.ProvisioningEventPlannedReboot,
			metal.ProvisioningEventPhonedHome,
		},
	},
	{
		Event: metal.ProvisioningEventPlannedReboot,
		Src: []metal.ProvisioningEventType{
			metal.ProvisioningEventPXEBooting,
			metal.ProvisioningEventPreparing,
			metal.ProvisioningEventRegistering,
			metal.ProvisioningEventWaiting,
			metal.ProvisioningEventInstalling,
			metal.ProvisioningEventBootingNewKernel,
			metal.ProvisioningEventPhonedHome,
		},
		Dst: metal.ProvisioningEventPlannedReboot,
	},
	{
		Event: metal.ProvisioningEventMachineReclaim,
		Src: []metal.ProvisioningEventType{
			metal.ProvisioningEventPXEBooting,
			metal.ProvisioningEventPreparing,
			metal.ProvisioningEventRegistering,
			metal.ProvisioningEventWaiting,
			metal.ProvisioningEventInstalling,
			metal.ProvisioningEventBootingNewKernel,
			metal.ProvisioningEventPhonedHome,
		},
		Dst: metal.ProvisioningEventMachineReclaim,
	},
	{
		Event: metal.ProvisioningEventAlive,
		Src: []metal.ProvisioningEventType{
			metal.ProvisioningEventPreparing,
			metal.ProvisioningEventRegistering,
			metal.ProvisioningEventWaiting,
			metal.ProvisioningEventInstalling,
			metal.ProvisioningEventBootingNewKernel,
		},
	},
}

func ProvisioningEventForMachine(log *zap.SugaredLogger, ec *metal.ProvisioningEventContainer, event *metal.ProvisioningEvent) (*metal.ProvisioningEventContainer, error) {
	if ec == nil {
		return nil, fmt.Errorf("provisioning event container must not be nil")
	}

	if event == nil {
		return nil, fmt.Errorf("provisioning event must not be nil")
	}

	clone := *ec
	container := &clone

	now := time.Now()
	container.LastEventTime = &now

	container, err := handleProvisioningEvent(event, container)
	if err != nil {
		return nil, fmt.Errorf("internal error while calculating provisioning event container for machine %s", container.ID)
	}

	return container, nil
}

func handleProvisioningEvent(event *metal.ProvisioningEvent, container *metal.ProvisioningEventContainer) (*metal.ProvisioningEventContainer, error) {
	if len(container.Events) == 0 {
		container.Events = append(container.Events, *event)
		container.LastEventTime = &event.Time
		container.Liveliness = metal.MachineLivelinessAlive

		return container, nil
	}

	provisioningFSM := newProvisioningFSM(container.Events[len(container.Events)-1].Event, container, event)

	err := provisioningFSM.fsm.Event(event.Event, provisioningFSM, event)
	if err == nil {
		return container, nil
	}

	if errors.As(err, &fsm.InvalidEventError{}) {
		container.Events = append(container.Events, *event)
		container.LastEventTime = &event.Time
		container.CrashLoop = true

		return container, nil
	}

	return nil, err
}

func newProvisioningFSM(initialState metal.ProvisioningEventType, container *metal.ProvisioningEventContainer, event *metal.ProvisioningEvent) *provisioningFSM {
	p := provisioningFSM{
		container: container,
		event:     event,
	}

	p.fsm = fsm.New(
		initialState,
		events,
		fsm.Callbacks[metal.ProvisioningEventType, metal.ProvisioningEventType]{
			fsm.Callback[metal.ProvisioningEventType, metal.ProvisioningEventType]{When: fsm.EnterState, State: metal.ProvisioningEventRegistering, F: p.appendEventToContainer},
			fsm.Callback[metal.ProvisioningEventType, metal.ProvisioningEventType]{When: fsm.EnterState, State: metal.ProvisioningEventWaiting, F: p.appendEventToContainer},
			fsm.Callback[metal.ProvisioningEventType, metal.ProvisioningEventType]{When: fsm.EnterState, State: metal.ProvisioningEventInstalling, F: p.appendEventToContainer},
			fsm.Callback[metal.ProvisioningEventType, metal.ProvisioningEventType]{When: fsm.EnterState, State: metal.ProvisioningEventBootingNewKernel, F: p.appendEventToContainer},
			fsm.Callback[metal.ProvisioningEventType, metal.ProvisioningEventType]{When: fsm.EnterState, State: metal.ProvisioningEventMachineReclaim, F: p.appendEventToContainer},

			fsm.Callback[metal.ProvisioningEventType, metal.ProvisioningEventType]{When: fsm.EnterState, State: metal.ProvisioningEventPXEBooting, F: p.resetFailedReclaim},
			fsm.Callback[metal.ProvisioningEventType, metal.ProvisioningEventType]{When: fsm.EnterState, State: metal.ProvisioningEventPreparing, F: p.resetFailedReclaim},
			fsm.Callback[metal.ProvisioningEventType, metal.ProvisioningEventType]{When: fsm.EnterState, State: metal.ProvisioningEventPlannedReboot, F: p.resetCrashLoop},
			fsm.Callback[metal.ProvisioningEventType, metal.ProvisioningEventType]{When: fsm.EnterState, State: metal.ProvisioningEventPhonedHome, F: p.handlePhonedHome},
			fsm.Callback[metal.ProvisioningEventType, metal.ProvisioningEventType]{When: fsm.BeforeAllEvents, F: p.updateEventTimeAndLiveliness},
		},
	)

	return &p
}

func (f *provisioningFSM) appendEventToContainer(e *fsm.CallbackContext[metal.ProvisioningEventType, metal.ProvisioningEventType]) {
	f.container.Events = append(f.container.Events, *f.event)
}

func (f *provisioningFSM) updateEventTimeAndLiveliness(e *fsm.CallbackContext[metal.ProvisioningEventType, metal.ProvisioningEventType]) {
	if e.Event == metal.ProvisioningEventPhonedHome && e.Src == metal.ProvisioningEventMachineReclaim {
		return
	}

	f.container.LastEventTime = &f.event.Time
	f.container.Liveliness = metal.MachineLivelinessAlive
}

func (f *provisioningFSM) resetFailedReclaim(e *fsm.CallbackContext[metal.ProvisioningEventType, metal.ProvisioningEventType]) {
	f.container.FailedMachineReclaim = false
	f.appendEventToContainer(e)
}

func (f *provisioningFSM) resetCrashLoop(e *fsm.CallbackContext[metal.ProvisioningEventType, metal.ProvisioningEventType]) {
	f.container.CrashLoop = false
	f.appendEventToContainer(e)
}

func (f *provisioningFSM) handlePhonedHome(e *fsm.CallbackContext[metal.ProvisioningEventType, metal.ProvisioningEventType]) {
	if e.Src != metal.ProvisioningEventMachineReclaim {
		f.appendEventToContainer(e)
	} else if f.container.LastEventTime != nil && f.event.Time.Sub(*f.container.LastEventTime) > timeOutAfterMachineReclaim {
		f.container.LastEventTime = &f.event.Time
		f.container.FailedMachineReclaim = true
	}

	f.container.CrashLoop = false
}
