package states

import (
	"context"
	"log/slog"
	"time"

	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

const (
	Initial          stateType = "State Initial"
	Alive            stateType = "State Alive"
	Crashing         stateType = "State Crashing"
	PXEBooting       stateType = "State PXE Booting"
	Preparing        stateType = "State Preparing"
	Registering      stateType = "State Registering"
	Waiting          stateType = "State Waiting"
	Installing       stateType = "State Installing"
	BootingNewKernel stateType = "State Booting New Kernel"
	PhonedHome       stateType = "State Phoned Home"
	PlannedReboot    stateType = "State Planned Reboot"
	MachineReclaim   stateType = "State Machine Reclaim"

	// failedMachineReclaimThreshold is the duration after which the machine reclaim is assumed to have failed.
	failedMachineReclaimThreshold = 5 * time.Minute
	// swallowBufferedPhonedHomeThreshold is the duration after which we don't expect any delayed phoned home events to occur.
	swallowBufferedPhonedHomeThreshold = 5 * time.Minute
)

type IFSMState interface {
	OnTransition(ctx context.Context, e *fsm.Event)
}

type FSMState struct {
	container *metal.ProvisioningEventContainer
	event     *metal.ProvisioningEvent
	log       *slog.Logger
}

type stateType string

func (t stateType) String() string {
	return string(t)
}

type StateConfig struct {
	Log       *slog.Logger
	Container *metal.ProvisioningEventContainer
	Event     *metal.ProvisioningEvent
}

func AllStates(c *StateConfig) map[string]IFSMState {
	return map[string]IFSMState{
		Alive.String():            newAlive(c),
		Crashing.String():         newCrash(c),
		Initial.String():          newInitial(c),
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
	updateTimeAndLiveliness(event, container)
	container.Events = append([]metal.ProvisioningEvent{*event}, container.Events...)
}

func updateTimeAndLiveliness(event *metal.ProvisioningEvent, container *metal.ProvisioningEventContainer) {
	container.LastEventTime = &event.Time
	container.Liveliness = metal.MachineLivelinessAlive
}

func (s *FSMState) swallowBufferedPhonedHome(e *fsm.Event) {
	if e.Event == metal.ProvisioningEventPhonedHome.String() {
		if s.container.LastEventTime != nil && s.event.Time.Sub(*s.container.LastEventTime) < swallowBufferedPhonedHomeThreshold {
			s.log.Debug("swallowing delayed phoned home event", "current event", s.event, "id", s.container.ID)
			return
		}
	}
}
