package states

import (
	"time"

	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

// failedMachineReclaimThreshold is the duration after which the machine reclaim is assumed to have failed.
const failedMachineReclaimThreshold = 5 * time.Minute

type PhonedHomeState struct {
	container *metal.ProvisioningEventContainer
	event     *metal.ProvisioningEvent
}

func newPhonedHome(c *StateConfig) *PhonedHomeState {
	return &PhonedHomeState{
		container: c.Container,
		event:     c.Event,
	}
}

func (_ *PhonedHomeState) Name() string {
	return PhonedHome.String()
}

func (p *PhonedHomeState) Handle(e *fsm.Event) {
	switch e.Src {
	case p.Name():
		// swallow on repeated phoned home
		UpdateTimeAndLiveliness(p.event, p.container)
	case MachineReclaim.String():
		// swallow on machine reclaim
		if p.container.LastEventTime != nil && p.event.Time.Sub(*p.container.LastEventTime) > failedMachineReclaimThreshold {
			UpdateTimeAndLiveliness(p.event, p.container)
			p.container.FailedMachineReclaim = true
		}
	default:
		p.container.CrashLoop = false
		appendEventToContainer(p.event, p.container)
	}

}
