package datastore

import (
	"crypto/sha256"
	"fmt"
	"strconv"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
)

func generateVrfID(i string) (uint, error) {
	sha := sha256.Sum256([]byte(i))
	// cut four bytes of hash
	hexTrunc := fmt.Sprintf("%x", sha)[:4]
	hash, err := strconv.ParseUint(hexTrunc, 16, 16)
	if err != nil {
		return 0, err
	}
	return uint(hash), err
}

// TODO: Use generic methods from rethinkstore

func (rs *RethinkStore) FindVrf(f map[string]interface{}) (*metal.Vrf, error) {
	q := *rs.vrfTable()
	q = q.Filter(f)
	res, err := q.Run(rs.session)
	defer res.Close()
	if res.IsNil() {
		return nil, nil
	}
	var vrf metal.Vrf
	err = res.One(&vrf)
	if err != nil {
		return nil, err
	}
	return &vrf, nil
}

func (rs *RethinkStore) CreateVrf(v *metal.Vrf) error {
	_, err := rs.vrfTable().Insert(v).RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot create vrf in database: %v", err)
	}
	return nil
}

func (rs *RethinkStore) ReserveNewVrf(tenant, projectid string) (*metal.Vrf, error) {
	var hashInput = tenant + projectid
	for {
		id, err := generateVrfID(hashInput)
		if err != nil {
			return nil, err
		}
		vrf, err := rs.FindVrf(map[string]interface{}{"id": id})
		if err != nil {
			return nil, err
		}
		if vrf != nil {
			hashInput += "salt"
			continue
		}
		vrf = &metal.Vrf{
			ID:        id,
			Tenant:    tenant,
			ProjectID: projectid,
		}
		_, err = rs.vrfTable().Insert(vrf).RunWrite(rs.session)
		if err != nil {
			return nil, err
		}
		return vrf, nil
	}
}
