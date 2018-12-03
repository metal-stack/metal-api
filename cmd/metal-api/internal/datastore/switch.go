package datastore

import (
	"fmt"

	"git.f-i-ts.de/cloud-native/metal/metal-api/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

func (rs *RethinkStore) FindSwitch(id string) (*metal.Switch, error) {
	res, err := rs.switchTable().Get(id).Run(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot get switch from database: %v", err)
	}
	defer res.Close()
	if res.IsNil() {
		return nil, metal.ErrNotFound
	}
	var sw metal.Switch
	err = res.One(&sw)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch results: %v", err)
	}
	return &sw, nil
}

func (rs *RethinkStore) ListSwitches() ([]metal.Switch, error) {
	res, err := rs.switchTable().Run(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot search switches from database: %v", err)
	}
	defer res.Close()
	switches := make([]metal.Switch, 0)
	err = res.All(&switches)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch results: %v", err)
	}
	return switches, nil
}

func (rs *RethinkStore) CreateSwitch(s *metal.Switch) (*metal.Switch, error) {
	res, err := rs.switchTable().Insert(s).RunWrite(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot create switch in database: %v", err)
	}
	if s.ID == "" {
		s.ID = res.GeneratedKeys[0]
	}
	return s, nil
}

func (rs *RethinkStore) DeleteSwitch(id string) (*metal.Switch, error) {
	img, err := rs.FindSwitch(id)
	if err != nil {
		return nil, fmt.Errorf("cannot find switch with id %q: %v", id, err)
	}
	_, err = rs.switchTable().Get(id).Delete().RunWrite(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot delete switch from database: %v", err)
	}
	return img, nil
}

func (rs *RethinkStore) UpdateSwitch(oldSwitch *metal.Switch, newSwitch *metal.Switch) error {
	_, err := rs.switchTable().Get(oldSwitch.ID).Replace(func(row r.Term) r.Term {
		return r.Branch(row.Field("changed").Eq(r.Expr(oldSwitch.Changed)), newSwitch, r.Error("the switch was changed from another, please retry"))
	}).RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot update switch: %v", err)
	}
	return nil
}
