package datastore

import (
	"fmt"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

// FindSite return a site for the given id.
func (rs *RethinkStore) FindSite(id string) (*metal.Site, error) {
	res, err := rs.siteTable().Get(id).Run(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot get Site from database: %v", err)
	}
	defer res.Close()
	if res.IsNil() {
		return nil, metal.NotFound("no siete %q found", id)
	}
	var r metal.Site
	err = res.One(&r)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch results: %v", err)
	}
	return &r, nil
}

// ListSites returns all sites.
func (rs *RethinkStore) ListSites() ([]metal.Site, error) {
	res, err := rs.siteTable().Run(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot search facilities from database: %v", err)
	}
	defer res.Close()
	data := make([]metal.Site, 0)
	err = res.All(&data)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch results: %v", err)
	}
	return data, nil
}

// CreateSite creates a new site.
func (rs *RethinkStore) CreateSite(f *metal.Site) error {
	res, err := rs.siteTable().Insert(f).RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot create Site in database: %v", err)
	}
	if f.ID == "" {
		f.ID = res.GeneratedKeys[0]
	}
	return nil
}

// DeleteSite delets a site.
func (rs *RethinkStore) DeleteSite(id string) (*metal.Site, error) {
	f, err := rs.FindSite(id)
	if err != nil {
		return nil, err
	}
	_, err = rs.siteTable().Get(id).Delete().RunWrite(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot delete Site from database: %v", err)
	}
	return f, nil
}

// UpdateSite updates a site.
func (rs *RethinkStore) UpdateSite(oldF *metal.Site, newF *metal.Site) error {
	_, err := rs.siteTable().Get(oldF.ID).Replace(func(row r.Term) r.Term {
		return r.Branch(row.Field("changed").Eq(r.Expr(oldF.Changed)), newF, r.Error("the Site was changed from another, please retry"))
	}).RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot update Site: %v", err)
	}
	return nil
}
