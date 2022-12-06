package states

import (
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

func (p *InstallingState) OnTransition(e *fsm.Event) {
	appendEventToContainer(p.event, p.container)
}
