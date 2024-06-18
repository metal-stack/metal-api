package generic

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

const entityAlreadyModifiedErrorMessage = "the entity was changed from another, please retry"

type EntityQuery interface {
	Query(q r.Term) r.Term
}

type Storage[E metal.Entity] interface {
	Create(ctx context.Context, e E) error
	Update(ctx context.Context, new, old E) error
	Upsert(ctx context.Context, e E) error
	Delete(ctx context.Context, e E) error
	Get(ctx context.Context, id string) (E, error)
	Find(ctx context.Context, query EntityQuery) (E, error)
	Search(ctx context.Context, query EntityQuery) ([]E, error)
	List(ctx context.Context) ([]E, error)
}

type rethinkStore[E metal.Entity] struct {
	log           *slog.Logger
	queryExecutor r.QueryExecutor
	dbname        string
	table         r.Term
	tableName     string
}

type Datastore struct {
	event               Storage[*metal.ProvisioningEventContainer]
	filesystemlayout    Storage[*metal.FilesystemLayout]
	image               Storage[*metal.Image]
	ip                  Storage[*metal.IP]
	machine             Storage[*metal.Machine]
	network             Storage[*metal.Network]
	partition           Storage[*metal.Partition]
	size                Storage[*metal.Size]
	sizeimageConstraint Storage[*metal.SizeImageConstraint]
	sw                  Storage[*metal.Switch]
	switchStatus        Storage[*metal.SwitchStatus]
}

func New(log *slog.Logger, dbname string, queryExecutor r.QueryExecutor) *Datastore {
	return &Datastore{
		event:               newStorage[*metal.ProvisioningEventContainer](log, dbname, "event", queryExecutor),
		filesystemlayout:    newStorage[*metal.FilesystemLayout](log, dbname, "filesystemlayout", queryExecutor),
		image:               newStorage[*metal.Image](log, dbname, "image", queryExecutor),
		ip:                  newStorage[*metal.IP](log, dbname, "ip", queryExecutor),
		machine:             newStorage[*metal.Machine](log, dbname, "machine", queryExecutor),
		network:             newStorage[*metal.Network](log, dbname, "network", queryExecutor),
		partition:           newStorage[*metal.Partition](log, dbname, "partition", queryExecutor),
		size:                newStorage[*metal.Size](log, dbname, "size", queryExecutor),
		sizeimageConstraint: newStorage[*metal.SizeImageConstraint](log, dbname, "sizeimageconstraint", queryExecutor),
		sw:                  newStorage[*metal.Switch](log, dbname, "switch", queryExecutor),
		switchStatus:        newStorage[*metal.SwitchStatus](log, dbname, "switchstatus", queryExecutor),
	}
}

func (d *Datastore) Event() Storage[*metal.ProvisioningEventContainer] {
	return d.event
}

func (d *Datastore) FilesystemLayout() Storage[*metal.FilesystemLayout] {
	return d.filesystemlayout
}
func (d *Datastore) Image() Storage[*metal.Image] {
	return d.image
}
func (d *Datastore) IP() Storage[*metal.IP] {
	return d.ip
}
func (d *Datastore) Machine() Storage[*metal.Machine] {
	return d.machine
}
func (d *Datastore) Network() Storage[*metal.Network] {
	return d.network
}
func (d *Datastore) Partition() Storage[*metal.Partition] {
	return d.partition
}
func (d *Datastore) Size() Storage[*metal.Size] {
	return d.size
}
func (d *Datastore) SizeImageConstraint() Storage[*metal.SizeImageConstraint] {
	return d.sizeimageConstraint
}
func (d *Datastore) Switch() Storage[*metal.Switch] {
	return d.sw
}
func (d *Datastore) SwitchStatus() Storage[*metal.SwitchStatus] {
	return d.switchStatus
}

// newStorage creates a new Storage which uses the given database abstraction.
func newStorage[E metal.Entity](log *slog.Logger, dbname, tableName string, queryExecutor r.QueryExecutor) Storage[E] {
	ds := &rethinkStore[E]{
		log:           log,
		queryExecutor: queryExecutor,
		dbname:        dbname,
		table:         r.DB(dbname).Table(tableName),
		tableName:     tableName,
	}
	return ds
}

// Create implements Storage.
func (rs *rethinkStore[E]) Create(ctx context.Context, e E) error {
	now := time.Now()
	e.SetCreated(now)
	e.SetChanged(now)

	res, err := rs.table.Insert(e).RunWrite(rs.queryExecutor, r.RunOpts{Context: ctx})
	if err != nil {
		if r.IsConflictErr(err) {
			return metal.Conflict("cannot create %v in database, entity already exists: %s", rs.tableName, e.GetID())
		}
		return fmt.Errorf("cannot create %v in database: %w", rs.tableName, err)
	}

	if e.GetID() == "" && len(res.GeneratedKeys) > 0 {
		e.SetID(res.GeneratedKeys[0])
	}

	return nil
}

