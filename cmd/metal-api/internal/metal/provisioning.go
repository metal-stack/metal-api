package metal

import (
	"strconv"
	"time"

	"github.com/looplab/fsm"
	"go.uber.org/zap"
)

// FSM states
const (
	FsmStateInitial          string = "Initial State"
	FsmStatePXEBooting       string = "PXE Booting state"
	FsmStatePreparing        string = "Rreparing state"
	FsmStateRegistering      string = "Registering state"
	FsmStateWaiting          string = "Waiting state"
	FsmStateInstalling       string = "Installing state"
	FsmStateBootingNewKernel string = "Booting New Kernel state"
	FsmStateProvisioned      string = "Provisioned state"
	FsmStatePlannedReboot    string = "Planned Reboot state"
	FsmStateCrashed          string = "Crashed state"
	FsmStateAlive            string = "Alive state"
	FsmStateResestFailCount  string = "Reset Fail Count state"
)

var allStates = []string{
	FsmStateInitial,
	FsmStatePXEBooting,
	FsmStatePreparing,
	FsmStateRegistering,
	FsmStateWaiting,
	FsmStateInstalling,
	FsmStateBootingNewKernel,
	FsmStateProvisioned,
	FsmStatePlannedReboot,
	FsmStateCrashed,
	FsmStateAlive,
	FsmStateResestFailCount,
}

var events = []fsm.EventDesc{
	{Name: string(ProvisioningEventPXEBooting), Src: []string{FsmStateInitial, FsmStatePlannedReboot, FsmStatePXEBooting, FsmStateCrashed, FsmStateProvisioned, FsmStateResestFailCount}, Dst: FsmStatePXEBooting},
	{Name: string(ProvisioningEventPreparing), Src: []string{FsmStatePXEBooting, FsmStateInitial, FsmStatePlannedReboot, FsmStateCrashed, FsmStateProvisioned, FsmStateResestFailCount}, Dst: FsmStatePreparing},
	{Name: string(ProvisioningEventRegistering), Src: []string{FsmStatePreparing, FsmStateInitial, FsmStateResestFailCount}, Dst: FsmStateRegistering},
	{Name: string(ProvisioningEventWaiting), Src: []string{FsmStateRegistering, FsmStateInitial, FsmStateResestFailCount}, Dst: FsmStateWaiting},
	{Name: string(ProvisioningEventInstalling), Src: []string{FsmStateWaiting, FsmStateInitial, FsmStateResestFailCount}, Dst: FsmStateInstalling},
	{Name: string(ProvisioningEventBootingNewKernel), Src: []string{FsmStateInstalling, FsmStateInitial, FsmStateResestFailCount}, Dst: FsmStateBootingNewKernel},
	{Name: string(ProvisioningEventPhonedHome), Src: []string{FsmStateBootingNewKernel, FsmStateProvisioned, FsmStateInitial, FsmStateResestFailCount}, Dst: FsmStateProvisioned},
	{Name: string(ProvisioningEventPlannedReboot), Src: allStates, Dst: FsmStatePlannedReboot},
	{Name: string(ProvisioningEventCrashed), Src: []string{FsmStateInitial}, Dst: FsmStateCrashed},
	{Name: string(ProvisioningEventResetFailCount), Src: allStates, Dst: FsmStateResestFailCount},
}

var callbacks = map[string]fsm.Callback{}

// ProvisioningEventType indicates an event emitted by a machine during the provisioning sequence
type ProvisioningEventType string

// The enums for the machine provisioning events.
const (
	ProvisioningEventAlive            ProvisioningEventType = "Alive"
	ProvisioningEventCrashed          ProvisioningEventType = "Crashed"
	ProvisioningEventResetFailCount   ProvisioningEventType = "Reset Fail Count"
	ProvisioningEventPXEBooting       ProvisioningEventType = "PXE Booting"
	ProvisioningEventPlannedReboot    ProvisioningEventType = "Planned Reboot"
	ProvisioningEventPreparing        ProvisioningEventType = "Preparing"
	ProvisioningEventRegistering      ProvisioningEventType = "Registering"
	ProvisioningEventWaiting          ProvisioningEventType = "Waiting"
	ProvisioningEventInstalling       ProvisioningEventType = "Installing"
	ProvisioningEventBootingNewKernel ProvisioningEventType = "Booting New Kernel"
	ProvisioningEventPhonedHome       ProvisioningEventType = "Phoned Home"
)

type provisioningEventSequence []ProvisioningEventType

