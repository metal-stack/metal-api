package fsm

import (
	"errors"
	"fmt"

	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/fsm/states"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"go.uber.org/zap"
)

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

	var (
		clone       = *ec
		container   = &clone
		latestEvent = func() string {
			if len(container.Events) != 0 {
				return container.Events[0].Event.String()
			}
			return ""
		}()
		allStates = states.AllStates(&states.StateConfig{Log: log, Event: event, Container: container})
		callbacks = fsm.Callbacks{
			// unfortunately, the FSM does not trigger the specific state callback when a state transitions to itself
			// it does trigger the enter_state callback though from which we can trigger the state-specific callback
			"enter_state": func(e *fsm.Event) {
				if e.Dst != "" {
					return
				}

				if state, ok := allStates[e.Src]; ok {
					state.OnTransition(e)
				}
			},
		}
	)

	for name, state := range allStates {
		callbacks["enter_"+name] = state.OnTransition
	}

	provisioningFSM := fsm.NewFSM(
		getEventDestination(latestEvent),
		Events(),
		callbacks,
	)

	err := provisioningFSM.Event(event.Event.String())
	if err == nil {
		return container, nil
	}

	if errors.As(err, &fsm.InvalidEventError{}) {
		switch e := event.Event; e { //nolint:exhaustive
		case metal.ProvisioningEventPXEBooting, metal.ProvisioningEventPreparing:
			container.Events = append([]metal.ProvisioningEvent{*event}, container.Events...)
			container.LastEventTime = &event.Time
			container.CrashLoop = true
			return container, nil
		default:
			// we generally decline unexpected events that arrive out of order.
			// this is because when storing these events, it could happen that the FSM gets stuck in invalid transitions
			// (e.g. when an alive event gets stored all following transitions are invalid).
			return nil, fmt.Errorf("declining unexpected event %q for machine %s", e, container.ID)
		}
	}

	return nil, fmt.Errorf("internal error while calculating provisioning event container for machine %s: %w", container.ID, err)
}

func getEventDestination(event string) string {
	for _, e := range Events() {
		if e.Name == event && e.Dst != "" {
			return e.Dst
		}
	}

	return states.Initial.String()
}
