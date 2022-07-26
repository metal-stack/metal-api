package fsm

import (
	"errors"
	"fmt"
	"time"

	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"go.uber.org/zap"
)

type fsmStateType string

func (t fsmStateType) String() string {
	return string(t)
}

const (
	fsmStatePXEBooting       fsmStateType = "PXE Booting"
	fsmStatePreparing        fsmStateType = "Preparing"
	fsmStateRegistering      fsmStateType = "Registering"
	fsmStateWaiting          fsmStateType = "Waiting"
	fsmStateInstalling       fsmStateType = "Installing"
	fsmStateBootingNewKernel fsmStateType = "Booting New Kernel"
	fsmStatePhonedHome       fsmStateType = "Phoned Home"
	fsmStatePlannedReboot    fsmStateType = "Planned Reboot"
	fsmStateMachineReclaim   fsmStateType = "Machine Reclaim"
)

// failedMachineReclaimThreshold is the duration after which the machine reclaim is assumed to have failed.
const failedMachineReclaimThreshold = 5 * time.Minute

type provisioningFSM struct {
	fsm       *fsm.FSM
	container *metal.ProvisioningEventContainer
	event     *metal.ProvisioningEvent
	log       *zap.SugaredLogger
}

var Events = fsm.Events{
	{
		Name: metal.ProvisioningEventPXEBooting.String(),
		Src: []string{
			fsmStateMachineReclaim.String(),
		},
		Dst: fsmStatePXEBooting.String(),
	},
	{
		Name: metal.ProvisioningEventPXEBooting.String(),
		Src: []string{
			fsmStatePXEBooting.String(),
		},
	},
	{
		Name: metal.ProvisioningEventPreparing.String(),
		Src: []string{
			fsmStatePXEBooting.String(),
			fsmStateMachineReclaim.String(), // MachineReclaim is a valid src for Preparing because some machines might be incapable of sending PXEBoot events
		},
		Dst: fsmStatePreparing.String(),
	},
	{
		Name: metal.ProvisioningEventRegistering.String(),
		Src: []string{
			fsmStatePreparing.String(),
		},
		Dst: fsmStateRegistering.String(),
	},
	{
		Name: metal.ProvisioningEventWaiting.String(),
		Src: []string{
			fsmStateRegistering.String(),
		},
		Dst: fsmStateWaiting.String(),
	},
	{
		Name: metal.ProvisioningEventInstalling.String(),
		Src: []string{
			fsmStateWaiting.String(),
		},
		Dst: fsmStateInstalling.String(),
	},
	{
		Name: metal.ProvisioningEventBootingNewKernel.String(),
		Src: []string{
			fsmStateInstalling.String(),
		},
		Dst: fsmStateBootingNewKernel.String(),
	},
	{
		Name: metal.ProvisioningEventPhonedHome.String(),
		Src: []string{
			fsmStateBootingNewKernel.String(),
			fsmStateMachineReclaim.String(),
		},
		Dst: fsmStatePhonedHome.String(),
	},
	{
		Name: metal.ProvisioningEventPhonedHome.String(),
		Src: []string{
			fsmStatePlannedReboot.String(),
			fsmStatePhonedHome.String(),
		},
	},
	{
		Name: metal.ProvisioningEventPlannedReboot.String(),
		Src: []string{
			fsmStatePXEBooting.String(),
			fsmStatePreparing.String(),
			fsmStateRegistering.String(),
			fsmStateWaiting.String(),
			fsmStateInstalling.String(),
			fsmStateBootingNewKernel.String(),
			fsmStatePhonedHome.String(),
		},
		Dst: fsmStatePlannedReboot.String(),
	},
	{
		Name: metal.ProvisioningEventMachineReclaim.String(),
		Src: []string{
			fsmStatePXEBooting.String(),
			fsmStatePreparing.String(),
			fsmStateRegistering.String(),
			fsmStateWaiting.String(),
			fsmStateInstalling.String(),
			fsmStateBootingNewKernel.String(),
			fsmStatePhonedHome.String(),
		},
		Dst: fsmStateMachineReclaim.String(),
	},
	{
		Name: metal.ProvisioningEventAlive.String(),
		Src: []string{
			fsmStatePreparing.String(),
			fsmStateRegistering.String(),
			fsmStateWaiting.String(),
			fsmStateInstalling.String(),
			fsmStateBootingNewKernel.String(),
		},
	},
}

