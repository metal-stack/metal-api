package datastore

import (
	"errors"
	"fmt"
	"sync"

	"sort"

	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

// MigrateFunc is a function that contains database migration logic
type MigrateFunc func(db *r.Term, session r.QueryExecutor, rs *RethinkStore) error

// Migrations is a list of migrations
type Migrations []Migration

// Migration defines a database migration
type Migration struct {
	Name    string
	Version int
	Up      MigrateFunc
}

// MigrationVersionEntry is a version entry in the migration database
type MigrationVersionEntry struct {
	Version int    `rethinkdb:"id"`
	Name    string `rethinkdb:"name"`
}

var (
	migrations            Migrations
	migrationRegisterLock sync.Mutex
)

// MustRegisterMigration registers a migration and panics when a problem occurs
func MustRegisterMigration(m Migration) {
	if m.Version < 1 {
		panic(fmt.Sprintf("migrations should start from version number '1', but found version %q", m.Version))
	}
	migrationRegisterLock.Lock()
	defer migrationRegisterLock.Unlock()
	for _, migration := range migrations {
		if migration.Version == m.Version {
			panic(fmt.Sprintf("migration with version %d is defined multiple times", m.Version))
		}
	}
	migrations = append(migrations, m)
}

// Between returns a sorted slice of migrations that are between the given current version
// and target version (target version contained). If target version is nil all newer versions
// than current are contained in the slice.
func (ms Migrations) Between(current int, target *int) (Migrations, error) {
	var result Migrations
	targetFound := false
	for _, m := range ms {
		if target != nil {
			if m.Version > *target {
				continue
			}
			if m.Version == *target {
				targetFound = true
			}
		}

		if m.Version <= current {
			continue
		}

		result = append(result, m)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Version < result[j].Version
	})

	if target != nil && !targetFound {
		return nil, errors.New("target version not found")
	}

	return result, nil
}

// Migrate runs database migrations and puts the database into read only mode for demoted runtime users.
func (rs *RethinkStore) Migrate(targetVersion *int, dry bool) error {
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

	if targetVersion != nil && *targetVersion < current.Version {
		return fmt.Errorf("target version (=%d) smaller than current version (=%d) and down migrations not supported", *targetVersion, current.Version)
	}
	ms, err := migrations.Between(current.Version, targetVersion)
	if err != nil {
		return err
	}

	if len(ms) == 0 {
		rs.log.Info("no database migration required", "current-version", current.Version)
		return nil
	}

	rs.log.Info("database migration required", "current-version", current.Version, "newer-versions", len(ms), "target-version", ms[len(ms)-1].Version)

	if dry {
		for _, m := range ms {
			rs.log.Info("database migration dry run", "version", m.Version, "name", m.Name)
		}
		return nil
	}

	rs.log.Info("setting demoted runtime user to read only", "user", DemotedUser)
	_, err = rs.db().Grant(DemotedUser, map[string]interface{}{"read": true, "write": false}).RunWrite(rs.session)
	if err != nil {
		return err
	}
	defer func() {
		rs.log.Info("removing read only", "user", DemotedUser)
		_, err = rs.db().Grant(DemotedUser, map[string]interface{}{"read": true, "write": true}).RunWrite(rs.session)
		if err != nil {
			rs.log.Error("error giving back write permissions", "user", DemotedUser)
		}
	}()

	for _, m := range ms {
		rs.log.Info("running database migration", "version", m.Version, "name", m.Name)
		err = m.Up(rs.db(), rs.session, rs)
		if err != nil {
			return fmt.Errorf("error running database migration: %w", err)
		}

		_, err := rs.migrationTable().Insert(MigrationVersionEntry{Version: m.Version, Name: m.Name}, r.InsertOpts{
			Conflict: "replace",
		}).RunWrite(rs.session)
		if err != nil {
			return fmt.Errorf("error updating database migration version: %w", err)
		}
	}

	rs.log.Info("database migration succeeded")

	return nil
}
