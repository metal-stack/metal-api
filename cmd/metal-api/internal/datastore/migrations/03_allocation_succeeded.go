package migrations

import (
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
)

func init() {
	datastore.MustRegisterMigration(datastore.Migration{
		Name:    "set allocation succeeded to true for allocated machines (#210)",
		Version: 3,
		Up: func(db *r.Term, session r.QueryExecutor, rs *datastore.RethinkStore) error {
			ms, err := rs.ListMachines()
			if err != nil {
				return err
			}

			for i := range ms {
				old := ms[i]
				if old.Allocation == nil || old.Allocation.ConsolePassword == "" {
					// these machines have never succeeded finalize-allocation
					continue
				}

				if old.Allocation.Succeeded {
					continue
				}

				n := old
				n.Allocation.Succeeded = true
				err = rs.UpdateMachine(&old, &n)
				if err != nil {
					return err
				}
			}
			return nil
		},
	})
}
