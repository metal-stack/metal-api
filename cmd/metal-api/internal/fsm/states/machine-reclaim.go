package states

import (
	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

type MachineReclaimState struct {
	container *metal.ProvisioningEventContainer
	event     *metal.ProvisioningEvent
}

func newMachineReclaim(c *StateConfig) *MachineReclaimState {
	return &MachineReclaimState{
		container: c.Container,
		event:     c.Event,
	}
}

func (p *MachineReclaimState) OnTransition(e *fsm.Event) {
	p.container.CrashLoop = false
	appendEventToContainer(p.event, p.container)
}
