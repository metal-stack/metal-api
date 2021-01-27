package migrations

import (
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"

	"github.com/google/uuid"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
)

func init() {
	datastore.MustRegisterMigration(datastore.Migration{
		Name:    "generate allocation uuids for new ip address field (#70)",
		Version: 2,
		Up: func(db *r.Term, session r.QueryExecutor, rs *datastore.RethinkStore) error {
			ips, err := rs.ListIPs()
			if err != nil {
				return err
			}

			for _, old := range ips {
				if old.AllocationUUID != "" {
					continue
				}

				uuid, err := uuid.NewRandom()
				if err != nil {
					return err
				}

				new := old
				new.AllocationUUID = uuid.String()
				err = rs.UpdateIP(&old, &new)
				if err != nil {
					return err
				}
			}
			return nil
		},
	})
}
