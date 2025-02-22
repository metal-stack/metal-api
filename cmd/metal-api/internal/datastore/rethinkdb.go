package datastore

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"

	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

const (
	DemotedUser                       = "metal"
	entityAlreadyModifiedErrorMessage = "the entity was changed from another, please retry"
)

var tables = []string{
	ASNIntegerPool.String(), ASNIntegerPool.String() + "info",
	"event",
	"filesystemlayout",
	"image",
	"ip",
	"machine",
	"migration",
	"network",
	"partition",
	"sharedmutex",
	"size",
	"sizeimageconstraint",
	"sizereservation",
	"switch",
	"switchstatus",
	VRFIntegerPool.String(), VRFIntegerPool.String() + "info",
}

// A RethinkStore is the database access layer for rethinkdb.
type RethinkStore struct {
	log *slog.Logger

	session   r.QueryExecutor
	dbsession *r.Session

	dbname string
	dbuser string
	dbpass string
	dbhost string

	// TODO: should not be public
	VRFPoolRangeMin uint
	VRFPoolRangeMax uint
	ASNPoolRangeMin uint
	ASNPoolRangeMax uint

	sharedMutexCtx           context.Context
	sharedMutexCancel        context.CancelFunc
	sharedMutex              *sharedMutex
	sharedMutexCheckInterval time.Duration
}

// New creates a new rethink store.
func New(log *slog.Logger, dbhost string, dbname string, dbuser string, dbpass string) *RethinkStore {
	return &RethinkStore{
		log:    log,
		dbhost: dbhost,
		dbname: dbname,
		dbuser: dbuser,
		dbpass: dbpass,

		VRFPoolRangeMin: DefaultVRFPoolRangeMin,
		VRFPoolRangeMax: DefaultVRFPoolRangeMax,
		ASNPoolRangeMin: DefaultASNPoolRangeMin,
		ASNPoolRangeMax: DefaultASNPoolRangeMax,

		sharedMutexCheckInterval: defaultSharedMutexCheckInterval,
	}
}

// Session exported for migration unit test
func (rs *RethinkStore) Session() r.QueryExecutor {
	return rs.session
}

func multi(session r.QueryExecutor, tt ...r.Term) error {
	for _, t := range tt {
		if err := t.Exec(session); err != nil {
			return err
		}
	}
	return nil
}

// Initialize initializes the database, it should be called before serving the metal-api
// in order to ensure that tables, pools, permissions are properly initialized
func (rs *RethinkStore) Initialize() error {
	return rs.initializeTables(r.TableCreateOpts{Shards: 1, Replicas: 1})
}

func (rs *RethinkStore) initializeTables(opts r.TableCreateOpts) error {
	rs.log.Info("starting database init")

	db := rs.db()

	err := multi(rs.session,
		// rename old integerpool to vrfpool
		// FIXME enable and remove once migrated
		// db.TableList().Contains("integerpool").Do(func(r r.Term) r.Term {
		// 	return db.Table("integerpool").Config().Update(map[string]interface{}{"name": VRFIntegerPoolName})
		// }),
		// db.TableList().Contains("integerpoolinfo").Do(func(r r.Term) r.Term {
		// 	return db.Table("integerpoolinfo").Config().Update(map[string]interface{}{"name": VRFIntegerPoolName + "info"})
		// }),
		// create our tables
		r.Expr(tables).Difference(db.TableList()).ForEach(func(r r.Term) r.Term {
			return db.TableCreate(r, opts)
		}),
		// create indices
		db.Table("machine").IndexList().Contains("project").Do(func(i r.Term) r.Term {
			return r.Branch(i, nil, db.Table("machine").IndexCreate("project"))
		}),
	)
	if err != nil {
		return err
	}

	// be graceful after table creation and wait until ready
	res, err := db.Wait().Run(rs.session)
	if err != nil {
		return err
	}
	defer res.Close()

	// demoted runtime user creation / update
	rs.log.Info("creating / updating demoted runtime user")
	_, err = rs.userTable().Insert(map[string]interface{}{"id": DemotedUser, "password": rs.dbpass}, r.InsertOpts{
		Conflict: "update",
	}).RunWrite(rs.session)
	if err != nil {
		return err
	}
	rs.log.Info("ensuring demoted user can read and write")
	_, err = rs.db().Grant(DemotedUser, map[string]interface{}{"read": true, "write": true}).RunWrite(rs.session)
	if err != nil {
		return err
	}
	_, err = r.DB("rethinkdb").Grant(DemotedUser, map[string]interface{}{"read": true}).RunWrite(rs.session)
	if err != nil {
		return err
	}

	// integer pools
	err = rs.GetVRFPool().initIntegerPool(rs.log)
	if err != nil {
		return err
	}

	err = rs.GetASNPool().initIntegerPool(rs.log)
	if err != nil {
		return err
	}

	rs.log.Info("database init complete")

	return nil
}

