package states

import (
	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

type CrashState struct {
	container *metal.ProvisioningEventContainer
	event     *metal.ProvisioningEvent
}

func newCrash(c *StateConfig) *CrashState {
	return &CrashState{
		container: c.Container,
		event:     c.Event,
	}
}

func (p *CrashState) OnTransition(e *fsm.Event) {
	p.container.CrashLoop = true

	appendEventToContainer(p.event, p.container)
}
