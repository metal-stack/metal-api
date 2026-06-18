package migrations

import (
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

func init() {
	datastore.MustRegisterMigration(datastore.Migration{
		Name:    "convert reserved machine to tainted machines",
		Version: 10,
		Up: func(db *r.Term, session r.QueryExecutor, rs *datastore.RethinkStore) error {
			ms, err := rs.ListMachines()
			if err != nil {
				return err
			}

			for i := range ms {
				old := ms[i]

				n := old

				if n.State.Value == metal.ReservedState {
					n.State.Value = metal.TaintedState
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
