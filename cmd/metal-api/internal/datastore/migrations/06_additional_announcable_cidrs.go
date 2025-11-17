package migrations

import (
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
)

func init() {
	datastore.MustRegisterMigration(datastore.Migration{
		Name:    "migrate super tenant networks to contain additionannouncablecidrs",
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
				new := old

				if len(old.AdditionalAnnouncableCIDRs) == 0 {
					new.AdditionalAnnouncableCIDRs = []string{
						// This was the previous hard coded default in metal-core
						"10.240.0.0/12",
					}
				}

				err = rs.UpdateNetwork(&old, &new)
				if err != nil {
					return err
				}
			}
			return nil
		},
	})
}
