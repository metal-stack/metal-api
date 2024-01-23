package migrations

import (
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"

	"github.com/google/uuid"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
)

func init() {
	datastore.MustRegisterMigration(datastore.Migration{
		Name:    "generate allocation uuids for already allocated machines",
		Version: 5,
		Up: func(db *r.Term, session r.QueryExecutor, rs *datastore.RethinkStore) error {
			machines, err := rs.ListMachines()
			if err != nil {
				return err
			}

			for _, m := range machines {
				m := m

				if m.Allocation == nil {
					continue
				}

				if m.Allocation.UUID != "" {
					continue
				}

				newMachine := m
				m.Allocation.UUID = uuid.New().String()

				err = rs.UpdateMachine(&m, &newMachine)
				if err != nil {
					return err
				}
			}
			return nil
		},
	})
}
