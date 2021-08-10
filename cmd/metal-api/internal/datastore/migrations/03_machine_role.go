package migrations

import (
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

func init() {
	datastore.MustRegisterMigration(datastore.Migration{
		Name:    "introduction of machine roles (#24)",
		Version: 3,
		Up: func(db *r.Term, session r.QueryExecutor, rs *datastore.RethinkStore) error {
			ms, err := rs.ListMachines()
			if err != nil {
				return err
			}

			for i := range ms {
				old := ms[i]
				if old.Allocation == nil {
					continue
				}

				n := old

				if isFirewall(n.Allocation.MachineNetworks) {
					n.Allocation.Role = metal.RoleFirewall
				} else {
					n.Allocation.Role = metal.RoleMachine
				}

				err = rs.UpdateMachine(&old, &n)
				if err != nil {
					return err
				}
			}
			return nil
		},
	})
}

func isFirewall(nws []*metal.MachineNetwork) bool {
	// only firewalls are part of the underlay network, so that is a unique and sufficient indicator
	for _, n := range nws {
		if n.Underlay {
			return true
		}
	}
	return false
}
