package states

import (
	"context"
	"log/slog"

	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

type PreparingState struct {
	container *metal.ProvisioningEventContainer
	event     *metal.ProvisioningEvent
	log       *slog.Logger
}

func newPreparing(c *StateConfig) *PreparingState {
	return &PreparingState{
		container: c.Container,
		event:     c.Event,
		log:       c.Log,
	}
}

func (p *PreparingState) OnTransition(ctx context.Context, e *fsm.Event) {
	if e.Event == metal.ProvisioningEventPhonedHome.String() {
		if p.container.LastEventTime != nil && p.event.Time.Sub(*p.container.LastEventTime) < swallowBufferedPhonedHomeThreshold {
			p.log.Debug("swallowing delayed phoned home event after preparing event was already received", "id", p.container.ID)
			return
		}
	}

	p.container.FailedMachineReclaim = false

	appendEventToContainer(p.event, p.container)
}
