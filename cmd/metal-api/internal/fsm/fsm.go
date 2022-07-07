package fsm

import (
	"errors"
	"fmt"
	"time"

	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"go.uber.org/zap"
)

// failedMachineReclaimThreshold is the duration after which the machine reclaim is assumed to have failed.
const failedMachineReclaimThreshold = 5 * time.Minute

type provisioningFSM struct {
	fsm       *fsm.FSM
	container *metal.ProvisioningEventContainer
	event     *metal.ProvisioningEvent
}

var events = fsm.Events{
	{
		Name: metal.ProvisioningEventPXEBooting.String(),
		Src: []string{
			metal.ProvisioningEventMachineReclaim.String(),
		},
		Dst: metal.ProvisioningEventPXEBooting.String(),
	},
	{
		Name: metal.ProvisioningEventPXEBooting.String(),
		Src: []string{
			metal.ProvisioningEventPXEBooting.String(),
		},
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
			metal.ProvisioningEventMachineReclaim.String(),
		},
		Dst: metal.ProvisioningEventPhonedHome.String(),
	},
	{
		Name: metal.ProvisioningEventPhonedHome.String(),
		Src: []string{
			metal.ProvisioningEventPlannedReboot.String(),
			metal.ProvisioningEventPhonedHome.String(),
		},
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
			metal.ProvisioningEventRegistering.String(),
			metal.ProvisioningEventWaiting.String(),
			metal.ProvisioningEventInstalling.String(),
			metal.ProvisioningEventBootingNewKernel.String(),
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

	err := provisioningFSM.fsm.Event(event.Event.String(), provisioningFSM, event)
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

	p.fsm = fsm.NewFSM(
		initialState.String(),
		events,
		fsm.Callbacks{
			"enter_" + metal.ProvisioningEventRegistering.String():      p.appendEventToContainer,
			"enter_" + metal.ProvisioningEventWaiting.String():          p.appendEventToContainer,
			"enter_" + metal.ProvisioningEventInstalling.String():       p.appendEventToContainer,
			"enter_" + metal.ProvisioningEventBootingNewKernel.String(): p.appendEventToContainer,
			"enter_" + metal.ProvisioningEventMachineReclaim.String():   p.appendEventToContainer,

			"enter_" + metal.ProvisioningEventPXEBooting.String():    p.resetFailedReclaim,
			"enter_" + metal.ProvisioningEventPreparing.String():     p.resetFailedReclaim,
			"enter_" + metal.ProvisioningEventPlannedReboot.String(): p.resetCrashLoop,
			"enter_" + metal.ProvisioningEventPhonedHome.String():    p.handlePhonedHome,
			"before_event": p.updateEventTimeAndLiveliness,
		},
	)

	return &p
}

func (f *provisioningFSM) appendEventToContainer(e *fsm.Event) {
	f.container.Events = append(f.container.Events, *f.event)
}

func (f *provisioningFSM) updateEventTimeAndLiveliness(e *fsm.Event) {
	if e.Event == metal.ProvisioningEventPhonedHome.String() && e.Src == metal.ProvisioningEventMachineReclaim.String() {
		return
	}
	
	f.container.LastEventTime = &f.event.Time
	f.container.Liveliness = metal.MachineLivelinessAlive
}

func (f *provisioningFSM) resetFailedReclaim(e *fsm.Event) {
	f.container.FailedMachineReclaim = false
	f.appendEventToContainer(e)
}

func (f *provisioningFSM) resetCrashLoop(e *fsm.Event) {
	f.container.CrashLoop = false
	f.appendEventToContainer(e)
}

func (f *provisioningFSM) handlePhonedHome(e *fsm.Event) {
	if e.Src != metal.ProvisioningEventMachineReclaim.String() {
		f.appendEventToContainer(e)
	} else if f.container.LastEventTime != nil && f.event.Time.Sub(*f.container.LastEventTime) > timeOutAfterMachineReclaim {
		f.container.LastEventTime = &f.event.Time
		f.container.FailedMachineReclaim = true
	}

	f.container.CrashLoop = false
}
