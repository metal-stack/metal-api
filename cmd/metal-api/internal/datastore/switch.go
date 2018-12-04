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
		return nil, metal.NotFound("no switch %q found", id)
	}
	var sw metal.Switch
	err = res.One(&sw)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch results: %v", err)
	}
	return &sw, nil
}

func (rs *RethinkStore) findSwitchByRack(rackid string) ([]metal.Switch, error) {
	q := *rs.switchTable()
	if rackid != "" {
		q = q.Filter(func(s r.Term) r.Term {
			return s.Field("rackid").Eq(rackid)
		})
	}
	res, err := q.Run(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot search switch by rackid: %v", err)
	}
	defer res.Close()
	data := make([]metal.Switch, 0)
	err = res.All(&data)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch results: %v", err)
	}

	return data, nil
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
		return nil, err
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

func (rs *RethinkStore) UpdateSwitchConnections(dev *metal.Device) error {
	switches, err := rs.findSwitchByRack(dev.RackID)
	if err != nil {
		return err
	}
	for _, sw := range switches {
		oldSwitch := sw
		sw.ConnectDevice(dev)
		err := rs.UpdateSwitch(&oldSwitch, &sw)
		if err != nil {
			return err
		}
	}
	return nil
}

func (rs *RethinkStore) findSwithcByMac(macs []metal.Nic) ([]metal.Switch, error) {
	var searchmacs []string
	for _, m := range macs {
		searchmacs = append(searchmacs, string(m.MacAddress))
	}
	macexpr := r.Expr(searchmacs)

	res, err := rs.switchTable().Filter(func(row r.Term) r.Term {
		return macexpr.SetIntersection(row.Field("network_interfaces").Field("macAddress")).Count().Gt(1)
	}).Run(rs.session)
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
