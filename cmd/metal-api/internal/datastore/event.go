package datastore

import (
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-lib/zapup"
)

// ListProvisioningEventContainers returns all machine provisioning event containers.
func (rs *RethinkStore) ListProvisioningEventContainers() (metal.ProvisioningEventContainers, error) {
	es := make(metal.ProvisioningEventContainers, 0)
	err := rs.listEntities(rs.eventTable(), &es)
	return es, err
}

// FindProvisioningEventContainer finds a provisioning event container to a given machine id.
func (rs *RethinkStore) FindProvisioningEventContainer(id string) (*metal.ProvisioningEventContainer, error) {
	var e metal.ProvisioningEventContainer
	err := rs.findEntityByID(rs.eventTable(), &e, id)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// UpdateProvisioningEventContainer updates a provisioning event container.
func (rs *RethinkStore) UpdateProvisioningEventContainer(old *metal.ProvisioningEventContainer, new *metal.ProvisioningEventContainer) error {
	return rs.updateEntity(rs.eventTable(), new, old)
}

// CreateProvisioningEventContainer creates a new provisioning event container.
func (rs *RethinkStore) CreateProvisioningEventContainer(ec *metal.ProvisioningEventContainer) error {
	return rs.createEntity(rs.eventTable(), ec)
}

// UpsertProvisioningEventContainer inserts a machine's event container.
func (rs *RethinkStore) UpsertProvisioningEventContainer(ec *metal.ProvisioningEventContainer) error {
	return rs.upsertEntity(rs.eventTable(), ec)
}
func (rs *RethinkStore) ProvisioningEventForMachine(machineID, event, message string) (*metal.ProvisioningEventContainer, error) {
	ec, err := rs.FindProvisioningEventContainer(machineID)
	if err != nil && !metal.IsNotFound(err) {
		return nil, err
	}

	if ec == nil {
		ec = &metal.ProvisioningEventContainer{
			Base: metal.Base{
				ID: machineID,
			},
			Liveliness: metal.MachineLivelinessAlive,
		}
	}
	now := time.Now()
	ec.LastEventTime = &now

	ev := metal.ProvisioningEvent{
		Time:    now,
		Event:   metal.ProvisioningEventType(event),
		Message: message,
	}
	if ev.Event == metal.ProvisioningEventAlive {
		zapup.MustRootLogger().Sugar().Debugw("received provisioning alive event", "id", ec.ID)
		ec.Liveliness = metal.MachineLivelinessAlive
	} else if ev.Event == metal.ProvisioningEventPhonedHome && len(ec.Events) > 0 && ec.Events[0].Event == metal.ProvisioningEventPhonedHome {
		zapup.MustRootLogger().Sugar().Debugw("swallowing repeated phone home event", "id", ec.ID)
		ec.Liveliness = metal.MachineLivelinessAlive
	} else {
		ec.Events = append([]metal.ProvisioningEvent{ev}, ec.Events...)
		ec.IncompleteProvisioningCycles = ec.CalculateIncompleteCycles(zapup.MustRootLogger().Sugar())
		ec.Liveliness = metal.MachineLivelinessAlive
	}
	ec.TrimEvents(metal.ProvisioningEventsInspectionLimit)

	err = rs.UpsertProvisioningEventContainer(ec)
	return ec, err
}
