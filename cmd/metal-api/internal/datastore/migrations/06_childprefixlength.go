package migrations

import (
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
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
				cursor, err := db.Table("partition").Get(old.PartitionID).Run(session)
				if err != nil {
					return err
				}
				var partition tmpPartition
				err = cursor.One(&partition)
				if err != nil {
					return err
				}

				// TODO: does not work somehow
				new := old

				af, err := metal.GetAddressFamily(new.Prefixes)
				if err != nil {
					return err
				}
				if af != nil {
					if new.AddressFamilies == nil {
						new.AddressFamilies = make(map[metal.AddressFamily]bool)
					}
					new.AddressFamilies[*af] = true
				}
				if new.PrivateSuper {
					if new.DefaultChildPrefixLength == nil {
						new.DefaultChildPrefixLength = make(map[metal.AddressFamily]uint8)
					}
					new.DefaultChildPrefixLength[*af] = partition.PrivateNetworkPrefixLength
				}
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
