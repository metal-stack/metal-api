package states

import (
	"time"

	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"go.uber.org/zap"
)

// failedMachineReclaimThreshold is the duration after which the machine reclaim is assumed to have failed.
const failedMachineReclaimThreshold = 5 * time.Minute

type PhonedHomeState struct {
	noopState
	log       *zap.SugaredLogger
	container *metal.ProvisioningEventContainer
	event     *metal.ProvisioningEvent
}

func newPhonedHome(c *StateConfig) *PhonedHomeState {
	return &PhonedHomeState{
		log:       c.Log,
		container: c.Container,
		event:     c.Event,
	}
}

func (p *PhonedHomeState) OnEnter(e *fsm.Event) {
	switch e.Src {
	case PhonedHome.String():
		updateTimeAndLiveliness(p.event, p.container)
		p.log.Debugw("swallowing repeated phoned home event", "id", p.container.ID)
	case MachineReclaim.String():
		// swallow on machine reclaim
		if p.container.LastEventTime != nil && p.event.Time.Sub(*p.container.LastEventTime) > failedMachineReclaimThreshold {
			updateTimeAndLiveliness(p.event, p.container)
			p.container.FailedMachineReclaim = true
		}
	default:
		p.container.CrashLoop = false
		appendEventToContainer(p.event, p.container)
	}
}
