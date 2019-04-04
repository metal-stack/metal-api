package metal

import (
	"strconv"
	"time"
)

// ProvisioningEventType indicates an event emitted by a machine during the provisioning sequence
type ProvisioningEventType string

// The enums for the machine provisioning events.
const (
	ProvisioningEventAlive                ProvisioningEventType = "Alive"
	ProvisioningEventPreparing            ProvisioningEventType = "Preparing"
	ProvisioningEventRegistering          ProvisioningEventType = "Registering"
	ProvisioningEventWaiting              ProvisioningEventType = "Waiting"
	ProvisioningEventInstalling           ProvisioningEventType = "Installing"
	ProvisioningEventInstallationFinished ProvisioningEventType = "InstallationFinished"
)

type provisioningEventSequence []ProvisioningEventType

var (
	// AllProvisioningEventTypes are all provisioning events that exist
	AllProvisioningEventTypes = map[ProvisioningEventType]bool{
		ProvisioningEventAlive:                true,
		ProvisioningEventPreparing:            true,
		ProvisioningEventRegistering:          true,
		ProvisioningEventWaiting:              true,
		ProvisioningEventInstalling:           true,
		ProvisioningEventInstallationFinished: true,
	}
	// ProvisioningEventsHistoryLength The length of how many provisioning events are persisted in the database
	ProvisioningEventsHistoryLength = 3 * len(AllProvisioningEventTypes)
	// ExpectedProvisioningEventSequence is the expected sequence in which
	ExpectedProvisioningEventSequence = provisioningEventSequence{
		ProvisioningEventPreparing,
		ProvisioningEventRegistering,
		ProvisioningEventWaiting,
		ProvisioningEventInstalling,
		ProvisioningEventInstallationFinished,
	}
)

// ProvisioningEvents is just a list of ProvisioningEvents
type ProvisioningEvents []ProvisioningEvent

func (p provisioningEventSequence) lastEvent() ProvisioningEventType {
	return p[len(p)-1]
}

func (e *ProvisioningEvent) hasExpectedSuccessor(successor ProvisioningEventType) bool {
	currentEvent := e.Event

	var indexOfCurrent int
	for i, event := range ExpectedProvisioningEventSequence {
		if currentEvent == event {
			indexOfCurrent = i
			break
		}
	}

	var expectedSuccessor ProvisioningEventType
	expectedSuccessorIndex := indexOfCurrent + 1
	if expectedSuccessorIndex >= len(ExpectedProvisioningEventSequence)-1 {
		expectedSuccessor = ExpectedProvisioningEventSequence[0]
	} else {
		expectedSuccessor = ExpectedProvisioningEventSequence[expectedSuccessorIndex]
	}

	return successor == expectedSuccessor
}

// CalculateIncompleteCycles calculates the number of events that occurred out of order. Can be used to determine if a machine is in an error loop.
func (m *ProvisioningEventContainer) CalculateIncompleteCycles() string {
	incompleteCycles := 0
	atLeastOneTimeCompleted := true
	var successor ProvisioningEventType
	for i, event := range m.Events {
		if event.Event == ExpectedProvisioningEventSequence.lastEvent() {
			// cycle complete at this point, we can leave
			break
		}
		if successor != "" && !event.hasExpectedSuccessor(successor) {
			incompleteCycles++
		}
		successor = event.Event
		if i >= ProvisioningEventsHistoryLength-1 {
			// we have reached the end of the events without having reached the last state once...
			atLeastOneTimeCompleted = false
		}
	}
	result := strconv.Itoa(incompleteCycles)
	if !atLeastOneTimeCompleted {
		result = "more than " + result
	}
	return result
}

// ProvisioningEvent is an event emitted by a machine during the provisioning sequence
type ProvisioningEvent struct {
	Time time.Time `json:"time" description:"the time that this event was received" optional:"true" readOnly:"true" rethinkdb:"time"`
	// Event enums have to be enumerated here as defined above in AllProvisioningEventTypes!!
	Event   ProvisioningEventType `enum:"Alive|Preparing|Registering|Waiting|Installing|InstallationFinished|Provisioned" json:"event" description:"the event emitted by the machine" rethinkdb:"event"`
	Message string                `json:"message" description:"an additional message to add to the event" optional:"true" rethinkdb:"message"`
}

// ProvisioningEventContainer stores the provisioning events of a machine
type ProvisioningEventContainer struct {
	ID                           string             `json:"id" description:"references the machine" rethinkdb:"id"`
	Events                       ProvisioningEvents `json:"events" description:"the history of machine provisioning events" rethinkdb:"events"`
	LastEventTime                *time.Time         `json:"last_event_time" description:"the time where the last event was received" rethinkdb:"last_event_time"`
	IncompleteProvisioningCycles string             `json:"incomplete_provisioning_cycles" description:"the amount of incomplete provisioning cycles in the event container" rethinkdb:"incomplete_cycles"`
}

// ProvisioningEventContainers is a list of machine provisioning event containers.
type ProvisioningEventContainers []ProvisioningEventContainer

// MachineProvisioningEventContainerMap is an indexed map of machine event containers.
type ProvisioningEventContainerMap map[string]ProvisioningEventContainer

// ByID creates a map of event provisioning containers with the id as the index.
func (m ProvisioningEventContainers) ByID() ProvisioningEventContainerMap {
	res := make(ProvisioningEventContainerMap)
	for i, f := range m {
		res[f.ID] = m[i]
	}
	return res
}
