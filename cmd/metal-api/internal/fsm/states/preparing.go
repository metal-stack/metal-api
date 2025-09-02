package states

import (
	"context"

	"github.com/looplab/fsm"
)

type PreparingState struct {
	*FSMState
}

func newPreparing(c *StateConfig) *PreparingState {
	return &PreparingState{
		FSMState: &FSMState{
			container: c.Container,
			event:     c.Event,
			log:       c.Log,
		},
	}
}

func (p *PreparingState) OnTransition(ctx context.Context, e *fsm.Event) {
	if p.swallowBufferedPhonedHome(e) {
		return
	}
	p.container.FailedMachineReclaim = false
	appendEventToContainer(p.event, p.container)
}
