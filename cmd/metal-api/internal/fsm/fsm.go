package fsm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/fsm/states"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

// HandleProvisioningEvent can be called to determine whether the given incoming event follows an expected lifecycle of a machine considering the event history of the given provisioning event container.
//
// The function returns a new provisioning event container that can then be safely persisted in the database. If an error is returned, the incoming event is not supposed to be persisted in the database.
//
// Among other things, this function can detect crash loops or other irregularities within a machine lifecycle and enriches the returned provisioning event container with this information.
func HandleProvisioningEvent(ctx context.Context, log *slog.Logger, ec *metal.ProvisioningEventContainer, event *metal.ProvisioningEvent) (*metal.ProvisioningEventContainer, error) {
	if ec == nil {
		return nil, fmt.Errorf("provisioning event container must not be nil")
	}

	if event == nil {
		return nil, fmt.Errorf("provisioning event must not be nil")
	}

	var (
		clone     = *ec
		container = &clone
		f         = fsm.NewFSM(
			initialStateFromEventContainer(container),
			Events(),
			eventCallbacks(&states.StateConfig{Log: log, Event: event, Container: container}),
		)
	)

	err := f.Event(ctx, event.Event.String())
	if err == nil {
		return container, nil
	}

	if errors.As(err, &fsm.InvalidEventError{}) {
		if event.Message == "" {
			event.Message = fmt.Sprintf("[unexpectedly received in %s]", strings.ToLower(f.Current()))
		} else {
			event.Message = fmt.Sprintf("[unexpectedly received in %s]: %s", strings.ToLower(f.Current()), event.Message)
		}

		container.LastEventTime = &event.Time
		container.Liveliness = metal.MachineLivelinessAlive
		container.LastErrorEvent = event

		switch e := event.Event; e { //nolint:exhaustive
		case metal.ProvisioningEventPXEBooting, metal.ProvisioningEventPreparing:
			container.CrashLoop = true
			container.Events = append([]metal.ProvisioningEvent{*event}, container.Events...)
		case metal.ProvisioningEventAlive:
			// under no circumstances we want to persists alive in the events container.
			// when this happens the FSM gets stuck in invalid transitions
			// (e.g. all following transitions are invalid and all subsequent alive events will be stored, cramping history).
		default:
			container.Events = append([]metal.ProvisioningEvent{*event}, container.Events...)
		}

		return container, nil
	}

	return nil, fmt.Errorf("internal error while calculating provisioning event container for machine %s: %w", container.ID, err)
}

func initialStateFromEventContainer(container *metal.ProvisioningEventContainer) string {
	lastEvent := ""
	if len(container.Events) != 0 {
		lastEvent = container.Events[0].Event.String()
	}

	return getEventDestination(lastEvent)
}

func getEventDestination(event string) string {
	for _, e := range Events() {
		if e.Name == event && e.Dst != SelfTransitionState {
			return e.Dst
		}
	}

	return states.Initial.String()
}