// Delete implements Storage.
func (rs *rethinkStore[E]) Delete(ctx context.Context, e E) error {
	_, err := rs.table.Get(e.GetID()).Delete().RunWrite(rs.queryExecutor, r.RunOpts{Context: ctx})
	if err != nil {
		return fmt.Errorf("cannot delete %v with id %q from database: %w", rs.tableName, e.GetID(), err)
	}
	return nil
}

// Find implements Storage.
func (rs *rethinkStore[E]) Find(ctx context.Context, query EntityQuery) (E, error) {
	var zero E
	res, err := query.Query(rs.table).Run(rs.queryExecutor, r.RunOpts{Context: ctx})
	if err != nil {
		return zero, fmt.Errorf("cannot find %v in database: %w", rs.tableName, err)
	}
	defer res.Close()
	if res.IsNil() {
		return zero, metal.NotFound("no %v with found", rs.tableName)
	}

	e := new(E)
	hasResult := res.Next(e)
	if !hasResult {
		return zero, fmt.Errorf("cannot find %v in database: %w", rs.tableName, err)
	}

	next := new(E)
	hasResult = res.Next(&next)
	if hasResult {
		return zero, fmt.Errorf("more than one %v exists", rs.tableName)
	}

	return *e, nil
}

func (rs *rethinkStore[E]) Search(ctx context.Context, query EntityQuery) ([]E, error) {
	res, err := query.Query(rs.table).Run(rs.queryExecutor, r.RunOpts{Context: ctx})
	if err != nil {
		return nil, fmt.Errorf("cannot search %v in database: %w", rs.tableName, err)
	}
	defer res.Close()

	result := new([]E)
	err = res.All(result)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch all entities: %w", err)
	}
	return *result, nil
}

func (rs *rethinkStore[E]) List(ctx context.Context) ([]E, error) {
	res, err := rs.table.Run(rs.queryExecutor, r.RunOpts{Context: ctx})
	if err != nil {
		return nil, fmt.Errorf("cannot list %v from database: %w", rs.tableName, err)
	}
	defer res.Close()

	result := new([]E)
	err = res.All(result)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch all entities: %w", err)
	}
	return *result, nil
}

// Get implements Storage.
func (rs *rethinkStore[E]) Get(ctx context.Context, id string) (E, error) {
	var zero E
	res, err := rs.table.Get(id).Run(rs.queryExecutor, r.RunOpts{Context: ctx})
	if err != nil {
		return zero, fmt.Errorf("cannot find %v with id %q in database: %w", rs.tableName, id, err)
	}
	defer res.Close()
	if res.IsNil() {
		return zero, metal.NotFound("no %v with id %q found", rs.tableName, id)
	}
	e := new(E)
	err = res.One(e)
	if err != nil {
		return zero, fmt.Errorf("more than one %v with same id exists: %w", rs.tableName, err)
	}
	return *e, nil
}

// Update implements Storage.
func (rs *rethinkStore[E]) Update(ctx context.Context, new, old E) error {
	new.SetChanged(time.Now())

	// FIXME use context
	_, err := rs.table.Get(old.GetID()).Replace(func(row r.Term) r.Term {
		return r.Branch(row.Field("changed").Eq(r.Expr(old.GetChanged())), new, r.Error(entityAlreadyModifiedErrorMessage))
	}).RunWrite(rs.queryExecutor)
	if err != nil {
		if strings.Contains(err.Error(), entityAlreadyModifiedErrorMessage) {
			return metal.Conflict("cannot update %v (%s): %s", rs.tableName, old.GetID(), entityAlreadyModifiedErrorMessage)
		}
		return fmt.Errorf("cannot update %v (%s): %w", rs.tableName, old.GetID(), err)
	}

	return nil
}

func (rs *rethinkStore[E]) Upsert(ctx context.Context, e E) error {
	now := time.Now()
	if e.GetCreated().IsZero() {
		e.SetCreated(now)
	}
	e.SetChanged(now)

	res, err := rs.table.Insert(e, r.InsertOpts{
		Conflict: "replace",
	}).RunWrite(rs.queryExecutor)
	if err != nil {
		return fmt.Errorf("cannot upsert %v (%s) in database: %w", rs.tableName, e.GetID(), err)
	}

	if e.GetID() == "" && len(res.GeneratedKeys) > 0 {
		e.SetID(res.GeneratedKeys[0])
	}
	return nil
}