var (
	// AllProvisioningEventTypes are all provisioning events that exist
	AllProvisioningEventTypes = map[ProvisioningEventType]bool{
		ProvisioningEventAlive:            true,
		ProvisioningEventCrashed:          true,
		ProvisioningEventResetFailCount:   true,
		ProvisioningEventPlannedReboot:    true,
		ProvisioningEventPXEBooting:       true,
		ProvisioningEventPreparing:        true,
		ProvisioningEventRegistering:      true,
		ProvisioningEventWaiting:          true,
		ProvisioningEventInstalling:       true,
		ProvisioningEventBootingNewKernel: true,
		ProvisioningEventPhonedHome:       true,
	}
	// ProvisioningEventsInspectionLimit The length of how many provisioning events are being inspected for calculating incomplete cycles
	ProvisioningEventsInspectionLimit = 2 * len(expectedBootSequence) // only saved events count
	// expectedBootSequence is the expected provisioning event sequence of an expected machine boot from PXE up to a running OS
	expectedBootSequence = provisioningEventSequence{
		ProvisioningEventPXEBooting,
		ProvisioningEventPreparing,
		ProvisioningEventRegistering,
		ProvisioningEventWaiting,
		ProvisioningEventInstalling,
		ProvisioningEventBootingNewKernel,
		ProvisioningEventPhonedHome,
	}
	provisioningEventsThatTerminateCycle = provisioningEventSequence{
		ProvisioningEventPlannedReboot,
		ProvisioningEventResetFailCount,
		*expectedBootSequence.lastEvent(),
	}
)

// ProvisioningEvents is just a list of ProvisioningEvents
type ProvisioningEvents []ProvisioningEvent

// Is return true if given event is equal to specific EventType
func (p ProvisioningEventType) Is(event string) bool {
	return string(p) == event
}

func (p provisioningEventSequence) lastEvent() *ProvisioningEventType {
	if len(p) == 0 {
		return nil
	}
	return &p[len(p)-1]
}

func (p provisioningEventSequence) containsEvent(event ProvisioningEventType) bool {
	result := false
	for _, e := range p {
		if event == e {
			result = true
			break
		}
	}
	return result
}

func postEvent(currentFSM *fsm.FSM, event ProvisioningEventType) (*fsm.FSM, error) {
	var nextFSM *fsm.FSM
	err := currentFSM.Event(string(event))
	if err != nil && err.Error() != "no transition" {
		nextFSM = newProvisioningFSM()
		nextFSM.Event(string(event))
	} else {
		nextFSM = currentFSM
	}
	return nextFSM, err
}

func newProvisioningFSM() *fsm.FSM {
	return fsm.NewFSM(
		FsmStateInitial,
		events,
		callbacks,
	)
}

// CalculateIncompleteCycles calculates the number of events that occurred out of order. Can be used to determine if a machine is in an error loop.
func (p *ProvisioningEventContainer) CalculateIncompleteCycles(log *zap.SugaredLogger) string {
	fsm := newProvisioningFSM()
	incompleteCycles := 0
	for _, event := range p.Events {
		var err error = nil
		if fsm, err = postEvent(fsm, event.Event); err != nil && err.Error() != "no transition" {
			incompleteCycles++
		}
		if provisioningEventsThatTerminateCycle.containsEvent(event.Event) {
			incompleteCycles = 0
		}
	}
	result := strconv.Itoa(incompleteCycles)
	return result
}

// TrimEvents trim the events to maxCount
func (p *ProvisioningEventContainer) TrimEvents(maxCount int) {
	if len(p.Events) > maxCount {
		p.Events = p.Events[:maxCount]
	}
}

// ProvisioningEvent is an event emitted by a machine during the provisioning sequence
type ProvisioningEvent struct {
	Time    time.Time             `rethinkdb:"time" json:"time"`
	Event   ProvisioningEventType `rethinkdb:"event" json:"event"`
	Message string                `rethinkdb:"message" json:"message"`
}

// ProvisioningEventContainer stores the provisioning events of a machine
type ProvisioningEventContainer struct {
	Base
	Liveliness                   MachineLiveliness  `rethinkdb:"liveliness" json:"liveliness"`
	Events                       ProvisioningEvents `rethinkdb:"events" json:"events"`
	LastEventTime                *time.Time         `rethinkdb:"last_event_time" json:"last_event_time"`
	IncompleteProvisioningCycles string             `rethinkdb:"incomplete_cycles" json:"incomplete_cycles"`
}

// ProvisioningEventContainers is a list of machine provisioning event containers.
type ProvisioningEventContainers []ProvisioningEventContainer

// ProvisioningEventContainerMap is an indexed map of machine event containers.
type ProvisioningEventContainerMap map[string]ProvisioningEventContainer

// ByID creates a map of event provisioning containers with the id as the index.
func (p ProvisioningEventContainers) ByID() ProvisioningEventContainerMap {
	res := make(ProvisioningEventContainerMap)
	for i, f := range p {
		res[f.ID] = p[i]
	}
	return res
}
