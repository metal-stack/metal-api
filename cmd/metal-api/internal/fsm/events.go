package fsm

import (
	"context"

	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/fsm/states"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

const SelfTransitionState = "self_transition"

func Events() fsm.Events {
	return fsm.Events{
		{
			Name: metal.ProvisioningEventPXEBooting.String(),
			Src: []string{
				states.PlannedReboot.String(),
				states.MachineReclaim.String(),
				states.Crashing.String(),
				states.Initial.String(),
			},
			Dst: states.PXEBooting.String(),
		},
		{
			// PXE boot events may occur multiple times in a row
			Name: metal.ProvisioningEventPXEBooting.String(),
			Src: []string{
				states.PXEBooting.String(),
			},
			Dst: SelfTransitionState,
		},
		{
			Name: metal.ProvisioningEventPreparing.String(),
			Src: []string{
				states.PXEBooting.String(),
				states.PlannedReboot.String(),  // can happen because some machine are incapable of sending PXE boot events
				states.MachineReclaim.String(), // can happen because some machine are incapable of sending PXE boot events
				states.Crashing.String(),
				states.Initial.String(),
			},
			Dst: states.Preparing.String(),
		},
		{
			Name: metal.ProvisioningEventRegistering.String(),
			Src: []string{
				states.Preparing.String(),
				states.Initial.String(),
			},
			Dst: states.Registering.String(),
		},
		{
			Name: metal.ProvisioningEventWaiting.String(),
			Src: []string{
				states.Registering.String(),
				states.Initial.String(),
			},
			Dst: states.Waiting.String(),
		},
		{
			Name: metal.ProvisioningEventInstalling.String(),
			Src: []string{
				states.Registering.String(), // wait can be skipped by hammer when allocating machine by specific ID or in reinstall feature
				states.Waiting.String(),
				states.Initial.String(),
			},
			Dst: states.Installing.String(),
		},
		{
			Name: metal.ProvisioningEventBootingNewKernel.String(),
			Src: []string{
				states.Installing.String(),
				states.Initial.String(),
			},
			Dst: states.BootingNewKernel.String(),
		},
		{
			Name: metal.ProvisioningEventPhonedHome.String(),
			Src: []string{
				states.BootingNewKernel.String(),
				states.MachineReclaim.String(),
				states.Initial.String(),
			},
			Dst: states.PhonedHome.String(),
		},
		{
			Name: metal.ProvisioningEventPhonedHome.String(),
			Src: []string{
				states.PlannedReboot.String(),
				states.PhonedHome.String(),
			},
			Dst: SelfTransitionState,
		},
		{
			Name: metal.ProvisioningEventPlannedReboot.String(),
			Src:  states.AllStateNames(),
			Dst:  states.PlannedReboot.String(),
		},
		{
			Name: metal.ProvisioningEventPlannedReboot.String(),
			Src: []string{
				states.PlannedReboot.String(),
			},
			Dst: SelfTransitionState,
		},
		{
			Name: metal.ProvisioningEventMachineReclaim.String(),
			Src:  states.AllStateNames(),
			Dst:  states.MachineReclaim.String(),
		},
		{
			Name: metal.ProvisioningEventMachineReclaim.String(),
			Src: []string{
				states.MachineReclaim.String(),
			},
			Dst: SelfTransitionState,
		},
		{
			Name: metal.ProvisioningEventAlive.String(),
			Src: []string{
				states.PXEBooting.String(),
				states.Preparing.String(),
				states.Registering.String(),
				states.Waiting.String(),
				states.Installing.String(),
				states.BootingNewKernel.String(),
				states.Initial.String(),
			},
			Dst: states.Alive.String(),
		},
		{
			Name: metal.ProvisioningEventCrashed.String(),
			Src: []string{
				states.PXEBooting.String(),
				states.Preparing.String(),
				states.Registering.String(),
				states.Waiting.String(),
				states.Installing.String(),
				states.BootingNewKernel.String(),
				states.Initial.String(),
			},
			Dst: states.Crashing.String(),
		},
	}
}

func eventCallbacks(config *states.StateConfig) fsm.Callbacks {
	allStates := states.AllStates(config)

	callbacks := fsm.Callbacks{
		// unfortunately, the FSM does not trigger the specific state callback when a state transitions to itself
		// therefore we have an artificial state self_transition from which we can trigger the state-specific callback
		"enter_" + SelfTransitionState: func(ctx context.Context, e *fsm.Event) {
			if state, ok := allStates[e.Src]; ok {
				state.OnTransition(ctx, e)
			}
		},
	}

	for name, state := range allStates {
		callbacks["enter_"+name] = state.OnTransition
	}

	return callbacks
}
