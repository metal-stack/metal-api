package states

import (
	"context"

	"github.com/looplab/fsm"
)

type PlannedRebootState struct {
	*FSMState
}

func newPlannedReboot(c *StateConfig) *PlannedRebootState {
	return &PlannedRebootState{
		FSMState: &FSMState{
			container: c.Container,
			event:     c.Event,
			log:       c.Log,
		},
	}
}

func (p *PlannedRebootState) OnTransition(ctx context.Context, e *fsm.Event) {
	p.container.CrashLoop = false
	appendEventToContainer(p.event, p.container)
}
