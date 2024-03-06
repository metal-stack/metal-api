package states

import (
	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"go.uber.org/zap"
)

type AliveState struct {
	noopState
	log       *zap.SugaredLogger
	container *metal.ProvisioningEventContainer
	event     *metal.ProvisioningEvent
	machine   *metal.Machine
}

func newAlive(c *StateConfig) *AliveState {
	return &AliveState{
		log:       c.Log,
		container: c.Container,
		event:     c.Event,
		machine:   c.Machine,
	}
}

func (p *AliveState) OnEnter(e *fsm.Event) {
	p.log.Debugw("received provisioning alive event", "id", p.container.ID)

	if p.machine != nil && p.machine.State.Hibernation.Enabled {
		p.container.LastEventTime = &p.event.Time // machine is about to shutdown and is still sending alive events
		return
	}

	updateTimeAndLiveliness(p.event, p.container)
}
