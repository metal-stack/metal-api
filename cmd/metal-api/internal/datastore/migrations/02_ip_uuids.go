package migrations

import (
	"context"

	r "gopkg.in/rethinkdb/rethinkdb-go.v6"

	"github.com/google/uuid"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
)

func init() {
	datastore.MustRegisterMigration(datastore.Migration{
		Name:    "generate allocation uuids for new ip address field (#70)",
		Version: 2,
		Up: func(db *r.Term, session r.QueryExecutor, rs *datastore.RethinkStore) error {
			ips, err := rs.ListIPs(context.Background())
			if err != nil {
				return err
			}

			for i := range ips {
				old := ips[i] // avoids implicit memory aliasing
				if old.AllocationUUID != "" {
					continue
				}

				u, err := uuid.NewRandom()
				if err != nil {
					return err
				}

				n := old
				n.AllocationUUID = u.String()
				err = rs.UpdateIP(context.Background(), &old, &n)
				if err != nil {
					return err
				}
			}
			return nil
		},
	})
}
