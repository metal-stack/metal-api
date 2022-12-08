package states

import (
	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

type BootingNewKernelState struct {
	noopState
	container *metal.ProvisioningEventContainer
	event     *metal.ProvisioningEvent
}

func newBootingNewKernel(c *StateConfig) *BootingNewKernelState {
	return &BootingNewKernelState{
		container: c.Container,
		event:     c.Event,
	}
}

func (p *BootingNewKernelState) OnEnter(e *fsm.Event) {
	appendEventToContainer(p.event, p.container)
}
