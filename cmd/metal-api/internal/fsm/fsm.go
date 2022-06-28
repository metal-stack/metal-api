package fsm

import (
	"errors"
	"strconv"
	"time"

	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"go.uber.org/zap"
)

type EventContainer struct {
	IncompleteCycles int
	ID               string
	LastEventTime    *time.Time
	Events           metal.ProvisioningEvents
	Liveliness       metal.MachineLiveliness
}

const timeOutAfterPlannedReboot = time.Minute * 5

var events = fsm.Events{
	{
		Name: string(metal.ProvisioningEventPXEBooting),
		Src: []string{
			string(metal.ProvisioningEventPXEBooting),
			string(metal.ProvisioningEventPlannedReboot),
			string(metal.ProvisioningEventPhonedHome),
		},
		Dst: string(metal.ProvisioningEventPXEBooting),
	},
	{
		Name: string(metal.ProvisioningEventPreparing),
		Src: []string{
			string(metal.ProvisioningEventPXEBooting),
			string(metal.ProvisioningEventPlannedReboot),
		},
		Dst: string(metal.ProvisioningEventPreparing),
	},
	{
		Name: string(metal.ProvisioningEventRegistering),
		Src: []string{
			string(metal.ProvisioningEventPreparing),
		},
		Dst: string(metal.ProvisioningEventRegistering),
	},
	{
		Name: string(metal.ProvisioningEventWaiting),
		Src: []string{
			string(metal.ProvisioningEventRegistering),
		},
		Dst: string(metal.ProvisioningEventWaiting),
	},
	{
		Name: string(metal.ProvisioningEventInstalling),
		Src: []string{
			string(metal.ProvisioningEventWaiting),
		},
		Dst: string(metal.ProvisioningEventInstalling),
	},
	{
		Name: string(metal.ProvisioningEventBootingNewKernel),
		Src: []string{
			string(metal.ProvisioningEventInstalling),
		},
		Dst: string(metal.ProvisioningEventBootingNewKernel),
	},
	{
		Name: string(metal.ProvisioningEventPhonedHome),
		Src: []string{
			string(metal.ProvisioningEventBootingNewKernel),
			string(metal.ProvisioningEventPhonedHome),
		},
		Dst: string(metal.ProvisioningEventPhonedHome),
	},
	{
		Name: string(metal.ProvisioningEventPlannedReboot),
		Src: []string{
			string(metal.ProvisioningEventPXEBooting),
			string(metal.ProvisioningEventPreparing),
			string(metal.ProvisioningEventRegistering),
			string(metal.ProvisioningEventWaiting),
			string(metal.ProvisioningEventInstalling),
			string(metal.ProvisioningEventBootingNewKernel),
			string(metal.ProvisioningEventPhonedHome),
			string(metal.ProvisioningEventAlive),
		},
		Dst: string(metal.ProvisioningEventPlannedReboot),
	},
	{
		Name: string(metal.ProvisioningEventAlive),
		Src: []string{
			string(metal.ProvisioningEventPreparing),
			string(metal.ProvisioningEventRegistering),
			string(metal.ProvisioningEventWaiting),
			string(metal.ProvisioningEventInstalling),
			string(metal.ProvisioningEventBootingNewKernel),
		},
		Dst: string(metal.ProvisioningEventAlive),
	},
}

// every callback expects parameters of types *EventContainer, *metal.ProvisioningEvent, *zap.SugaredLogger
var callbacks = fsm.Callbacks{
	"enter_state": handleStateTransition,
	"enter_" + string(metal.ProvisioningEventPlannedReboot): handlePlannedRebootEvent,
	"enter_" + string(metal.ProvisioningEventPhonedHome):    handlePhonedHomeEvent,
	"enter_" + string(metal.ProvisioningEventAlive):         handleAliveEvent,
}

func handleStateTransition(e *fsm.Event) {
	container, event, _, err := parseCallbackArguments(e.Args)
	if err != nil {
		e.Err = err
		return
	}

	if e.Event != string(metal.ProvisioningEventPhonedHome) {
		container.Events = append(container.Events, *event)
		container.LastEventTime = &(*event).Time
		container.Liveliness = metal.MachineLivelinessAlive
	}
}

