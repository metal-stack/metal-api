package states

import (
	"github.com/looplab/fsm"
)

type WaitingState struct {
	config *StateConfig
}

func newWaiting(c *StateConfig) *WaitingState {
	return &WaitingState{
		config: c,
	}
}

func (p *WaitingState) OnTransition(e *fsm.Event) {
	appendEventToContainer(p.config.Event, p.config.Container)

	if p.config.Scaler != nil {
		e.Err = p.config.Scaler.AdjustNumberOfWaitingMachines(p.config.Machine, p.config.Partition)
	}
}

func (p *WaitingState) OnLeave(e *fsm.Event) {
	if p.config.Scaler != nil {
		e.Err = p.config.Scaler.AdjustNumberOfWaitingMachines(p.config.Machine, p.config.Partition)
	}
}
