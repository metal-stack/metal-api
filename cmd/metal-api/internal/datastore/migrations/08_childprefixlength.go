package migrations

import (
	"net/netip"
	"slices"

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
		Version: 8,
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
				if cursor.IsNil() {
					_ = cursor.Close()
					continue
				}
				var partition tmpPartition
				err = cursor.One(&partition)
				if err != nil {
					_ = cursor.Close()
					return err
				}
				err = cursor.Close()
				if err != nil {
					return err
				}
				new := old

				var (
					afs                      metal.AddressFamilies
					defaultChildPrefixLength = metal.ChildPrefixLength{}
				)
				for _, prefix := range new.Prefixes {
					parsed, err := netip.ParsePrefix(prefix.String())
					if err != nil {
						return err
					}
					var af metal.AddressFamily
					if parsed.Addr().Is4() {
						af = metal.IPv4AddressFamily
						defaultChildPrefixLength[af] = 22
					}
					if parsed.Addr().Is6() {
						af = metal.IPv6AddressFamily
						defaultChildPrefixLength[af] = 64
					}
					if !slices.Contains(afs, af) {
						afs = append(afs, af)
					}
				}
				new.AddressFamilies = afs

				if new.PrivateSuper {
					if new.DefaultChildPrefixLength == nil {
						new.DefaultChildPrefixLength = metal.ChildPrefixLength{}
					}
					if partition.PrivateNetworkPrefixLength > 0 {
						defaultChildPrefixLength[metal.IPv4AddressFamily] = partition.PrivateNetworkPrefixLength
					}
					new.DefaultChildPrefixLength = defaultChildPrefixLength
				}
				err = rs.UpdateNetwork(&old, &new)
				if err != nil {
					return err
				}
			}

			_, err = db.Table("partition").Replace(r.Row.Without("privatenetworkprefixlength")).RunWrite(session)
			return err
		},
	})
}
