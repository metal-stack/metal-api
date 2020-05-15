package datastore

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"go4.org/sort"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

// MigrateFunc is a function that contains database migration logic
type MigrateFunc func(rs *RethinkStore) error

// Migrations is a list of migrations
type Migrations []Migration

// Migration defines a database migration
type Migration struct {
	Name    string
	Version uint
	Up      MigrateFunc
}

// MigrationVersionEntry is a version entry in the migration database
type MigrationVersionEntry struct {
	Version uint `rethinkdb:"id"`
}

var (
	migrations            Migrations
	migrationRegisterLock sync.Mutex
)

// MustRegisterMigration registers a migration and panics when a problem occurs
func MustRegisterMigration(m Migration) {
	migrationRegisterLock.Lock()
	defer migrationRegisterLock.Unlock()
	for _, migration := range migrations {
		if migration.Version == m.Version {
			panic(fmt.Sprintf("migration with version %d is defined multiple times", m.Version))
		}
	}
	migrations = append(migrations, m)
}

// NewerThan returns a sorted slice of migrations that are newer than the given version
func (ms Migrations) NewerThan(version uint) Migrations {
	var result Migrations
	for _, m := range ms {
		if m.Version > version {
			result = append(result, m)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Version < result[j].Version
	})

	return result
}

// Migrate runs database migrations and puts the database into read only mode for runtime users
func (rs *RethinkStore) Migrate() error {
	_, err := rs.migrationTable().Insert(MigrationVersionEntry{Version: 0}, r.InsertOpts{
		Conflict: "replace",
	}).RunWrite(rs.session)
	if err != nil {
		return err
	}

	results, err := rs.migrationTable().Max().Run(rs.session)
	if err != nil {
		return err
	}
	defer results.Close()

	var current MigrationVersionEntry
	err = results.One(&current)
	if err != nil {
		return err
	}

	ms := migrations.NewerThan(current.Version)
	migrationRequired := len(ms) > 0

	if !migrationRequired {
		rs.SugaredLogger.Infow("no database migration required", "current-version", current.Version)
		return nil
	}

	rs.SugaredLogger.Infow("database migration required", "current-version", current.Version, "newer-versions", len(ms), "target-version", ms[len(ms)-1].Version)

	rs.SugaredLogger.Infow("setting demoted runtime user to read only", "user", MetalUser)
	_, err = rs.db().Grant(MetalUser, map[string]interface{}{"write": false}).RunWrite(rs.session)
	if err != nil {
		return err
	}
	defer func() {
		rs.SugaredLogger.Infow("removing read only", "user", MetalUser)
		_, err = rs.db().Grant(MetalUser, map[string]interface{}{"write": true}).RunWrite(rs.session)
		if err != nil {
			rs.SugaredLogger.Errorw("error giving back write permissions", "user", MetalUser)
		}
	}()

	for _, m := range ms {
		rs.SugaredLogger.Infow("running database migration", "version", m.Version, "name", m.Name)
		err = m.Up(rs)
		if err != nil {
			return errors.Wrap(err, "error running database migration")
		}

		_, err := rs.migrationTable().Insert(MigrationVersionEntry{Version: m.Version}, r.InsertOpts{
			Conflict: "replace",
		}).RunWrite(rs.session)
		if err != nil {
			return errors.Wrap(err, "error updating database migration version")
		}
	}

	rs.SugaredLogger.Infow("database migration succeeded")

	return nil
}
