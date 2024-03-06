package states

import (
	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

type PXEBootingState struct {
	noopState
	container *metal.ProvisioningEventContainer
	event     *metal.ProvisioningEvent
}

func newPXEBooting(c *StateConfig) *PXEBootingState {
	return &PXEBootingState{
		container: c.Container,
		event:     c.Event,
	}
}

func (p *PXEBootingState) OnEnter(e *fsm.Event) {
	p.container.FailedMachineReclaim = false

	if e.Src == PXEBooting.String() {
		// swallow repeated pxe booting events, which happens regularly
		updateTimeAndLiveliness(p.event, p.container)
		return
	}

	appendEventToContainer(p.event, p.container)
}
