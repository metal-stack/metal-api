package states

import (
	"context"
	"fmt"

	"github.com/looplab/fsm"
)

type InitialState struct {
	*FSMState
}

func newInitial(c *StateConfig) *InitialState {
	return &InitialState{
		FSMState: &FSMState{
			container: c.Container,
			event:     c.Event,
			log:       c.Log,
		},
	}
}

func (p *InitialState) OnTransition(ctx context.Context, e *fsm.Event) {
	e.Err = fmt.Errorf("unexpected transition back to initial state")
}
