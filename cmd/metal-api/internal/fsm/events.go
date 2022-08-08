package fsm

import (
	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/fsm/states"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

func Events() fsm.Events {
	return fsm.Events{
		{
			Name: metal.ProvisioningEventPXEBooting.String(),
			Src: []string{
				states.MachineReclaim.String(),
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
		},
		{
			Name: metal.ProvisioningEventPreparing.String(),
			Src: []string{
				states.PXEBooting.String(),
				states.MachineReclaim.String(), // MachineReclaim is a valid src for Preparing because some machines might be incapable of sending PXEBoot events
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
		},
		{
			Name: metal.ProvisioningEventPlannedReboot.String(),
			Src:  append(states.AllStateNames(), states.Initial.String()),
			Dst:  states.PlannedReboot.String(),
		},
		{
			Name: metal.ProvisioningEventMachineReclaim.String(),
			Src:  append(states.AllStateNames(), states.Initial.String()),
			Dst:  states.MachineReclaim.String(),
		},
		{
			Name: metal.ProvisioningEventAlive.String(),
			Src: []string{
				states.Preparing.String(),
				states.Registering.String(),
				states.Waiting.String(),
				states.Installing.String(),
				states.BootingNewKernel.String(),
				states.Initial.String(),
			},
		},
	}
}
