package datastore

import (
	"context"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

// ListProvisioningEventContainers returns all machine provisioning event containers.
func (rs *RethinkStore) ListProvisioningEventContainers(ctx context.Context) (metal.ProvisioningEventContainers, error) {
	es := make(metal.ProvisioningEventContainers, 0)
	err := rs.listEntities(ctx, rs.eventTable(), &es)
	return es, err
}

// FindProvisioningEventContainer finds a provisioning event container to a given machine id.
func (rs *RethinkStore) FindProvisioningEventContainer(ctx context.Context, id string) (*metal.ProvisioningEventContainer, error) {
	var e metal.ProvisioningEventContainer
	err := rs.findEntityByID(ctx, rs.eventTable(), &e, id)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// UpdateProvisioningEventContainer updates a provisioning event container.
func (rs *RethinkStore) UpdateProvisioningEventContainer(ctx context.Context, old *metal.ProvisioningEventContainer, new *metal.ProvisioningEventContainer) error {
	return rs.updateEntity(ctx, rs.eventTable(), new, old)
}

// CreateProvisioningEventContainer creates a new provisioning event container.
func (rs *RethinkStore) CreateProvisioningEventContainer(ctx context.Context, ec *metal.ProvisioningEventContainer) error {
	return rs.createEntity(ctx, rs.eventTable(), ec)
}

// UpsertProvisioningEventContainer inserts a machine's event container.
func (rs *RethinkStore) UpsertProvisioningEventContainer(ctx context.Context, ec *metal.ProvisioningEventContainer) error {
	return rs.upsertEntity(ctx, rs.eventTable(), ec)
}
