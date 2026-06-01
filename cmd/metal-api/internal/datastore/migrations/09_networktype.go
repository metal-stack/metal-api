package migrations

import (
	"slices"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"

	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

func init() {
	datastore.MustRegisterMigration(datastore.Migration{
		Name:    "migrate networks to have proper networkType set",
		Version: 9,
		Up: func(db *r.Term, session r.QueryExecutor, ds *datastore.RethinkStore) error {
			nws, err := ds.ListNetworks()
			if err != nil {
				return err
			}

			// first detect all shared vrf ids
			// maps vrfid > count
			var (
				vrfCount   = make(map[uint]int)
				sharedVrfs []uint
			)

			for _, nw := range nws {
				if nw.Vrf == 0 {
					continue
				}
				count, ok := vrfCount[nw.Vrf]
				if !ok {
					vrfCount[nw.Vrf] = 1
				} else {
					vrfCount[nw.Vrf] = count + 1
				}
			}

			for vrf, count := range vrfCount {
				if count > 1 {
					sharedVrfs = append(sharedVrfs, vrf)
				}
			}

			// now convert all networks
			for _, old := range nws {
				newNetwork := old

				// assume external network by default
				newNetwork.NetworkType = new(metal.ExternalNetworkType)

				if old.Shared && old.ParentNetworkID != "" {
					newNetwork.NetworkType = new(metal.ChildSharedNetworkType)
				}
				if old.Shared && old.ParentNetworkID == "" {
					newNetwork.NetworkType = new(metal.ExternalNetworkType)
				}
				if !old.Shared && old.ParentNetworkID != "" && !slices.Contains(sharedVrfs, old.Vrf) {
					newNetwork.NetworkType = new(metal.ChildNetworkType)
				}
				if old.ProjectID == "" && old.ParentNetworkID == "" && !slices.Contains(sharedVrfs, old.Vrf) {
					newNetwork.NetworkType = new(metal.ExternalNetworkType)
				}
				if old.PrivateSuper {
					newNetwork.NetworkType = new(metal.SuperNetworkType)
				}
				if old.Underlay {
					newNetwork.NetworkType = new(metal.UnderlayNetworkType)
				}

				if old.Nat {
					newNetwork.NATType = new(metal.IPv4MasqueradeNATType)
				} else {
					newNetwork.NATType = new(metal.NoneNATType)
				}

				err := ds.UpdateNetwork(&old, &newNetwork)
				if err != nil {
					return err
				}
			}

			return nil
		},
	})
}
