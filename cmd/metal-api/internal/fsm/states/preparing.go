package states

import (
	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

type PreparingState struct {
	container *metal.ProvisioningEventContainer
	event     *metal.ProvisioningEvent
}

func newPreparing(c *StateConfig) *PreparingState {
	return &PreparingState{
		container: c.Container,
		event:     c.Event,
	}
}

func (p *PreparingState) OnTransition(e *fsm.Event) {
	p.container.FailedMachineReclaim = false

	appendEventToContainer(p.event, p.container)
}
