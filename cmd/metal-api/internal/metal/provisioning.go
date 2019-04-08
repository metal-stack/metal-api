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
	ProvisioningEventPlannedReboot    ProvisioningEventType = "Planned Reboot"
	ProvisioningEventPreparing        ProvisioningEventType = "Preparing"
	ProvisioningEventRegistering      ProvisioningEventType = "Registering"
	ProvisioningEventWaiting          ProvisioningEventType = "Waiting"
	ProvisioningEventInstalling       ProvisioningEventType = "Installing"
	ProvisioningEventBootingNewKernel ProvisioningEventType = "Booting New Kernel"
)

type provisioningEventSequence []ProvisioningEventType

var (
	// AllProvisioningEventTypes are all provisioning events that exist
	AllProvisioningEventTypes = map[ProvisioningEventType]bool{
		ProvisioningEventAlive:            true,
		ProvisioningEventCrashed:          true,
		ProvisioningEventPlannedReboot:    true,
		ProvisioningEventPreparing:        true,
		ProvisioningEventRegistering:      true,
		ProvisioningEventWaiting:          true,
		ProvisioningEventInstalling:       true,
		ProvisioningEventBootingNewKernel: true,
	}
	// ProvisioningEventsInspectionLimit The length of how many provisioning events are being inspected for calculating incomplete cycles
	provisioningEventsInspectionLimit = 3 * (len(AllProvisioningEventTypes) - 2) // reset and alive can be neglected
	// ExpectedProvisioningEventSequence is the expected sequence in which
	expectedProvisioningEventSequence = provisioningEventSequence{
		ProvisioningEventPreparing,
		ProvisioningEventRegistering,
		ProvisioningEventWaiting,
		ProvisioningEventInstalling,
		ProvisioningEventBootingNewKernel,
	}
	expectedSuccessorEventMap = map[ProvisioningEventType]provisioningEventSequence{
		ProvisioningEventPlannedReboot:    provisioningEventSequence{ProvisioningEventPreparing},
		ProvisioningEventPreparing:        provisioningEventSequence{ProvisioningEventRegistering, ProvisioningEventPlannedReboot},
		ProvisioningEventRegistering:      provisioningEventSequence{ProvisioningEventWaiting, ProvisioningEventPlannedReboot},
		ProvisioningEventWaiting:          provisioningEventSequence{ProvisioningEventInstalling, ProvisioningEventPlannedReboot},
		ProvisioningEventInstalling:       provisioningEventSequence{ProvisioningEventBootingNewKernel, ProvisioningEventPlannedReboot},
		ProvisioningEventBootingNewKernel: provisioningEventSequence{ProvisioningEventPreparing, ProvisioningEventPlannedReboot},
		ProvisioningEventCrashed:          provisioningEventSequence{ProvisioningEventPreparing},
	}
)

// ProvisioningEvents is just a list of ProvisioningEvents
type ProvisioningEvents []ProvisioningEvent

func (p provisioningEventSequence) firstEvent() *ProvisioningEventType {
	if len(p) == 0 {
		return nil
	}
	return &p[0]
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
func (m *ProvisioningEventContainer) CalculateIncompleteCycles(log *zap.SugaredLogger) string {
	incompleteCycles := 0
	atLeastOneTimeCompleted := true
	var successor ProvisioningEventType
	lastEventInSequence := expectedProvisioningEventSequence.lastEvent()
	for i, event := range m.Events {
		if lastEventInSequence != nil && event.Event == *lastEventInSequence {
			// cycle complete at this point, we can leave
			break
		}

		if successor != "" && !event.hasExpectedSuccessor(log, successor) {
			incompleteCycles++
		}
		successor = event.Event

		if i >= provisioningEventsInspectionLimit-1 {
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

// ProvisioningEvent is an event emitted by a machine during the provisioning sequence
type ProvisioningEvent struct {
	Time time.Time `json:"time" description:"the time that this event was received" optional:"true" readOnly:"true" rethinkdb:"time"`
	// Event enums have to be enumerated here as defined above in AllProvisioningEventTypes!!
	Event   ProvisioningEventType `enum:"Alive|Planned Reboot|Preparing|Registering|Waiting|Installing|Booting New Kernel" json:"event" description:"the event emitted by the machine" rethinkdb:"event"`
	Message string                `json:"message" description:"an additional message to add to the event" optional:"true" rethinkdb:"message"`
}

// ProvisioningEventContainer stores the provisioning events of a machine
type ProvisioningEventContainer struct {
	ID                           string             `json:"id" description:"references the machine" rethinkdb:"id"`
	Events                       ProvisioningEvents `json:"events" description:"the history of machine provisioning events" rethinkdb:"events"`
	LastEventTime                *time.Time         `json:"last_event_time" description:"the time where the last event was received" rethinkdb:"last_event_time"`
	IncompleteProvisioningCycles string             `json:"incomplete_provisioning_cycles" description:"the amount of incomplete provisioning cycles in the event container" rethinkdb:"incomplete_cycles"`
}

// RecentProvisioningEvents is for the client response with an amount of events trimmed down to reduce the size of the payload
type RecentProvisioningEvents struct {
	Events                       ProvisioningEvents `json:"log" description:"the log of recent machine provisioning events"`
	LastEventTime                *time.Time         `json:"last_event_time" description:"the time where the last event was received"`
	IncompleteProvisioningCycles string             `json:"incomplete_provisioning_cycles" description:"the amount of incomplete provisioning cycles in the event container"`
}

// RecentProvisioningEventsLimit defines how many recent events are added to the RecentProvisioningEvents struct
const RecentProvisioningEventsLimit = 5

// ProvisioningEventContainers is a list of machine provisioning event containers.
type ProvisioningEventContainers []ProvisioningEventContainer

// ProvisioningEventContainerMap is an indexed map of machine event containers.
type ProvisioningEventContainerMap map[string]ProvisioningEventContainer

// ByID creates a map of event provisioning containers with the id as the index.
func (m ProvisioningEventContainers) ByID() ProvisioningEventContainerMap {
	res := make(ProvisioningEventContainerMap)
	for i, f := range m {
		res[f.ID] = m[i]
	}
	return res
}