func (rs *RethinkStore) sizeTable() *r.Term {
	res := r.DB(rs.dbname).Table("size")
	return &res
}

func (rs *RethinkStore) imageTable() *r.Term {
	res := r.DB(rs.dbname).Table("image")
	return &res
}

func (rs *RethinkStore) partitionTable() *r.Term {
	res := r.DB(rs.dbname).Table("partition")
	return &res
}

func (rs *RethinkStore) machineTable() *r.Term {
	res := r.DB(rs.dbname).Table("machine")
	return &res
}

func (rs *RethinkStore) switchTable() *r.Term {
	res := r.DB(rs.dbname).Table("switch")
	return &res
}
func (rs *RethinkStore) switchStatusTable() *r.Term {
	res := r.DB(rs.dbname).Table("switchstatus")
	return &res
}
func (rs *RethinkStore) eventTable() *r.Term {
	res := r.DB(rs.dbname).Table("event")
	return &res
}

func (rs *RethinkStore) networkTable() *r.Term {
	res := r.DB(rs.dbname).Table("network")
	return &res
}

func (rs *RethinkStore) ipTable() *r.Term {
	res := r.DB(rs.dbname).Table("ip")
	return &res
}

func (rs *RethinkStore) filesystemLayoutTable() *r.Term {
	res := r.DB(rs.dbname).Table("filesystemlayout")
	return &res
}

func (rs *RethinkStore) sizeImageConstraintTable() *r.Term {
	res := r.DB(rs.dbname).Table("sizeimageconstraint")
	return &res
}

func (rs *RethinkStore) sizeReservationTable() *r.Term {
	res := r.DB(rs.dbname).Table("sizereservation")
	return &res
}

func (rs *RethinkStore) asnTable() *r.Term {
	res := r.DB(rs.dbname).Table(ASNIntegerPool.String())
	return &res
}

func (rs *RethinkStore) asnInfoTable() *r.Term {
	res := r.DB(rs.dbname).Table(ASNIntegerPool.String() + "info")
	return &res
}

func (rs *RethinkStore) vrfTable() *r.Term {
	res := r.DB(rs.dbname).Table(VRFIntegerPool.String())
	return &res
}

func (rs *RethinkStore) vrfInfoTable() *r.Term {
	res := r.DB(rs.dbname).Table(VRFIntegerPool.String() + "info")
	return &res
}

func (rs *RethinkStore) migrationTable() *r.Term {
	res := r.DB(rs.dbname).Table("migration")
	return &res
}

func (rs *RethinkStore) userTable() *r.Term {
	res := r.DB("rethinkdb").Table("users")
	return &res
}

func (rs *RethinkStore) db() *r.Term {
	res := r.DB(rs.dbname)
	return &res
}

// Mock return the mock from the rethinkdb driver and sets the
// session to this mock. This MUST NOT be called in productive code.
func (rs *RethinkStore) Mock() *r.Mock {
	m := r.NewMock()
	rs.session = m
	return m
}

