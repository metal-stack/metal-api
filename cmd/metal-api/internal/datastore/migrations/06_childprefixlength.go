package migrations

import (
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
)

func init() {
	type tmpPartition struct {
		PrivateNetworkPrefixLength uint8 `rethinkdb:"privatenetworkprefixlength"`
	}
	datastore.MustRegisterMigration(datastore.Migration{
		Name:    "migrate partition.childprefixlength to tenant super network",
		Version: 6,
		Up: func(db *r.Term, session r.QueryExecutor, rs *datastore.RethinkStore) error {
			nws, err := rs.ListNetworks()
			if err != nil {
				return err
			}

			for _, old := range nws {
				if !old.PrivateSuper {
					continue
				}

				cursor, err := db.Table("partition").Get(old.PartitionID).Run(session)
				if err != nil {
					return err
				}
				var partition tmpPartition
				err = cursor.One(&partition)
				if err != nil {
					return err
				}

				new := old
				new.ChildPrefixLength = &partition.PrivateNetworkPrefixLength
				err = rs.UpdateNetwork(&old, &new)
				if err != nil {
					return err
				}
				err = cursor.Close()
				if err != nil {
					return err
				}
			}

			_, err = db.Table("partition").Replace(r.Row.Without("privatenetworkprefixlength")).RunWrite(session)
			return err
		},
	})
}
