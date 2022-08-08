package states

import (
	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

type WaitingState struct {
	container *metal.ProvisioningEventContainer
	event     *metal.ProvisioningEvent
}

func newWaiting(c *StateConfig) *WaitingState {
	return &WaitingState{
		container: c.Container,
		event:     c.Event,
	}
}

func (p *WaitingState) OnTransition(e *fsm.Event) {
	appendEventToContainer(p.event, p.container)
}
