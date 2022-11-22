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

	if p.config.AdjustWaitingMachines != nil {
		e.Err = p.config.AdjustWaitingMachines(p.config.Log, p.config.Publisher, p.config.Machine)
	}
}

func (p *WaitingState) OnLeave(e *fsm.Event) {
	if p.config.AdjustWaitingMachines != nil {
		e.Err = p.config.AdjustWaitingMachines(p.config.Log, p.config.Publisher, p.config.Machine)
	}
}
