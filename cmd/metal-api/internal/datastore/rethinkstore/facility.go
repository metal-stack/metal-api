package rethinkstore

import (
	"fmt"

	"git.f-i-ts.de/cloud-native/maas/metal-api/pkg/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

func (rs *RethinkStore) FindFacility(id string) (*metal.Facility, error) {
	res, err := rs.facilityTable.Get(id).Run(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot get facility from database: %v", err)
	}
	defer res.Close()
	var r metal.Facility
	err = res.One(&r)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch results: %v", err)
	}
	return &r, nil
}

func (rs *RethinkStore) SearchFacility() error {
	return fmt.Errorf("not implemented yet")
}

func (rs *RethinkStore) ListFacilities() ([]metal.Facility, error) {
	res, err := rs.facilityTable.Run(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot search facilities from database: %v", err)
	}
	defer res.Close()
	data := make([]metal.Facility, 0)
	err = res.All(&data)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch results: %v", err)
	}
	return data, nil
}

func (rs *RethinkStore) CreateFacility(f *metal.Facility) error {
	res, err := rs.facilityTable.Insert(f).RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot create facility in database: %v", err)
	}
	if f.ID == "" {
		f.ID = res.GeneratedKeys[0]
	}
	return nil
}

func (rs *RethinkStore) DeleteFacility(id string) (*metal.Facility, error) {
	f, err := rs.FindFacility(id)
	if err != nil {
		return nil, fmt.Errorf("cannot find facility with id %q: %v", id, err)
	}
	_, err = rs.facilityTable.Get(id).Delete().RunWrite(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot delete facility from database: %v", err)
	}
	return f, nil
}

func (rs *RethinkStore) DeleteFacilities() error {
	// we do not support this here! do we really need such a method?
	return nil
}

func (rs *RethinkStore) UpdateFacility(oldF *metal.Facility, newF *metal.Facility) error {
	_, err := rs.facilityTable.Get(oldF.ID).Replace(func(row r.Term) r.Term {
		return r.Branch(row.Field("changed").Eq(r.Expr(oldF.Changed)), newF, r.Error("the facility was changed from another, please retry"))
	}).RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot update facility: %v", err)
	}
	return nil
}
