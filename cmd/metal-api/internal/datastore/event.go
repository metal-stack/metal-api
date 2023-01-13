package datastore

import (
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/fsm/states"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	s "github.com/metal-stack/metal-api/cmd/metal-api/internal/scaler"
	"github.com/metal-stack/metal-lib/bus"
	"go.uber.org/zap"
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
func (rs *RethinkStore) ProvisioningEventForMachine(log *zap.SugaredLogger, publisher bus.Publisher, event *metal.ProvisioningEvent, machine *metal.Machine) (*metal.ProvisioningEventContainer, error) {
	ec, err := rs.FindProvisioningEventContainer(machine.ID)
	if err != nil && !metal.IsNotFound(err) {
		return nil, err
	}

	if ec == nil {
		ec = &metal.ProvisioningEventContainer{
			Base: metal.Base{
				ID: machine.ID,
			},
			Liveliness: metal.MachineLivelinessAlive,
		}
	}

	p, err := rs.FindPartition(machine.PartitionID)
	if err != nil {
		return nil, err
	}

	manager := &manager{
		rs:          rs,
		publisher:   publisher,
		partitionid: p.ID,
		sizeid:      machine.SizeID,
	}

	c := &s.PoolScalerConfig{
		Log:       log.Named("pool-scaler"),
		Manager:   manager,
		Partition: *p,
	}
	scaler := s.NewPoolScaler(c)

	config := states.StateConfig{
		Log:       log.Named("fsm"),
		Container: ec,
		Event:     event,
		Scaler:    scaler,
		Machine:   machine,
	}

	newEC, err := fsm.HandleProvisioningEvent(&config)
	if err != nil {
		return nil, err
	}

	if err = newEC.Validate(); err != nil {
		return nil, err
	}

	newEC.TrimEvents(100)
	err = rs.UpsertProvisioningEventContainer(newEC)
	if err != nil {
		return nil, err
	}

	return newEC, err
}
