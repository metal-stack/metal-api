package migrations

import (
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
)

func init() {
	datastore.MustRegisterMigration(datastore.Migration{
		Name:    "remove wait table (not used anymore since grpc wait server was introduced)",
		Version: 1,
		Up: func(db *r.Term, session r.QueryExecutor, rs *datastore.RethinkStore) error {
			_, err := db.TableDrop("wait").RunWrite(session)
			return err
		},
	})
}
