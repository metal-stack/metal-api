package migrations

import (
	"net/netip"

	r "gopkg.in/rethinkdb/rethinkdb-go.v6"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

func init() {
	type tmpPartition struct {
		// In theory this might be set in a partition, but in reality its not set anywhere
		PrivateNetworkPrefixLength uint8 `rethinkdb:"privatenetworkprefixlength"`
	}
	datastore.MustRegisterMigration(datastore.Migration{
		Name:    "migrate partition.childprefixlength to tenant super network",
		Version: 7,
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

				new := old

				var (
					af                       metal.AddressFamily
					defaultChildPrefixLength = metal.ChildPrefixLength{}
				)
				parsed, err := netip.ParsePrefix(new.Prefixes[0].String())
				if err != nil {
					return err
				}
				if parsed.Addr().Is4() {
					af = metal.IPv4AddressFamily
					defaultChildPrefixLength[af] = 22
				}
				if parsed.Addr().Is6() {
					af = metal.IPv6AddressFamily
					defaultChildPrefixLength[af] = 64
				}

				if new.AddressFamilies == nil {
					new.AddressFamilies = make(map[metal.AddressFamily]bool)
				}
				new.AddressFamilies[af] = true

				if new.PrivateSuper {
					if new.DefaultChildPrefixLength == nil {
						new.DefaultChildPrefixLength = make(map[metal.AddressFamily]uint8)
					}
					if partition.PrivateNetworkPrefixLength > 0 {
						new.DefaultChildPrefixLength[af] = partition.PrivateNetworkPrefixLength
					} else {
						new.DefaultChildPrefixLength = defaultChildPrefixLength
					}

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
