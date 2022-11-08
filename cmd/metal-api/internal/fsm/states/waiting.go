package states

import (
	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

type WaitingState struct {
	container *metal.ProvisioningEventContainer
	event     *metal.ProvisioningEvent
	config    *StateConfig
}

func newWaiting(c *StateConfig) *WaitingState {
	return &WaitingState{
		container: c.Container,
		event:     c.Event,
		config:    c,
	}
}

func (p *WaitingState) OnTransition(e *fsm.Event) {
	appendEventToContainer(p.event, p.container)
	p.config.Message = FSMMessageEnterWaitingState
}

func (p *WaitingState) OnLeave(e *fsm.Event) {
	p.config.Message = FSMMessageLeaveWaitingState
}
