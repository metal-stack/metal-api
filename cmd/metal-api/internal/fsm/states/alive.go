package states

import (
	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"go.uber.org/zap"
)

type AliveState struct {
	log       *zap.SugaredLogger
	container *metal.ProvisioningEventContainer
	event     *metal.ProvisioningEvent
}

func newAlive(c *StateConfig) *AliveState {
	return &AliveState{
		log:       c.Log,
		container: c.Container,
		event:     c.Event,
	}
}

func (p *AliveState) OnTransition(e *fsm.Event) {
	updateTimeAndLiveliness(p.event, p.container)
	p.log.Debugw("received provisioning alive event", "id", p.container.ID)
}
