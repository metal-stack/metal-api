package states

import (
	"context"

	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

type InstallingState struct {
	noopState
	container *metal.ProvisioningEventContainer
	event     *metal.ProvisioningEvent
}

func newInstalling(c *StateConfig) *InstallingState {
	return &InstallingState{
		container: c.Container,
		event:     c.Event,
	}
}

func (p *InstallingState) OnEnter(ctx context.Context, e *fsm.Event) {
	appendEventToContainer(p.event, p.container)
}
