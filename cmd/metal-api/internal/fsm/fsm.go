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

	clone := *ec
	container := &clone

	initial := states.Initial.String()
	if len(container.Events) != 0 {
		initial = container.Events[0].Event.String()
	}

	var (
		allStates = states.AllStates(&states.StateConfig{Log: log, Event: event, Container: container})
		callbacks = fsm.Callbacks{
			"before_" + metal.ProvisioningEventAlive.String(): func(e *fsm.Event) {
				states.UpdateTimeAndLiveliness(event, container)
				log.Debugw("received provisioning alive event", "id", container.ID)
			},
			// unfortunately, the FSM does not trigger the specific state callback when a state transitions to itself
			// it does trigger the enter_state callback though from which we can trigger a state-specific callback
			"enter_state": func(e *fsm.Event) {
				if e.Dst != "" || e.Event == metal.ProvisioningEventAlive.String() {
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
		getEventDestination(initial),
		Events(),
		callbacks,
	)

	err := provisioningFSM.Event(event.Event.String())
	if err == nil {
		return container, nil
	}

	if errors.As(err, &fsm.InvalidEventError{}) {
		if event.Event.Is(metal.ProvisioningEventAlive.String()) {
			// under no circumstances alive events should be persisted.
			// when this happens, the FSM will always return invalid transition.
			return nil, fmt.Errorf("invalid arrival of alive event for machine %s", container.ID)
		}

		container.Events = append([]metal.ProvisioningEvent{*event}, container.Events...)
		container.LastEventTime = &event.Time
		container.CrashLoop = true

		return container, nil
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
