package states

import (
	"fmt"

	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

type InitialState struct {
	noopState
	container *metal.ProvisioningEventContainer
	event     *metal.ProvisioningEvent
}

func newInitial(c *StateConfig) *InitialState {
	return &InitialState{
		container: c.Container,
		event:     c.Event,
	}
}

func (p *InitialState) OnTransition(e *fsm.Event) {
	e.Err = fmt.Errorf("unexpected transition back to initial state")
}
