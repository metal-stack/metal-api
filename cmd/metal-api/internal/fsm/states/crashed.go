package states

import (
	"context"

	"github.com/looplab/fsm"
)

type CrashState struct {
	*FSMState
}

func newCrash(c *StateConfig) *CrashState {
	return &CrashState{
		FSMState: &FSMState{
			container: c.Container,
			event:     c.Event,
			log:       c.Log,
		},
	}
}

func (p *CrashState) OnTransition(ctx context.Context, e *fsm.Event) {
	p.container.CrashLoop = true
	p.container.LastErrorEvent = p.event
	appendEventToContainer(p.event, p.container)
}
