package datastore

import (
	"log/slog"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
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

func (rs *RethinkStore) ProvisioningEventForMachine(log *slog.Logger, event *metal.ProvisioningEvent, machineID string) (*metal.ProvisioningEventContainer, error) {
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

	newEC, err := fsm.HandleProvisioningEvent(log, ec, event)
	if err != nil {
		return nil, err
	}

	if err = newEC.Validate(); err != nil {
		return nil, err
	}

	newEC.TrimEvents(100)

	err = rs.UpsertProvisioningEventContainer(newEC)
	return newEC, err
}