// HandleProvisioningEvent can be called to determine whether the given incoming event follows an expected lifecycle of a machine considering the event history of the given provisioning event container.
//
// The function returns a new provisioning event container that can then be persisted in the database. If an error is returned, the incoming event is not supposed to be persisted in the database.
//
// Among other things, this function can detect crash loops or other irregularities within a machine lifecycle and enriches the returned provisioning event container with this information.
func HandleProvisioningEvent(log *zap.SugaredLogger, ec *metal.ProvisioningEventContainer, event *metal.ProvisioningEvent) (*metal.ProvisioningEventContainer, error) {
	if ec == nil {
		return nil, fmt.Errorf("provisioning event container must not be nil")
	}

	if event == nil {
		return nil, fmt.Errorf("provisioning event must not be nil")
	}

	clone := *ec
	container := &clone

	container, err := checkProvisioningEvent(event, container, log)
	if err != nil {
		return nil, fmt.Errorf("internal error while calculating provisioning event container for machine %s", container.ID)
	}

	return container, nil
}

func checkProvisioningEvent(event *metal.ProvisioningEvent, container *metal.ProvisioningEventContainer, log *zap.SugaredLogger) (*metal.ProvisioningEventContainer, error) {
	if len(container.Events) == 0 {
		container.Events = append(container.Events, *event)
		container.LastEventTime = &event.Time
		container.Liveliness = metal.MachineLivelinessAlive

		return container, nil
	}

	provisioningFSM := newProvisioningFSM(container.Events[len(container.Events)-1].Event, container, event, log)

	err := provisioningFSM.fsm.Event(event.Event.String(), provisioningFSM, event)
	if err == nil {
		return container, nil
	}

	if errors.As(err, &fsm.InvalidEventError{}) {
		container.Events = append([]metal.ProvisioningEvent{*event}, container.Events...)
		container.LastEventTime = &event.Time
		container.CrashLoop = true

		return container, nil
	}

	return nil, err
}

func newProvisioningFSM(lastEvent metal.ProvisioningEventType, container *metal.ProvisioningEventContainer, event *metal.ProvisioningEvent, log *zap.SugaredLogger) *provisioningFSM {
	p := provisioningFSM{
		container: container,
		event:     event,
		log:       log,
	}

	p.fsm = fsm.NewFSM(
		getEventDestination(lastEvent.String()),
		Events,
		fsm.Callbacks{
			"enter_" + fsmStateRegistering.String():      p.appendEventToContainer,
			"enter_" + fsmStateWaiting.String():          p.appendEventToContainer,
			"enter_" + fsmStateInstalling.String():       p.appendEventToContainer,
			"enter_" + fsmStateBootingNewKernel.String(): p.appendEventToContainer,
			"enter_" + fsmStateMachineReclaim.String():   p.appendEventToContainer,

			"enter_" + fsmStatePXEBooting.String():    p.resetFailedReclaim,
			"enter_" + fsmStatePreparing.String():     p.resetFailedReclaim,
			"enter_" + fsmStatePlannedReboot.String(): p.resetCrashLoop,
			"enter_" + fsmStatePhonedHome.String():    p.handlePhonedHome,

			"before_event": p.updateEventTimeAndLiveliness,
			"before_" + metal.ProvisioningEventAlive.String():      p.logAliveEvent,
			"before_" + metal.ProvisioningEventPhonedHome.String(): p.checkRepeatedPhonedHome,
		},
	)

	return &p
}

func getEventDestination(event string) string {
	dst := fsmStatePXEBooting.String()

	for _, e := range Events {
		if e.Name == event && e.Dst != "" {
			dst = e.Dst
		}
	}

	return dst
}

func (f *provisioningFSM) appendEventToContainer(e *fsm.Event) {
	f.container.Events = append([]metal.ProvisioningEvent{*f.event}, f.container.Events...)
}

func (f *provisioningFSM) updateEventTimeAndLiveliness(e *fsm.Event) {
	if e.Event == metal.ProvisioningEventPhonedHome.String() && e.Src == fsmStateMachineReclaim.String() {
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
	if e.Src != fsmStateMachineReclaim.String() {
		f.appendEventToContainer(e)
	} else if f.container.LastEventTime != nil && f.event.Time.Sub(*f.container.LastEventTime) > failedMachineReclaimThreshold {
		f.container.LastEventTime = &f.event.Time
		f.container.FailedMachineReclaim = true
	}

	f.container.CrashLoop = false
}

func (f *provisioningFSM) logAliveEvent(e *fsm.Event) {
	f.log.Debugw("received provisioning alive event", "id", f.container.ID)
}

func (f *provisioningFSM) checkRepeatedPhonedHome(e *fsm.Event) {
	if e.Src == fsmStatePhonedHome.String() {
		f.log.Debugw("swallowing repeated phoned home event", "id", f.container.ID)
	}
}
