package states

import (
	"context"

	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

type RegisteringState struct {
	container *metal.ProvisioningEventContainer
	event     *metal.ProvisioningEvent
}

func newRegistering(c *StateConfig) *RegisteringState {
	return &RegisteringState{
		container: c.Container,
		event:     c.Event,
	}
}

func (p *RegisteringState) OnTransition(ctx context.Context, e *fsm.Event) {
	appendEventToContainer(p.event, p.container)
}