// Close closes the database session.
func (rs *RethinkStore) Close() error {
	if rs.dbsession != nil {
		err := rs.dbsession.Close()
		if err != nil {
			return err
		}
	}

	if rs.sharedMutexCancel != nil {
		rs.sharedMutexCancel()
	}

	rs.log.Info("Rethinkstore disconnected")

	return nil
}

// Connect connects to the database. If there is an error, it will run until there is
// a connection.
func (rs *RethinkStore) Connect() error {
	rs.dbsession = retryConnect(rs.log, []string{rs.dbhost}, rs.dbname, rs.dbuser, rs.dbpass)
	rs.log.Info("Rethinkstore connected")
	rs.session = rs.dbsession
	rs.sharedMutexCtx, rs.sharedMutexCancel = context.WithCancel(context.Background())
	var err error
	rs.sharedMutex, err = newSharedMutex(rs.sharedMutexCtx, rs.log, rs.dbsession, newMutexOptCheckInterval(rs.sharedMutexCheckInterval))
	if err != nil {
		return err
	}

	return nil
}

// Demote connects to the database with the demoted metal runtime user. this enables
// putting the database in read-only mode during database migrations
func (rs *RethinkStore) Demote() error {
	rs.log.Info("connecting with demoted runtime user")
	err := rs.Close()
	if err != nil {
		return err
	}

	rs.dbsession = retryConnect(rs.log, []string{rs.dbhost}, rs.dbname, DemotedUser, rs.dbpass)
	rs.session = rs.dbsession
	rs.sharedMutexCtx, rs.sharedMutexCancel = context.WithCancel(context.Background())
	rs.sharedMutex, err = newSharedMutex(rs.sharedMutexCtx, rs.log, rs.dbsession, newMutexOptCheckInterval(rs.sharedMutexCheckInterval))
	if err != nil {
		return err
	}

	rs.log.Info("rethinkstore connected with demoted user")
	return nil
}

