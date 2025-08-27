package states

import (
	"fmt"

	"context"
	"log/slog"

	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/scaler"
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
)

type FSMState interface {
	OnEnter(ctx context.Context, e *fsm.Event)
	OnLeave(ctx context.Context, e *fsm.Event)
}

type stateType string

func (t stateType) String() string {
	return string(t)
}

type StateConfig struct {
	Log       *slog.Logger
	Context   context.Context
	Container *metal.ProvisioningEventContainer
	Event     *metal.ProvisioningEvent
	Scaler    *scaler.PoolScaler
	Machine   *metal.Machine
}

func (c *StateConfig) Validate() error {
	if c.Container == nil {
		return fmt.Errorf("provisioning event container must not be nil")
	}

	if c.Event == nil {
		return fmt.Errorf("provisioning event must not be nil")
	}

	return nil
}

func AllStates(c *StateConfig) map[string]FSMState {
	return map[string]FSMState{
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
