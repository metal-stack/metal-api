package states

import (
	"context"
	"log/slog"

	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

type PXEBootingState struct {
	container *metal.ProvisioningEventContainer
	event     *metal.ProvisioningEvent
	log       *slog.Logger
}

func newPXEBooting(c *StateConfig) *PXEBootingState {
	return &PXEBootingState{
		container: c.Container,
		event:     c.Event,
		log:       c.Log,
	}
}

func (p *PXEBootingState) OnTransition(ctx context.Context, e *fsm.Event) {
	if e.Event == metal.ProvisioningEventPhonedHome.String() {
		if p.container.LastEventTime != nil && p.event.Time.Sub(*p.container.LastEventTime) < swallowBufferedPhonedHomeThreshold {
			p.log.Debug("swallowing delayed phoned home event after pxe booting event was already received", "id", p.container.ID)
			return
		}
	}

	p.container.FailedMachineReclaim = false

	if e.Src == PXEBooting.String() {
		// swallow repeated pxe booting events, which happens regularly
		updateTimeAndLiveliness(p.event, p.container)
		return
	}

	appendEventToContainer(p.event, p.container)
}
