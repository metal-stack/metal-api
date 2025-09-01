package states

import (
	"context"

	"github.com/looplab/fsm"
)

type WaitingState struct {
	*FSMState
}

func newWaiting(c *StateConfig) *WaitingState {
	return &WaitingState{
		FSMState: &FSMState{
			container: c.Container,
			event:     c.Event,
			log:       c.Log,
		},
	}
}

func (p *WaitingState) OnTransition(ctx context.Context, e *fsm.Event) {
	if p.swallowBufferedPhonedHome(e) {
		return
	}
	appendEventToContainer(p.event, p.container)
}
