package states

import (
	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

type PXEBootingState struct {
	container *metal.ProvisioningEventContainer
	event     *metal.ProvisioningEvent
}

func newPXEBooting(c *StateConfig) *PXEBootingState {
	return &PXEBootingState{
		container: c.Container,
		event:     c.Event,
	}
}

func (_ *PXEBootingState) Name() string {
	return PXEBooting.String()
}

func (p *PXEBootingState) Handle(e *fsm.Event) {
	p.container.FailedMachineReclaim = false

	if len(p.container.Events) > 0 && p.container.Events[0].Event == p.event.Event {
		// swallow repeated pxe booting events, which happens regularly
		UpdateTimeAndLiveliness(p.event, p.container)
		return
	}

	appendEventToContainer(p.event, p.container)
}
