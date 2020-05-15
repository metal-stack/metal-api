package migrations

import (
	"github.com/google/uuid"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
)

func init() {
	datastore.MustRegisterMigration(datastore.Migration{
		Name:    "generate allocation uuids for new ip address field",
		Version: 1,
		Up: func(rs *datastore.RethinkStore) error {
			ips, err := rs.ListIPs()
			if err != nil {
				return err
			}

			for _, old := range ips {
				if old.AllocationUUID != "" {
					continue
				}
				u, err := uuid.NewRandom()
				if err != nil {
					return err
				}
				new := old
				new.AllocationUUID = u.String()
				err = rs.UpdateIP(&old, &new)
				if err != nil {
					return err
				}
			}
			return nil
		},
	})
}
