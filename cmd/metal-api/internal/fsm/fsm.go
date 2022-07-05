package fsm

import (
	"errors"
	"reflect"
	"time"

	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"go.uber.org/zap"
)

const timeOutAfterMachineReclaim = time.Minute * 5

type ProvisioningFSM struct {
	fsm       *fsm.FSM
	container *metal.ProvisioningEventContainer
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

// every callback expects arguments of types *ProvisioningFSM, *metal.ProvisioningEvent
var callbacks = fsm.Callbacks{
	"enter_" + metal.ProvisioningEventRegistering.String():      handleStateTransition,
	"enter_" + metal.ProvisioningEventWaiting.String():          handleStateTransition,
	"enter_" + metal.ProvisioningEventInstalling.String():       handleStateTransition,
	"enter_" + metal.ProvisioningEventBootingNewKernel.String(): handleStateTransition,
	"enter_" + metal.ProvisioningEventMachineReclaim.String():   handleStateTransition,

	"enter_" + metal.ProvisioningEventPXEBooting.String():    resetFailedReclaim,
	"enter_" + metal.ProvisioningEventPreparing.String():     resetFailedReclaim,
	"enter_" + metal.ProvisioningEventPlannedReboot.String(): handlePlannedRebootEvent,
	"enter_" + metal.ProvisioningEventPhonedHome.String():    handlePhonedHomeEvent,
}

func handleStateTransition(e *fsm.Event) {
	provisioningFSM, event, err := parseCallbackArguments(e.Args)
	if err != nil {
		e.Err = err
		return
	}

	provisioningFSM.container.Events = append(provisioningFSM.container.Events, *event)
	provisioningFSM.container.LastEventTime = &(*event).Time
	provisioningFSM.container.Liveliness = metal.MachineLivelinessAlive
}

func resetFailedReclaim(e *fsm.Event) {
	provisioningFSM, _, err := parseCallbackArguments(e.Args)
	if err != nil {
		e.Err = err
		return
	}

	provisioningFSM.container.FailedMachineReclaim = false

	handleStateTransition(e)
}

func handlePlannedRebootEvent(e *fsm.Event) {
	provisioningFSM, _, err := parseCallbackArguments(e.Args)
	if err != nil {
		e.Err = err
		return
	}

	provisioningFSM.container.CrashLoop = false

	handleStateTransition(e)
}

func handlePhonedHomeEvent(e *fsm.Event) {
	provisioningFSM, event, err := parseCallbackArguments(e.Args)
	if err != nil {
		e.Err = err
		return
	}

	if e.Src != metal.ProvisioningEventMachineReclaim.String() {
		handleStateTransition(e)
	} else if event.Time.Sub(*provisioningFSM.container.LastEventTime) > timeOutAfterMachineReclaim {
		provisioningFSM.container.LastEventTime = &event.Time
		provisioningFSM.container.FailedMachineReclaim = true
	}

	provisioningFSM.container.CrashLoop = false
}

func parseCallbackArguments(args []interface{}) (*ProvisioningFSM, *metal.ProvisioningEvent, error) {
	if len(args) != 2 {
		return nil, nil, errors.New("expecting two arguments of types *ProvisioningFSM, *metal.ProvisioningEvent")
	}

	provisioningFSM, ok := args[0].(*ProvisioningFSM)
	if !ok {
		return nil, nil, errors.New("first argument must be of type *ProvisioningFSM")
	}

	event, ok := args[1].(*metal.ProvisioningEvent)
	if !ok {
		return nil, nil, errors.New("second argument must be of type *metal.ProvisioningEvent")
	}

	return provisioningFSM, event, nil
}

func newProvisioningFSM(initialState metal.ProvisioningEventType, container *metal.ProvisioningEventContainer) *ProvisioningFSM {
	f := fsm.NewFSM(
		initialState.String(),
		events,
		callbacks,
	)

	provisioningFSM := ProvisioningFSM{
		fsm:       f,
		container: container,
	}

	return &provisioningFSM
}

// HandleProvisioningEvent writes the ProvisioningEvent to the the ProvisioningEventContainer and checks if it is a valid event in the current state
func HandleProvisioningEvent(event *metal.ProvisioningEvent, container *metal.ProvisioningEventContainer, log *zap.SugaredLogger) (*metal.ProvisioningEventContainer, error) {
	if len(container.Events) == 0 {
		container.Events = append(container.Events, *event)
		container.LastEventTime = &(*event).Time
		container.Liveliness = metal.MachineLivelinessAlive

		return container, nil
	}

	provisioningFSM := newProvisioningFSM(container.Events[len(container.Events)-1].Event, container)

	var invalidEventError fsm.InvalidEventError

	err := provisioningFSM.fsm.Event(event.Event.String(), provisioningFSM, event)
	if err != nil {
		if errors.Is(err, fsm.NoTransitionError{}) {
			err = nil
			container.LastEventTime = &(*event).Time
			container.Liveliness = metal.MachineLivelinessAlive
			if event.Event == metal.ProvisioningEventAlive {
				log.Debugw("received provisioning alive event", "id", container.ID)

			} else if event.Event == metal.ProvisioningEventPhonedHome {
				log.Debugw("swallowing repeated phone home event", "id", container.ID)
			}
		} else if reflect.TypeOf(err) == reflect.TypeOf(invalidEventError) {
			container.Events = append(container.Events, *event)
			container.LastEventTime = &(*event).Time
			container.CrashLoop = true
			err = nil
		}
	}

	return container, err
}

func ProvisioningEventForMachine(log *zap.SugaredLogger, ec *metal.ProvisioningEventContainer, machineID, event, message string) *metal.ProvisioningEventContainer {
	if ec == nil {
		ec = &metal.ProvisioningEventContainer{
			Base: metal.Base{
				ID: machineID,
			},
			Liveliness: metal.MachineLivelinessAlive,
		}
	}
	now := time.Now()
	ec.LastEventTime = &now

	ev := metal.ProvisioningEvent{
		Time:    now,
		Event:   metal.ProvisioningEventType(event),
		Message: message,
	}

	ec, err := HandleProvisioningEvent(&ev, ec, log)
	if err != nil {
		log.Errorf("internal error while calculating provisioning event container for machine %v", machineID)
	}

	ec.TrimEvents(metal.ProvisioningEventsInspectionLimit)
	return ec
}
