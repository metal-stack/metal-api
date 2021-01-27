package migrations

import (
	"fmt"

	r "gopkg.in/rethinkdb/rethinkdb-go.v6"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
)

func init() {
	datastore.MustRegisterMigration(datastore.Migration{
		Name:    "migrate partition.childprefixlength to tenant super network",
		Version: 3,
		Up: func(db *r.Term, session r.QueryExecutor, rs *datastore.RethinkStore) error {
			nws, err := rs.ListNetworks()
			if err != nil {
				return err
			}

			for _, old := range nws {
				if !old.PrivateSuper {
					continue
				}

				new := old
				partition, err := rs.FindPartition(old.PartitionID)
				if err != nil {
					return err
				}
				if partition == nil {
					return fmt.Errorf("unable to find partition for network:%s", old.ID)
				}
				new.ChildPrefixLength = &partition.PrivateNetworkPrefixLength
				err = rs.UpdateNetwork(&old, &new)
				if err != nil {
					return err
				}
			}
			return nil
		},
	})
}
