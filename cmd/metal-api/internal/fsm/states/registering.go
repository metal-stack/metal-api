package states

import (
	"context"

	"github.com/looplab/fsm"
)

type RegisteringState struct {
	*FSMState
}

func newRegistering(c *StateConfig) *RegisteringState {
	return &RegisteringState{
		FSMState: &FSMState{
			container: c.Container,
			event:     c.Event,
			log:       c.Log,
		},
	}
}

func (p *RegisteringState) OnTransition(ctx context.Context, e *fsm.Event) {
	p.swallowBufferedPhonedHome(e)
	appendEventToContainer(p.event, p.container)
}
