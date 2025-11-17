package migrations

import (
	"fmt"

	r "gopkg.in/rethinkdb/rethinkdb-go.v6"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

type OldReservation_Mig07 struct {
	Amount       int               `rethinkdb:"amount" json:"amount"`
	Description  string            `rethinkdb:"description" json:"description"`
	ProjectID    string            `rethinkdb:"projectid" json:"projectid"`
	PartitionIDs []string          `rethinkdb:"partitionids" json:"partitionids"`
	Labels       map[string]string `rethinkdb:"labels" json:"labels"`
}

type OldReservations_Mig07 []OldReservation_Mig07

type OldSize_Mig07 struct {
	metal.Base
	Reservations OldReservations_Mig07 `rethinkdb:"reservations" json:"reservations"`
}

func init() {
	getOldSizes := func(db *r.Term, session r.QueryExecutor) ([]OldSize_Mig07, error) {
		res, err := db.Table("size").Run(session)
		if err != nil {
			return nil, err
		}
		defer res.Close()

		var entities []OldSize_Mig07
		err = res.All(&entities)
		if err != nil {
			return nil, fmt.Errorf("cannot fetch all entities: %w", err)
		}

		return entities, nil
	}

	datastore.MustRegisterMigration(datastore.Migration{
		Name:    "migrate size reservations to dedicated table",
		Version: 7,
		Up: func(db *r.Term, session r.QueryExecutor, rs *datastore.RethinkStore) error {
			oldSizes, err := getOldSizes(db, session)
			if err != nil {
				return err
			}

			for _, old := range oldSizes {
				for _, rv := range old.Reservations {
					err = rs.CreateSizeReservation(&metal.SizeReservation{
						Base: metal.Base{
							ID:          "",
							Name:        "",
							Description: rv.Description,
						},
						SizeID:       old.ID,
						Amount:       rv.Amount,
						ProjectID:    rv.ProjectID,
						PartitionIDs: rv.PartitionIDs,
						Labels:       rv.Labels,
					})
					if err != nil {
						return err
					}
				}
			}

			// now remove the old field

			_, err = db.Table("size").Replace(func(row r.Term) r.Term {
				return row.Without("reservations")
			}).RunWrite(session)
			if err != nil {
				return err
			}

			return nil
		},
	})
}
