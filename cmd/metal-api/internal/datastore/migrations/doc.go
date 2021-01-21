// Package migrations contains migration functions for migrating the RethinkDB.
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
// deployment orchestration to apply a migration.
//
// We also do not support down migrations for the time being because it also makes
// things more complicated than they need to be.
//
// Please ensure that your migrations are idempotent (they need to work for existing and
// for fresh deployments). Check the state before modifying it.
package migrations
