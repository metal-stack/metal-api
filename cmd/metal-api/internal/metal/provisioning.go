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
	LastErrorEvent       *ProvisioningEvent `rethinkdb:"last_error_event" json:"last_error_event"`
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
	if c == nil || len(c.Events) == 0 {
		return nil
	}

	lastEventTime := c.Events[0].Time

	// LastEventTime field in container may be equal or later than the time of the last event
	// because some events will update the field but not be appended
	if c.LastEventTime == nil || lastEventTime.After(*c.LastEventTime) {
		return fmt.Errorf("last event time not up to date in provisioning event container for machine %s", c.ID)
	}

	for _, e := range c.Events {
		if e.Time.After(lastEventTime) {
			return fmt.Errorf("provisioning event container for machine %s is not chronologically sorted", c.ID)
		}

		lastEventTime = e.Time
	}

	return nil
}
