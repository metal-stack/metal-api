package datastore

import (
	"github.com/google/uuid"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
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

var migrations = Migrations{
	{
		Name:    "generate allocation uuids for new ip address field",
		Version: 1,
		Up: func(rs *RethinkStore) error {
			ips := make(metal.IPs, 0)
			err := rs.listEntities(rs.ipTable(), &ips)
			if err != nil {
				return err
			}

			for _, old := range ips {
				if old.AllocationUUID == "" {
					u, err := uuid.NewRandom()
					if err != nil {
						return err
					}
					new := old
					new.AllocationUUID = u.String()
					err = rs.updateEntity(rs.ipTable(), &new, &old)
					if err != nil {
						return err
					}
				}
			}
			return nil
		},
	},
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

func (rs *RethinkStore) migrate() error {
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
	if len(ms) > 0 {
		rs.SugaredLogger.Infow("database migration required", "current-version", current.Version, "newer-versions", len(ms), "target-version", ms[len(ms)-1].Version)
	} else {
		rs.SugaredLogger.Infow("no database migration required", "current-version", current.Version)
	}

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

	return nil
}
