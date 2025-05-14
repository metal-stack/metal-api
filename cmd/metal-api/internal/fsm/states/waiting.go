package states

import (
	"context"

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

func (p *WaitingState) OnEnter(ctx context.Context, e *fsm.Event) {
	appendEventToContainer(p.config.Event, p.config.Container)

	if p.config.Scaler != nil {
		err := p.config.Scaler.AdjustNumberOfWaitingMachines()
		if err != nil {
			p.config.Log.Error("received error from pool scaler", "error", err)
		}
	}
}

func (p *WaitingState) OnLeave(ctx context.Context, e *fsm.Event) {
	if p.config.Scaler != nil && e.Dst != Alive.String() {
		err := p.config.Scaler.AdjustNumberOfWaitingMachines()
		if err != nil {
			p.config.Log.Error("received error from pool scaler", "error", err)
		}
	}
}
