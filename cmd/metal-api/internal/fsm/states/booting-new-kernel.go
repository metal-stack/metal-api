package states

import (
	"context"

	"github.com/looplab/fsm"
)

type BootingNewKernelState struct {
	*FSMState
}

func newBootingNewKernel(c *StateConfig) *BootingNewKernelState {
	return &BootingNewKernelState{
		FSMState: &FSMState{
			container: c.Container,
			event:     c.Event,
			log:       c.Log,
		},
	}
}

func (p *BootingNewKernelState) OnTransition(ctx context.Context, e *fsm.Event) {
	appendEventToContainer(p.event, p.container)
}
