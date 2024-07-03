package states

import (
	"context"

	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

type BootingNewKernelState struct {
	container *metal.ProvisioningEventContainer
	event     *metal.ProvisioningEvent
}

func newBootingNewKernel(c *StateConfig) *BootingNewKernelState {
	return &BootingNewKernelState{
		container: c.Container,
		event:     c.Event,
	}
}

func (p *BootingNewKernelState) OnTransition(ctx context.Context, e *fsm.Event) {
	appendEventToContainer(p.event, p.container)
}