func connect(hosts []string, dbname, user, pwd string) (*r.Session, error) {
	var err error
	session, err := r.Connect(r.ConnectOpts{
		Addresses: hosts,
		Database:  dbname,
		Username:  user,
		Password:  pwd,
		MaxIdle:   10,
		MaxOpen:   20,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot connect to DB: %w", err)
	}

	err = r.DBList().Contains(dbname).Do(func(row r.Term) r.Term {
		return r.Branch(row, nil, r.DBCreate(dbname))
	}).Exec(session)
	if err != nil {
		return nil, fmt.Errorf("cannot create database: %w", err)
	}

	return session, nil
}

// retryConnect infinitely tries to establish a database connection.
// in case a connection could not be established, the function will
// wait for a short period of time and try again.
func retryConnect(log *slog.Logger, hosts []string, dbname, user, pwd string) *r.Session {
tryAgain:
	s, err := connect(hosts, dbname, user, pwd)
	if err != nil {
		log.Error("db connection error", "db", dbname, "hosts", hosts, "error", err)
		time.Sleep(3 * time.Second)
		goto tryAgain
	}
	return s
}

func (rs *RethinkStore) findEntityByID(table *r.Term, entity interface{}, id string) error {
	res, err := table.Get(id).Run(rs.session)
	if err != nil {
		return fmt.Errorf("cannot find %v with id %q in database: %w", getEntityName(entity), id, err)
	}
	defer res.Close()
	if res.IsNil() {
		return metal.NotFound("no %v with id %q found", getEntityName(entity), id)
	}
	err = res.One(entity)
	if err != nil {
		return fmt.Errorf("more than one %v with same id exists: %w", getEntityName(entity), err)
	}
	return nil
}

func (rs *RethinkStore) findEntity(query *r.Term, entity interface{}) error {
	res, err := query.Run(rs.session)
	if err != nil {
		return fmt.Errorf("cannot find %v in database: %w", getEntityName(entity), err)
	}
	defer res.Close()
	if res.IsNil() {
		return metal.NotFound("no %v found", getEntityName(entity))
	}

	hasResult := res.Next(entity)
	if !hasResult {
		return fmt.Errorf("cannot find %v in database: %w", getEntityName(entity), err)
	}

	next := map[string]interface{}{}
	hasResult = res.Next(&next)
	if hasResult {
		return fmt.Errorf("more than one %v exists", getEntityName(entity))
	}

	return nil
}

func (rs *RethinkStore) searchEntities(query *r.Term, entity interface{}) error {
	res, err := query.Run(rs.session)
	if err != nil {
		return fmt.Errorf("cannot search %v in database: %w", getEntityName(entity), err)
	}
	defer res.Close()

	err = res.All(entity)
	if err != nil {
		return fmt.Errorf("cannot fetch all entities: %w", err)
	}
	return nil
}

func (rs *RethinkStore) listEntities(table *r.Term, entity interface{}) error {
	res, err := table.Run(rs.session)
	if err != nil {
		return fmt.Errorf("cannot list %v from database: %w", getEntityName(entity), err)
	}
	defer res.Close()

	err = res.All(entity)
	if err != nil {
		return fmt.Errorf("cannot fetch all entities: %w", err)
	}
	return nil
}

func (rs *RethinkStore) createEntity(table *r.Term, entity metal.Entity) error {
	now := time.Now()
	entity.SetCreated(now)
	entity.SetChanged(now)

	res, err := table.Insert(entity).RunWrite(rs.session)
	if err != nil {
		if r.IsConflictErr(err) {
			return metal.Conflict("cannot create %v in database, entity already exists: %s", getEntityName(entity), entity.GetID())
		}
		return fmt.Errorf("cannot create %v in database: %w", getEntityName(entity), err)
	}

	if entity.GetID() == "" && len(res.GeneratedKeys) > 0 {
		entity.SetID(res.GeneratedKeys[0])
	}

	return nil
}

func (rs *RethinkStore) upsertEntity(table *r.Term, entity metal.Entity) error {
	now := time.Now()
	if entity.GetCreated().IsZero() {
		entity.SetCreated(now)
	}
	entity.SetChanged(now)

	res, err := table.Insert(entity, r.InsertOpts{
		Conflict: "replace",
	}).RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot upsert %v (%s) in database: %w", getEntityName(entity), entity.GetID(), err)
	}

	if entity.GetID() == "" && len(res.GeneratedKeys) > 0 {
		entity.SetID(res.GeneratedKeys[0])
	}
	return nil
}

func (rs *RethinkStore) deleteEntity(table *r.Term, entity metal.Entity) error {
	_, err := table.Get(entity.GetID()).Delete().RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot delete %v with id %q from database: %w", getEntityName(entity), entity.GetID(), err)
	}
	return nil
}

func (rs *RethinkStore) updateEntity(table *r.Term, newEntity metal.Entity, oldEntity metal.Entity) error {
	newEntity.SetChanged(time.Now())

	_, err := table.Get(oldEntity.GetID()).Replace(func(row r.Term) r.Term {
		return r.Branch(row.Field("changed").Eq(r.Expr(oldEntity.GetChanged())), newEntity, r.Error(entityAlreadyModifiedErrorMessage))
	}).RunWrite(rs.session)
	if err != nil {
		if strings.Contains(err.Error(), entityAlreadyModifiedErrorMessage) {
			return metal.Conflict("cannot update %v (%s): %s", getEntityName(newEntity), oldEntity.GetID(), entityAlreadyModifiedErrorMessage)
		}
		return fmt.Errorf("cannot update %v (%s): %w", getEntityName(newEntity), oldEntity.GetID(), err)
	}

	return nil
}

func getEntityName(entity interface{}) string {
	t := reflect.TypeOf(entity)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return strings.ToLower(t.Name())
}
