package datastore

import (
	"fmt"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
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
	return &e, err
}

// UpdateProvisioningEventContainer updates a provisioning event container.
func (rs *RethinkStore) UpdateProvisioningEventContainer(old *metal.ProvisioningEventContainer, new *metal.ProvisioningEventContainer) error {
	return rs.updateEntity(rs.eventTable(), new, old)
}

// CreateProvisioningEventContainer creates a new provisioning event container.
func (rs *RethinkStore) CreateProvisioningEventContainer(ec *metal.ProvisioningEventContainer) error {
	return rs.createEntity(rs.eventTable(), ec)
}

// AddProvisioningEvent adds the provisioning event to a machine's event container.
func (rs *RethinkStore) AddProvisioningEvent(machineID string, event *metal.ProvisioningEvent) error {
	eventContainer, err := rs.FindProvisioningEventContainer(machineID)
	if err != nil {
		eventContainer = &metal.ProvisioningEventContainer{
			ID: machineID,
		}
	}

	eventContainer.LastEventTime = &event.Time

	if event.Event == metal.ProvisioningEventAlive {
		rs.SugaredLogger.Debugw("received provisioning alive event", "id", eventContainer.ID)
	} else {
		eventContainer.Events = append([]metal.ProvisioningEvent{*event}, eventContainer.Events...)
		eventContainer.IncompleteProvisioningCycles = eventContainer.CalculateIncompleteCycles(rs.SugaredLogger)
	}

	_, err = rs.eventTable().Insert(eventContainer, r.InsertOpts{
		Conflict: "replace",
	}).RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot upsert provisioning event container into event table: %v", err)
	}

	return nil
}
