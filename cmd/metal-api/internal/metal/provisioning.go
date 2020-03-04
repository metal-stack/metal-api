package metal

import (
	"strconv"
	"time"

	"go.uber.org/zap"
)

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
	ProvisioningEventReinstallAborted ProvisioningEventType = "Reinstall Aborted"
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
		ProvisioningEventReinstallAborted: true,
	}
	// ProvisioningEventsInspectionLimit The length of how many provisioning events are being inspected for calculating incomplete cycles
	ProvisioningEventsInspectionLimit = 2 * len(expectedProvisioningEventSequence) // only saved events count
	// ExpectedProvisioningEventSequence is the expected sequence in which
	expectedProvisioningEventSequence = provisioningEventSequence{
		ProvisioningEventPXEBooting,
		ProvisioningEventPreparing,
		ProvisioningEventRegistering,
		ProvisioningEventWaiting,
		ProvisioningEventInstalling,
		ProvisioningEventBootingNewKernel,
		ProvisioningEventPhonedHome,
	}
	expectedSuccessorEventMap = map[ProvisioningEventType]provisioningEventSequence{
		// some machines could be incapable of sending pxe boot events (depends on BIOS), therefore PXE Booting and Preparing are allowed initial states
		ProvisioningEventPlannedReboot:    {ProvisioningEventPXEBooting, ProvisioningEventPreparing},
		ProvisioningEventPXEBooting:       {ProvisioningEventPXEBooting, ProvisioningEventPreparing},
		ProvisioningEventPreparing:        {ProvisioningEventRegistering, ProvisioningEventPlannedReboot},
		ProvisioningEventRegistering:      {ProvisioningEventWaiting, ProvisioningEventPlannedReboot},
		ProvisioningEventWaiting:          {ProvisioningEventInstalling, ProvisioningEventPlannedReboot},
		ProvisioningEventInstalling:       {ProvisioningEventBootingNewKernel, ProvisioningEventPlannedReboot},
		ProvisioningEventBootingNewKernel: {ProvisioningEventPhonedHome},
		ProvisioningEventPhonedHome:       {ProvisioningEventPXEBooting, ProvisioningEventPreparing},
		ProvisioningEventCrashed:          {ProvisioningEventPXEBooting, ProvisioningEventPreparing},
		ProvisioningEventResetFailCount:   expectedProvisioningEventSequence,
	}
	provisioningEventsThatTerminateCycle = provisioningEventSequence{
		ProvisioningEventPlannedReboot,
		ProvisioningEventResetFailCount,
		*expectedProvisioningEventSequence.lastEvent(),
	}
)

// ProvisioningEvents is just a list of ProvisioningEvents
type ProvisioningEvents []ProvisioningEvent

// Is return true if given event is equal to specific EventType
func (p ProvisioningEventType) Is(event string) bool {
	return string(p) == event
}

// FIXME @gerrit this method is unused
// func (p provisioningEventSequence) firstEvent() *ProvisioningEventType {
// 	if len(p) == 0 {
// 		return nil
// 	}
// 	return &p[0]
// }

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

func (e *ProvisioningEvent) hasExpectedSuccessor(log *zap.SugaredLogger, actualSuccessor ProvisioningEventType) bool {
	currentEvent := e.Event

	expectedSuccessors, ok := expectedSuccessorEventMap[currentEvent]
	if !ok {
		log.Errorw("successor map does not contain an expected successor for event", "event", currentEvent)
		return false
	}

	log.Debugw("checking expected successor", "currentEvent", currentEvent, "actualSuccessor", actualSuccessor, "expectedSuccessor", expectedSuccessors)

	return expectedSuccessors.containsEvent(actualSuccessor)
}

// CalculateIncompleteCycles calculates the number of events that occurred out of order. Can be used to determine if a machine is in an error loop.
func (p *ProvisioningEventContainer) CalculateIncompleteCycles(log *zap.SugaredLogger) string {
	incompleteCycles := 0
	atLeastOneTimeCompleted := true
	var successor ProvisioningEventType
	for i, event := range p.Events {
		if successor != "" && !event.hasExpectedSuccessor(log, successor) {
			incompleteCycles++
		}
		successor = event.Event

		if provisioningEventsThatTerminateCycle.containsEvent(event.Event) {
			break
		}

		if i >= ProvisioningEventsInspectionLimit-1 {
			// we have reached the inspection limit without having reached the last event in the sequence once...
			atLeastOneTimeCompleted = false
		}
	}
	result := strconv.Itoa(incompleteCycles)
	if incompleteCycles > 0 && !atLeastOneTimeCompleted {
		result = "at least " + result
	}
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
