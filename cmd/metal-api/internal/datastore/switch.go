package datastore

import (
	"context"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

// FindSwitch returns a switch for a given id.
func (rs *RethinkStore) FindSwitch(ctx context.Context, id string) (*metal.Switch, error) {
	var s metal.Switch
	err := rs.findEntityByID(ctx, rs.switchTable(), &s, id)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// ListSwitches returns all known switches.
func (rs *RethinkStore) ListSwitches(ctx context.Context) ([]metal.Switch, error) {
	ss := make([]metal.Switch, 0)
	err := rs.listEntities(ctx, rs.switchTable(), &ss)
	return ss, err
}

// CreateSwitch creates a new switch.
func (rs *RethinkStore) CreateSwitch(ctx context.Context, s *metal.Switch) error {
	return rs.createEntity(ctx, rs.switchTable(), s)
}

// DeleteSwitch deletes a switch.
func (rs *RethinkStore) DeleteSwitch(ctx context.Context, s *metal.Switch) error {
	return rs.deleteEntity(ctx, rs.switchTable(), s)
}

// UpdateSwitch updates a switch.
func (rs *RethinkStore) UpdateSwitch(ctx context.Context, oldSwitch *metal.Switch, newSwitch *metal.Switch) error {
	return rs.updateEntity(ctx, rs.switchTable(), newSwitch, oldSwitch)
}

// SearchSwitches searches for switches by the given parameters.
func (rs *RethinkStore) SearchSwitches(ctx context.Context, rackid string, macs []string) ([]metal.Switch, error) {
	q := *rs.switchTable()

	if rackid != "" {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("rackid").Eq(rackid)
		})
	}

	if len(macs) > 0 {
		macexpr := r.Expr(macs)

		q = q.Filter(func(row r.Term) r.Term {
			return macexpr.SetIntersection(row.Field("network_interfaces").Field("macAddress")).Count().Gt(1)
		})
	}

	var ss []metal.Switch
	err := rs.searchEntities(ctx, &q, &ss)
	if err != nil {
		return nil, err
	}

	return ss, nil
}

// SearchSwitchesConnectedToMachine searches switches that are connected to the given machine.
func (rs *RethinkStore) SearchSwitchesConnectedToMachine(ctx context.Context, m *metal.Machine) ([]metal.Switch, error) {
	switches, err := rs.SearchSwitches(ctx, m.RackID, nil)
	if err != nil {
		return nil, err
	}

	res := []metal.Switch{}
	for _, sw := range switches {
		if _, has := sw.MachineConnections[m.ID]; has {
			res = append(res, sw)
		}
	}
	return res, nil
}
