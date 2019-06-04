package datastore

import (
	"fmt"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
)

// FindVrf returns a vrf for a given id.
func (rs *RethinkStore) FindVrf(id string) (*metal.Vrf, error) {
	var v metal.Vrf
	err := rs.findEntityByID(rs.vrfTable(), &v, id)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// FindVrfByProject returns the vrf with attached to the given project and tenant.
func (rs *RethinkStore) FindVrfByProject(projectID, tenantID string) (*metal.Vrf, error) {
	q := *rs.vrfTable()
	q = q.Filter(map[string]interface{}{"tenant": tenantID, "projectid": projectID})

	var vs []metal.Vrf
	err := rs.searchEntities(&q, &vs)
	if err != nil {
		return nil, err
	}
	if len(vs) == 0 {
		return nil, metal.NotFound("no vrf for project with id %q and tenant with id %q found", projectID, tenantID)
	}
	if len(vs) > 1 {
		return nil, fmt.Errorf("more than one vrf for project with id %q and tenant with id %q, which should not be the case", projectID, tenantID)
	}

	return &vs[0], nil
}

func (rs *RethinkStore) CreateVrf(v *metal.Vrf) error {
	_, err := rs.vrfTable().Insert(v).RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot create vrf in database: %v", err)
	}
	return nil
}

// ReserveNewVrf creates a new vrf wit ha generated vrf id (the id depends on projectid)
func (rs *RethinkStore) ReserveNewVrf(tenant, projectid string) (*metal.Vrf, error) {
	var hashInput = tenant + projectid
	try := 0
	for {
		id, err := metal.GenerateVrfID(hashInput)
		if err != nil {
			return nil, err
		}
		vrf := &metal.Vrf{
			Base: metal.Base{
				ID: id,
			},
			Tenant:    tenant,
			ProjectID: projectid,
		}
		err = rs.CreateVrf(vrf)
		if err != nil {
			if metal.IsConflict(err) {
				hashInput += "salt"
				if try > 100 {
					return nil, fmt.Errorf("reached threshold for trying to generate vrfID that does not yet exist")
				}
				try++
				continue
			}
			return nil, err
		}

		return vrf, nil
	}
}
