package states

import (
	"context"
	"log/slog"

	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

type RegisteringState struct {
	container *metal.ProvisioningEventContainer
	event     *metal.ProvisioningEvent
	log       *slog.Logger
}

func newRegistering(c *StateConfig) *RegisteringState {
	return &RegisteringState{
		container: c.Container,
		event:     c.Event,
		log:       c.Log,
	}
}

func (p *RegisteringState) OnTransition(ctx context.Context, e *fsm.Event) {
	if e.Event == metal.ProvisioningEventPhonedHome.String() {
		if p.container.LastEventTime != nil && p.event.Time.Sub(*p.container.LastEventTime) < swallowBufferedPhonedHomeThreshold {
			p.log.Debug("swallowing delayed phoned home event after registering event was already received", "id", p.container.ID)
			return
		}
	}

	appendEventToContainer(p.event, p.container)
}
