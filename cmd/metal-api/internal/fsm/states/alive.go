package states

import (
	"context"
	"log/slog"

	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

type AliveState struct {
	log       *slog.Logger
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

func (p *AliveState) OnTransition(ctx context.Context, e *fsm.Event) {
	updateTimeAndLiveliness(p.event, p.container)
	p.log.Debug("received provisioning alive event", "id", p.container.ID)
}
