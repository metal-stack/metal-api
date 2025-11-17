package states

import (
	"context"

	"github.com/looplab/fsm"
)

type AliveState struct {
	*FSMState
}

func newAlive(c *StateConfig) *AliveState {
	return &AliveState{
		FSMState: &FSMState{
			container: c.Container,
			event:     c.Event,
			log:       c.Log,
		},
	}
}

func (p *AliveState) OnTransition(ctx context.Context, e *fsm.Event) {
	updateTimeAndLiveliness(p.event, p.container)
	p.log.Debug("received provisioning alive event", "id", p.container.ID)
}
