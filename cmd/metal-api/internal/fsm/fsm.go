package fsm

import (
	"errors"
	"time"

	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

const timeOutAfterPlannedReboot = time.Minute * 5

var allStates = []string{
	string(metal.ProvisioningEventPXEBooting),
	string(metal.ProvisioningEventPreparing),
	string(metal.ProvisioningEventRegistering),
	string(metal.ProvisioningEventWaiting),
	string(metal.ProvisioningEventInstalling),
	string(metal.ProvisioningEventBootingNewKernel),
	string(metal.ProvisioningEventPhonedHome),
}

var events = []fsm.EventDesc{
	{Name: string(metal.ProvisioningEventPXEBooting), Src: []string{
		string(metal.ProvisioningEventPlannedReboot),
		string(metal.ProvisioningEventPXEBooting),
		string(metal.ProvisioningEventPhonedHome),
	}, Dst: string(metal.ProvisioningEventPXEBooting)},
	{Name: string(metal.ProvisioningEventPreparing), Src: []string{
		string(metal.ProvisioningEventPXEBooting),
		string(metal.ProvisioningEventPlannedReboot),
		string(metal.ProvisioningEventPhonedHome),
	}, Dst: string(metal.ProvisioningEventPreparing)},
	{Name: string(metal.ProvisioningEventRegistering), Src: []string{
		string(metal.ProvisioningEventPreparing),
	}, Dst: string(metal.ProvisioningEventRegistering)},
	{Name: string(metal.ProvisioningEventWaiting), Src: []string{
		string(metal.ProvisioningEventRegistering),
	}, Dst: string(metal.ProvisioningEventWaiting)},
	{Name: string(metal.ProvisioningEventInstalling), Src: []string{
		string(metal.ProvisioningEventWaiting),
	}, Dst: string(metal.ProvisioningEventInstalling)},
	{Name: string(metal.ProvisioningEventBootingNewKernel), Src: []string{
		string(metal.ProvisioningEventInstalling),
	}, Dst: string(metal.ProvisioningEventBootingNewKernel)},
	{Name: string(metal.ProvisioningEventPhonedHome), Src: []string{
		string(metal.ProvisioningEventBootingNewKernel),
		string(metal.ProvisioningEventPhonedHome),
		string(metal.ProvisioningEventPlannedReboot),
	}, Dst: string(metal.ProvisioningEventPhonedHome)},
	{Name: string(metal.ProvisioningEventPlannedReboot), Src: allStates, Dst: string(metal.ProvisioningEventPlannedReboot)},
	{Name: string(metal.ProvisioningEventCrashed), Src: []string{}, Dst: string(metal.ProvisioningEventCrashed)},
}

var callbacks = fsm.Callbacks{
	"before_event": func(e *fsm.Event) {
		if e.Event != string(metal.ProvisioningEventPhonedHome) {
			e.Err = upsertContainer(e.Args)
		}
	},
	"before_" + string(metal.ProvisioningEventPhonedHome): func(e *fsm.Event) {
		if e.FSM.Current() == string(metal.ProvisioningEventPlannedReboot) {
			if err := handlePhonedHomeAfterPlannedReboot(e.Args); err != nil {
				e.Cancel()
			}
		} else if e.FSM.Current() == string(metal.ProvisioningEventPhonedHome) {
			e.Err = updateAliveTimestamp(e.Args)
		} else {
			e.Err = upsertContainer(e.Args)
		}
	},
}

func updateAliveTimestamp(args []interface{}) error {
	// TODO: implement db update

	return nil
}

func handlePhonedHomeAfterPlannedReboot(args []interface{}) error {
	event, eventOk := args[0].(*metal.ProvisioningEvent)
	container, containerOK := args[1].(*metal.ProvisioningEventContainer)
	if !eventOk || !containerOK {
		return errors.New("internal error")
	}

	if event.Time.Sub(container.Events[0].Time) >= timeOutAfterPlannedReboot {
		return errors.New("bogus phoned home event after planned reboot")
	}
	return nil
}

func upsertContainer(args []interface{}) error {
	event, eventOk := args[0].(*metal.ProvisioningEvent)
	container, containerOK := args[1].(*metal.ProvisioningEventContainer)
	ds, dsOk := args[2].(*datastore.RethinkStore)
	if !eventOk || !containerOK || !dsOk {
		return errors.New("internal error")
	}
	err := ds.UpsertProvisioningEventContainer(&metal.ProvisioningEventContainer{
		Base:       metal.Base{ID: container.ID},
		Liveliness: metal.MachineLivelinessAlive,
		Events: []metal.ProvisioningEvent{
			*event,
		},
		LastEventTime:                &event.Time,
		IncompleteProvisioningCycles: "",
	})

	return err
}

// HandleProvisioningEvent writes the given event to the database and returns an error if it is an unexpected event
// in the current state of the machine
func HandleProvisioningEvent(event *metal.ProvisioningEvent, container *metal.ProvisioningEventContainer, ds *datastore.RethinkStore) error {
	currentState := string(metal.ProvisioningEventPXEBooting)
	if len(container.Events) > 0 {
		currentState = string(container.Events[0].Event)
	}

	state := newProvisioningFSM(currentState)
	err := state.Event(string(event.Event), event, container, ds)
	if err != nil {
		if _, ok := err.(fsm.NoTransitionError); !ok {
			return err
		}
	}

	return nil
}

func newProvisioningFSM(initial string) *fsm.FSM {
	provisioningFSM := fsm.NewFSM(
		initial,
		events,
		callbacks,
	)

	return provisioningFSM
}

// func (r machineResource) string(metal.ProvisioningEventForMachine(machineID string, e v1.Machinestring(metal.ProvisioningEvent) (*metal.string(metal.ProvisioningEventContainer, error) {
// 	ec, err := r.ds.Findstring(metal.ProvisioningEventContainer(machineID)
// 	if err != nil && !metal.IsNotFound(err) {
// 		return nil, err
// 	}

// 	if ec == nil {
// 		ec = &metal.string(metal.ProvisioningEventContainer{
// 			Base: metal.Base{
// 				ID: machineID,
// 			},
// 			Liveliness: metal.MachineLivelinessAlive,
// 		}
// 	}
// 	now := time.Now()
// 	ec.LastEventTime = &now

// 	event := metal.string(metal.ProvisioningEvent{
// 		Time:    now,
// 		Event:   metal.string(metal.ProvisioningEventType(e.Event),
// 		Message: e.Message,
// 	}
// 	if event.Event == metal.string(metal.ProvisioningEventAlive {
// 		zapup.MustRootLogger().Sugar().Debugw("received provisioning alive event", "id", ec.ID)
// 		ec.Liveliness = metal.MachineLivelinessAlive
// 	} else if event.Event == metal.string(metal.ProvisioningEventPhonedHome && len(ec.Events) > 0 && ec.Events[0].Event == metal.string(metal.ProvisioningEventPhonedHome {
// 		zapup.MustRootLogger().Sugar().Debugw("swallowing repeated phone home event", "id", ec.ID)
// 		ec.Liveliness = metal.MachineLivelinessAlive
// 	} else {
// 		ec.Events = append([]metal.string(metal.ProvisioningEvent{event}, ec.Events...)
// 		ec.IncompleteProvisioningCycles = ec.CalculateIncompleteCycles(zapup.MustRootLogger().Sugar())
// 		ec.Liveliness = metal.MachineLivelinessAlive
// 	}
// 	ec.TrimEvents(metal.string(metal.ProvisioningEventsInspectionLimit)

// 	err = r.ds.Upsertstring(metal.ProvisioningEventContainer(ec)
// 	return ec, err
// }
