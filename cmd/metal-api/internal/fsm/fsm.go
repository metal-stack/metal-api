package fsm

import (
	"fmt"
	"time"

	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"go.uber.org/zap"
)

const timeOutAfterMachineReclaim = time.Minute * 5

type provisioningFSM struct {
	fsm       *fsm.FSM
	container *metal.ProvisioningEventContainer
	event     *metal.ProvisioningEvent
}

var events = fsm.Events{
	{
		Name: metal.ProvisioningEventPXEBooting.String(),
		Src: []string{
			metal.ProvisioningEventPXEBooting.String(),
			metal.ProvisioningEventMachineReclaim.String(),
		},
		Dst: metal.ProvisioningEventPXEBooting.String(),
	},
	{
		Name: metal.ProvisioningEventPreparing.String(),
		Src: []string{
			metal.ProvisioningEventPXEBooting.String(),
			metal.ProvisioningEventMachineReclaim.String(), // MachineReclaim is a valid src for Preparing because some machines might be incapable of sending PXEBoot events
		},
		Dst: metal.ProvisioningEventPreparing.String(),
	},
	{
		Name: metal.ProvisioningEventRegistering.String(),
		Src: []string{
			metal.ProvisioningEventPreparing.String(),
		},
		Dst: metal.ProvisioningEventRegistering.String(),
	},
	{
		Name: metal.ProvisioningEventWaiting.String(),
		Src: []string{
			metal.ProvisioningEventRegistering.String(),
		},
		Dst: metal.ProvisioningEventWaiting.String(),
	},
	{
		Name: metal.ProvisioningEventInstalling.String(),
		Src: []string{
			metal.ProvisioningEventWaiting.String(),
		},
		Dst: metal.ProvisioningEventInstalling.String(),
	},
	{
		Name: metal.ProvisioningEventBootingNewKernel.String(),
		Src: []string{
			metal.ProvisioningEventInstalling.String(),
		},
		Dst: metal.ProvisioningEventBootingNewKernel.String(),
	},
	{
		Name: metal.ProvisioningEventPhonedHome.String(),
		Src: []string{
			metal.ProvisioningEventBootingNewKernel.String(),
			metal.ProvisioningEventPhonedHome.String(),
			metal.ProvisioningEventMachineReclaim.String(),
		},
		Dst: metal.ProvisioningEventPhonedHome.String(),
	},
	{
		Name: metal.ProvisioningEventPhonedHome.String(),
		Src: []string{
			metal.ProvisioningEventPlannedReboot.String(),
		},
		Dst: metal.ProvisioningEventPlannedReboot.String(),
	},
	{
		Name: metal.ProvisioningEventPlannedReboot.String(),
		Src: []string{
			metal.ProvisioningEventPXEBooting.String(),
			metal.ProvisioningEventPreparing.String(),
			metal.ProvisioningEventRegistering.String(),
			metal.ProvisioningEventWaiting.String(),
			metal.ProvisioningEventInstalling.String(),
			metal.ProvisioningEventBootingNewKernel.String(),
			metal.ProvisioningEventPhonedHome.String(),
		},
		Dst: metal.ProvisioningEventPlannedReboot.String(),
	},
	{
		Name: metal.ProvisioningEventMachineReclaim.String(),
		Src: []string{
			metal.ProvisioningEventPXEBooting.String(),
			metal.ProvisioningEventPreparing.String(),
			metal.ProvisioningEventRegistering.String(),
			metal.ProvisioningEventWaiting.String(),
			metal.ProvisioningEventInstalling.String(),
			metal.ProvisioningEventBootingNewKernel.String(),
			metal.ProvisioningEventPhonedHome.String(),
		},
		Dst: metal.ProvisioningEventMachineReclaim.String(),
	},
	{
		Name: metal.ProvisioningEventAlive.String(),
		Src: []string{
			metal.ProvisioningEventPreparing.String(),
		},
		Dst: metal.ProvisioningEventPreparing.String(),
	},
	{
		Name: metal.ProvisioningEventAlive.String(),
		Src: []string{
			metal.ProvisioningEventRegistering.String(),
		},
		Dst: metal.ProvisioningEventRegistering.String(),
	},
	{
		Name: metal.ProvisioningEventAlive.String(),
		Src: []string{
			metal.ProvisioningEventWaiting.String(),
		},
		Dst: metal.ProvisioningEventWaiting.String(),
	},
	{
		Name: metal.ProvisioningEventAlive.String(),
		Src: []string{
			metal.ProvisioningEventInstalling.String(),
		},
		Dst: metal.ProvisioningEventInstalling.String(),
	},
	{
		Name: metal.ProvisioningEventAlive.String(),
		Src: []string{
			metal.ProvisioningEventBootingNewKernel.String(),
		},
		Dst: metal.ProvisioningEventBootingNewKernel.String(),
	},
}

