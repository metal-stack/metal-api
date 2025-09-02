package states

import (
	"context"

	"github.com/looplab/fsm"
)

type PXEBootingState struct {
	*FSMState
}

func newPXEBooting(c *StateConfig) *PXEBootingState {
	return &PXEBootingState{
		FSMState: &FSMState{
			container: c.Container,
			event:     c.Event,
			log:       c.Log,
		},
	}
}

func (p *PXEBootingState) OnTransition(ctx context.Context, e *fsm.Event) {
	if p.swallowBufferedPhonedHome(e) {
		return
	}
	p.container.FailedMachineReclaim = false

	if e.Src == PXEBooting.String() {
		// swallow repeated pxe booting events, which happens regularly
		updateTimeAndLiveliness(p.event, p.container)
		return
	}

	appendEventToContainer(p.event, p.container)
}
