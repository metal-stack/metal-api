package metal

import (
	"fmt"
	"time"
)

// ProvisioningEventType indicates an event emitted by a machine during the provisioning sequence
type ProvisioningEventType string

func (t ProvisioningEventType) String() string {
	return string(t)
}

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
	ProvisioningEventMachineReclaim   ProvisioningEventType = "Machine Reclaim"
)

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
		ProvisioningEventMachineReclaim:   true,
	}
)

// ProvisioningEvents is just a list of ProvisioningEvents
type ProvisioningEvents []ProvisioningEvent

// Is return true if given event is equal to specific EventType
func (p ProvisioningEventType) Is(event string) bool {
	return string(p) == event
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
	Liveliness           MachineLiveliness  `rethinkdb:"liveliness" json:"liveliness"`
	Events               ProvisioningEvents `rethinkdb:"events" json:"events"`
	LastEventTime        *time.Time         `rethinkdb:"last_event_time" json:"last_event_time"`
	CrashLoop            bool               `rethinkdb:"crash_loop" json:"crash_loop"`
	FailedMachineReclaim bool               `rethinkdb:"failed_machine_reclaim" json:"failed_machine_reclaim"`
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

func (c *ProvisioningEventContainer) Validate() error {
	if len(c.Events) == 0 {
		return nil
	}

	lastEventTime := c.Events[0].Time
	for _, e := range c.Events {
		if e.Time.Before(lastEventTime) {
			return fmt.Errorf("provisioning event container for machine %s is not chronologically sorted", c.ID)
		}

		lastEventTime = e.Time
	}

	// last event time in container can be equal or later than the time of the last event, because some events
	// will update the LastEventTime field without being appended to the container.
	if c.LastEventTime == nil || lastEventTime.After(*c.LastEventTime) {
		return fmt.Errorf("inconsistency in container for machine %s: time of last event is not equal to last event time field", c.ID)
	}

	return nil
}
