package states

import (
	"context"

	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

type CrashState struct {
	noopState
	container *metal.ProvisioningEventContainer
	event     *metal.ProvisioningEvent
}

func newCrash(c *StateConfig) *CrashState {
	return &CrashState{
		container: c.Container,
		event:     c.Event,
	}
}

func (p *CrashState) OnEnter(ctx context.Context, e *fsm.Event) {
	p.container.CrashLoop = true
	p.container.LastErrorEvent = p.event
	appendEventToContainer(p.event, p.container)
}
