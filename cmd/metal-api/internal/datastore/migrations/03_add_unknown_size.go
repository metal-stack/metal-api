package migrations

import (
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

func init() {
	datastore.MustRegisterMigration(datastore.Migration{
		Name:    "generate a unknown size if not present",
		Version: 3,
		Up: func(db *r.Term, session r.QueryExecutor, rs *datastore.RethinkStore) error {
			_, err := rs.FindSize("unknown")
			if err != nil {
				return err
			}
			err = rs.CreateSize(metal.UnknownSize)
			if err != nil {
				return err
			}

			return nil
		},
	})
}
