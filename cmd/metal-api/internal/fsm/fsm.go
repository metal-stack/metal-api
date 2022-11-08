package fsm

import (
	"errors"
	"fmt"
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
func HandleProvisioningEvent(c *states.StateConfig) (*metal.ProvisioningEventContainer, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	var (
		f = fsm.NewFSM(
			initialStateFromEventContainer(c.Container),
			Events(),
			eventCallbacks(c),
		)
	)

	err := f.Event(c.Event.Event.String())
	if err == nil {
		return c.Container, nil
	}

	if errors.As(err, &fsm.InvalidEventError{}) {
		if c.Event.Message == "" {
			c.Event.Message = fmt.Sprintf("[unexpectedly received in %s]", strings.ToLower(f.Current()))
		} else {
			c.Event.Message = fmt.Sprintf("[unexpectedly received in %s]: %s", strings.ToLower(f.Current()), c.Event.Message)
		}

		c.Container.LastEventTime = &c.Event.Time
		c.Container.Liveliness = metal.MachineLivelinessAlive
		c.Container.LastErrorEvent = c.Event

		switch e := c.Event.Event; e { //nolint:exhaustive
		case metal.ProvisioningEventPXEBooting, metal.ProvisioningEventPreparing:
			c.Container.CrashLoop = true
			c.Container.Events = append([]metal.ProvisioningEvent{*c.Event}, c.Container.Events...)
		case metal.ProvisioningEventAlive:
			// under no circumstances we want to persists alive in the events container.
			// when this happens the FSM gets stuck in invalid transitions
			// (e.g. all following transitions are invalid and all subsequent alive events will be stored, cramping history).
		default:
			c.Container.Events = append([]metal.ProvisioningEvent{*c.Event}, c.Container.Events...)
		}

		return c.Container, nil
	}

	return nil, fmt.Errorf("internal error while calculating provisioning event container for machine %s: %w", c.Container.ID, err)
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
