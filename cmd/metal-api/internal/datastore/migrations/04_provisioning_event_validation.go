package migrations

import (
	"fmt"
	"sort"

	r "gopkg.in/rethinkdb/rethinkdb-go.v6"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
)

func init() {
	datastore.MustRegisterMigration(datastore.Migration{
		Name:    "introduction of provisioning event validation (#265)",
		Version: 4,
		Up: func(db *r.Term, session r.QueryExecutor, rs *datastore.RethinkStore) error {
			ecs, err := rs.ListProvisioningEventContainers()
			if err != nil {
				return err
			}

			for i := range ecs {
				old := ecs[i]
				if old.Validate() == nil {
					continue
				}

				n := old

				sort.Slice(n.Events, func(i, j int) bool {
					return n.Events[i].Time.After(n.Events[j].Time)
				})

				n.LastEventTime = &n.Events[0].Time

				if n.Validate() != nil {
					return fmt.Errorf("unable to fix invalid event container: %s", n.ID)
				}

				if err := rs.UpsertProvisioningEventContainer(&n); err != nil {
					return fmt.Errorf("unable to upsert event container: %w", err)
				}
			}
			return nil
		},
	})
}
