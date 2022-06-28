package fsm

import (
	"errors"
	"strconv"
	"time"

	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

type EventContainer struct {
	IncompleteCycles int
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
		},
		Dst: string(metal.ProvisioningEventPlannedReboot),
	},
}

// every callback expects parameters of types *EventContainer, *metal.ProvisioningEvent
var callbacks = fsm.Callbacks{
	"enter_state": handleStateTransition,
	"enter_" + string(metal.ProvisioningEventPlannedReboot): handlePlannedRebootEvent,
	"enter_" + string(metal.ProvisioningEventPhonedHome):    handlePhonedHomeEvent,
}

func handleStateTransition(e *fsm.Event) {
	container, event, err := parseCallbackArguments(e.Args)
	if err != nil {
		e.Err = err
		return
	}

	if e.Event != string(metal.ProvisioningEventPhonedHome) {
		container.Events = append(container.Events, *event)
		container.LastEventTime = &(*event).Time
	}
}

func handlePlannedRebootEvent(e *fsm.Event) {
	container, _, err := parseCallbackArguments(e.Args)
	if err != nil {
		e.Err = err
		return
	}

	container.IncompleteCycles = 0
}

func handlePhonedHomeEvent(e *fsm.Event) {
	container, event, err := parseCallbackArguments(e.Args)
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
	}

	container.IncompleteCycles = 0
}

func parseCallbackArguments(args []interface{}) (*EventContainer, *metal.ProvisioningEvent, error) {
	if len(args) != 2 {
		return nil, nil, errors.New("expecting two arguments of types *EventContainer, *metal.ProvisioningEvent")
	}

	container, ok := args[0].(*EventContainer)
	if !ok {
		return nil, nil, errors.New("first argument must be of type *EventContainer")
	}

	event, ok := args[1].(*metal.ProvisioningEvent)
	if !ok {
		return nil, nil, errors.New("second argument must be of type *metal.ProvisioningEvent")
	}

	return container, event, nil
}

func newProvisioningFSM(initialState metal.ProvisioningEventType) *fsm.FSM {
	provisioningFSM := fsm.NewFSM(
		string(initialState),
		events,
		callbacks,
	)

	return provisioningFSM
}

// HandleProvisioningEvent writes the given event to the database and returns an error if it is an unexpected event
// in the current state of the machine
func HandleProvisioningEvent(event *metal.ProvisioningEvent, container *metal.ProvisioningEventContainer, ds *datastore.RethinkStore) error {
	provisioningFSM := newProvisioningFSM(event.Event)
	eventContainer := EventContainer{
		IncompleteCycles: 0,
		Liveliness:       container.Liveliness,
	}

	var err error
	for _, e := range container.Events {
		err = provisioningFSM.Event(string(e.Event), &eventContainer, e)
		if err != nil {
			eventContainer.IncompleteCycles++
		}
	}

	err = provisioningFSM.Event(string(event.Event), &eventContainer, event)
	if err != nil {
		eventContainer.IncompleteCycles++
	}

	container.Events = eventContainer.Events
	container.LastEventTime = eventContainer.LastEventTime
	container.Liveliness = eventContainer.Liveliness
	container.IncompleteProvisioningCycles = "at least " + strconv.Itoa(eventContainer.IncompleteCycles)

	// TODO: store container in

	return nil
}
