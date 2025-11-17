package states

import (
	"context"

	"github.com/looplab/fsm"
)

type InstallingState struct {
	*FSMState
}

func newInstalling(c *StateConfig) *InstallingState {
	return &InstallingState{
		FSMState: &FSMState{
			container: c.Container,
			event:     c.Event,
			log:       c.Log,
		},
	}
}

func (p *InstallingState) OnTransition(ctx context.Context, e *fsm.Event) {
	appendEventToContainer(p.event, p.container)
}
