package datastore

import (
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
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

// FindProvisioningEventContainerAllowNil returns the provisioning event container with the given ID. If there is no
// such provisioning event container nil will be returned.
func (rs *RethinkStore) FindProvisioningEventContainerAllowNil(id string) (*metal.ProvisioningEventContainer, error) {
	var e metal.ProvisioningEventContainer
	err := rs.findEntityByIDAllowNil(rs.eventTable(), &e, id)
	if e.ID != "" {
		return &e, err
	}
	return nil, err
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
