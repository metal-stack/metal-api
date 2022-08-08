package states

import (
	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"go.uber.org/zap"
)

const (
	Initial          stateType = "Initial"
	PXEBooting       stateType = "PXE Booting"
	Preparing        stateType = "Preparing"
	Registering      stateType = "Registering"
	Waiting          stateType = "Waiting"
	Installing       stateType = "Installing"
	BootingNewKernel stateType = "Booting New Kernel"
	PhonedHome       stateType = "Phoned Home"
	PlannedReboot    stateType = "Planned Reboot"
	MachineReclaim   stateType = "Machine Reclaim"
)

type FSMState interface {
	OnTransition(e *fsm.Event)
}

type stateType string

func (t stateType) String() string {
	return string(t)
}

type StateConfig struct {
	Log       *zap.SugaredLogger
	Container *metal.ProvisioningEventContainer
	Event     *metal.ProvisioningEvent
}

func AllStates(c *StateConfig) map[string]FSMState {
	return map[string]FSMState{
		PXEBooting.String():       newPXEBooting(c),
		Preparing.String():        newPreparing(c),
		Registering.String():      newRegistering(c),
		Waiting.String():          newWaiting(c),
		Installing.String():       newInstalling(c),
		BootingNewKernel.String(): newBootingNewKernel(c),
		PhonedHome.String():       newPhonedHome(c),
		PlannedReboot.String():    newPlannedReboot(c),
		MachineReclaim.String():   newMachineReclaim(c),
	}
}

func AllStateNames() []string {
	var result []string

	for name := range AllStates(&StateConfig{}) {
		result = append(result, name)
	}

	return result
}

func appendEventToContainer(event *metal.ProvisioningEvent, container *metal.ProvisioningEventContainer) {
	UpdateTimeAndLiveliness(event, container)
	container.Events = append([]metal.ProvisioningEvent{*event}, container.Events...)
}

func UpdateTimeAndLiveliness(event *metal.ProvisioningEvent, container *metal.ProvisioningEventContainer) {
	container.LastEventTime = &event.Time
	container.Liveliness = metal.MachineLivelinessAlive
}