func handlePlannedRebootEvent(e *fsm.Event) {
	container, _, _, err := parseCallbackArguments(e.Args)
	if err != nil {
		e.Err = err
		return
	}

	container.IncompleteCycles = 0
	container.Liveliness = metal.MachineLivelinessAlive
}

func handlePhonedHomeEvent(e *fsm.Event) {
	container, event, log, err := parseCallbackArguments(e.Args)
	if err != nil {
		e.Err = err
		return
	}

	if e.Src == string(metal.ProvisioningEventPlannedReboot) {
		if event.Time.Sub(*container.LastEventTime) > timeOutAfterPlannedReboot {
			container.Liveliness = metal.MachineLivelinessUnknown
		}
	} else if e.Src != string(metal.ProvisioningEventPhonedHome) {
		container.Events = append(container.Events, *event)
		container.LastEventTime = &(*event).Time
	} else {
		log.Debugw("swallowing repeated phone home event", "id", container.ID)
	}

	container.IncompleteCycles = 0
	container.Liveliness = metal.MachineLivelinessAlive
}

func handleAliveEvent(e *fsm.Event) {
	container, _, log, err := parseCallbackArguments(e.Args)
	if err != nil {
		e.Err = err
		return
	}

	log.Debugw("received provisioning alive event", "id", container.ID)
	container.Liveliness = metal.MachineLivelinessAlive
}

func parseCallbackArguments(args []interface{}) (*EventContainer, *metal.ProvisioningEvent, *zap.SugaredLogger, error) {
	if len(args) != 3 {
		return nil, nil, nil, errors.New("expecting arguments of types *EventContainer, *metal.ProvisioningEvent, *zap.SugaredLogger")
	}

	container, ok := args[0].(*EventContainer)
	if !ok {
		return nil, nil, nil, errors.New("first argument must be of type *EventContainer")
	}

	event, ok := args[1].(*metal.ProvisioningEvent)
	if !ok {
		return nil, nil, nil, errors.New("second argument must be of type *metal.ProvisioningEvent")
	}

	log, ok := args[2].(*zap.SugaredLogger)
	if !ok {
		return nil, nil, nil, errors.New("third argument must be of type *zap.SugaredLogger")
	}

	return container, event, log, nil
}

func newProvisioningFSM(initialState metal.ProvisioningEventType) *fsm.FSM {
	provisioningFSM := fsm.NewFSM(
		string(initialState),
		events,
		callbacks,
	)

	return provisioningFSM
}

// HandleProvisioningEvent writes the ProvisioningEvent to the the ProvisioningEventContainer and checks if the events up to this point
// occured in the expected order
func HandleProvisioningEvent(event *metal.ProvisioningEvent, container *metal.ProvisioningEventContainer, log *zap.SugaredLogger) (*metal.ProvisioningEventContainer, error) {
	provisioningFSM := newProvisioningFSM(container.Events[0].Event)
	eventContainer := EventContainer{
		IncompleteCycles: 0,
		ID:               container.ID,
		Liveliness:       container.Liveliness,
	}

	var err error
	for _, e := range container.Events {
		err = provisioningFSM.Event(string(e.Event), &eventContainer, e, log)

		// where to continue after invalideventerror?
		if err != nil && errors.Is(err, fsm.InvalidEventError{}) {
			eventContainer.IncompleteCycles++
		}
	}

	err = provisioningFSM.Event(string(event.Event), &eventContainer, event, log)
	if err != nil {
		if errors.Is(err, fsm.InvalidEventError{}) {
			eventContainer.IncompleteCycles++
		} else if errors.Is(err, fsm.NoTransitionError{}) {
			err = nil
		}
	}

	container.Events = eventContainer.Events
	container.LastEventTime = eventContainer.LastEventTime
	container.Liveliness = eventContainer.Liveliness
	container.IncompleteProvisioningCycles = strconv.Itoa(eventContainer.IncompleteCycles)

	return container, err
}
