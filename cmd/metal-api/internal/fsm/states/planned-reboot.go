package states

import (
	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

type PlannedRebootState struct {
	noopState
	container *metal.ProvisioningEventContainer
	event     *metal.ProvisioningEvent
}

func newPlannedReboot(c *StateConfig) *PlannedRebootState {
	return &PlannedRebootState{
		container: c.Container,
		event:     c.Event,
	}
}

func (p *PlannedRebootState) OnEnter(e *fsm.Event) {
	p.container.CrashLoop = false
	appendEventToContainer(p.event, p.container)
}
