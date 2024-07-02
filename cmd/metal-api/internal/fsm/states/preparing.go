package states

import (
	"context"

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

func (p *PreparingState) OnTransition(ctx context.Context, e *fsm.Event) {
	p.container.FailedMachineReclaim = false

	appendEventToContainer(p.event, p.container)
}
