// Package migrations contain migration functions for migrating the RethinkDB.
//
// Migrating RethinkDB is a bit different than compared to regular SQL databases because
// clients define the schema and not the server.
//
// Currently, migrations are only intended to be run *after* the rollout of the new clients.
// This prevents older clients to write their old schema into the database after the migration
// was applied. This approach allows us to apply zero-downtime migrations for most of the
// use-cases we have seen in the past.
//
// There are probably scenarios where it makes sense to migrate *before* instance
// rollout and stop the instances before the migration (downtime migration) but for now
// this use-case has not been implemented and it possibly requires more difficult
// orchestration of the migration in the deployment.
//
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
