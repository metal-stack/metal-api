package states

import (
	"context"

	"github.com/looplab/fsm"
)

type MachineReclaimState struct {
	*FSMState
}

func newMachineReclaim(c *StateConfig) *MachineReclaimState {
	return &MachineReclaimState{
		FSMState: &FSMState{
			container: c.Container,
			event:     c.Event,
			log:       c.Log,
		},
	}
}

func (p *MachineReclaimState) OnTransition(ctx context.Context, e *fsm.Event) {
	p.container.CrashLoop = false
	appendEventToContainer(p.event, p.container)
}