func ProvisioningEventForMachine(log *zap.SugaredLogger, ec *metal.ProvisioningEventContainer, event *metal.ProvisioningEvent) (*metal.ProvisioningEventContainer, error) {
	if ec == nil {
		return nil, fmt.Errorf("provisioning event container must not be nil")
	}

	clone := *ec
	container := &clone

	now := time.Now()
	container.LastEventTime = &now

	if event == nil {
		return nil, fmt.Errorf("provisioning event must not be nil")
	}

	container, err := handleProvisioningEvent(event, container, log)
	if err != nil {
		return nil, fmt.Errorf("internal error while calculating provisioning event container for machine %s", container.ID)
	}

	return container, nil
}

// handleProvisioningEvent writes the ProvisioningEvent to the the ProvisioningEventContainer and checks if it is a valid event in the current state
func handleProvisioningEvent(event *metal.ProvisioningEvent, container *metal.ProvisioningEventContainer, log *zap.SugaredLogger) (*metal.ProvisioningEventContainer, error) {
	if len(container.Events) == 0 {
		container.Events = append(container.Events, *event)
		container.LastEventTime = &event.Time
		container.Liveliness = metal.MachineLivelinessAlive

		return container, nil
	}

	provisioningFSM := newProvisioningFSM(container.Events[len(container.Events)-1].Event, container, event)

	err := provisioningFSM.fsm.Event(event.Event.String(), provisioningFSM, event)
	if err == nil {
		return container, nil
	}

	switch err.(type) {
	case fsm.NoTransitionError:
		container.LastEventTime = &event.Time
		container.Liveliness = metal.MachineLivelinessAlive

		if event.Event == metal.ProvisioningEventAlive {
			log.Debugw("received provisioning alive event", "id", container.ID)
		} else if event.Event == metal.ProvisioningEventPhonedHome {
			log.Debugw("swallowing repeated phone home event", "id", container.ID)
		} else {
			return nil, err
		}

		return container, nil

	case fsm.InvalidEventError:
		container.Events = append(container.Events, *event)
		container.LastEventTime = &event.Time
		container.CrashLoop = true

		return container, nil

	default:
		return nil, err
	}
}

func newProvisioningFSM(initialState metal.ProvisioningEventType, container *metal.ProvisioningEventContainer, event *metal.ProvisioningEvent) *provisioningFSM {
	p := provisioningFSM{
		container: container,
		event:     event,
	}

	p.fsm = fsm.NewFSM(
		initialState.String(),
		events,
		fsm.Callbacks{
			"enter_" + metal.ProvisioningEventRegistering.String():      p.handleStateTransition,
			"enter_" + metal.ProvisioningEventWaiting.String():          p.handleStateTransition,
			"enter_" + metal.ProvisioningEventInstalling.String():       p.handleStateTransition,
			"enter_" + metal.ProvisioningEventBootingNewKernel.String(): p.handleStateTransition,
			"enter_" + metal.ProvisioningEventMachineReclaim.String():   p.handleStateTransition,

			"enter_" + metal.ProvisioningEventPXEBooting.String():    p.resetFailedReclaim,
			"enter_" + metal.ProvisioningEventPreparing.String():     p.resetFailedReclaim,
			"enter_" + metal.ProvisioningEventPlannedReboot.String(): p.handlePlannedRebootEvent,
			"enter_" + metal.ProvisioningEventPhonedHome.String():    p.handlePhonedHomeEvent,
		},
	)

	return &p
}

func (f *provisioningFSM) handleStateTransition(e *fsm.Event) {
	f.container.Events = append(f.container.Events, *f.event)
	f.container.LastEventTime = &f.event.Time
	f.container.Liveliness = metal.MachineLivelinessAlive
}

func (f *provisioningFSM) resetFailedReclaim(e *fsm.Event) {
	f.container.FailedMachineReclaim = false
	f.handleStateTransition(e)
}

func (f *provisioningFSM) handlePlannedRebootEvent(e *fsm.Event) {
	f.container.CrashLoop = false
	f.handleStateTransition(e)
}

func (f *provisioningFSM) handlePhonedHomeEvent(e *fsm.Event) {
	if e.Src != metal.ProvisioningEventMachineReclaim.String() {
		f.handleStateTransition(e)
	} else if f.container.LastEventTime != nil && f.event.Time.Sub(*f.container.LastEventTime) > timeOutAfterMachineReclaim {
		f.container.LastEventTime = &f.event.Time
		f.container.FailedMachineReclaim = true
	}

	f.container.CrashLoop = false
}
