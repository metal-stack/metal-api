package datastore

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"

	"go.uber.org/zap"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

const (
	DemotedUser = "metal"
)

var (
	tables = []string{"image", "size", "partition", "machine", "switch", "wait", "event", "network", "ip",
		"integerpool", "integerpoolinfo", "migration"}
)

// A RethinkStore is the database access layer for rethinkdb.
type RethinkStore struct {
	*zap.SugaredLogger
	session   r.QueryExecutor
	dbsession *r.Session

	dbname string
	dbuser string
	dbpass string
	dbhost string
}

// New creates a new rethink store.
func New(log *zap.Logger, dbhost string, dbname string, dbuser string, dbpass string) *RethinkStore {
	return &RethinkStore{
		SugaredLogger: log.Sugar(),
		dbhost:        dbhost,
		dbname:        dbname,
		dbuser:        dbuser,
		dbpass:        dbpass,
	}
}

func multi(session r.QueryExecutor, tt ...r.Term) error {
	for _, t := range tt {
		if err := t.Exec(session); err != nil {
			return err
		}
	}
	return nil
}

// Health checks if the connection to the database is ok.
func (rs *RethinkStore) Health() error {
	return multi(rs.session,
		r.Branch(
			rs.db().TableList().Difference(r.Expr(tables)).Count().Eq(0),
			r.Expr(true),
			r.Error("too many tables in DB")),
		r.Branch(
			r.Expr(tables).Difference(rs.db().TableList()).Count().Eq(0),
			r.Expr(true),
			r.Error("too less tables in DB")),
	)
}

// Initialize initializes the database, it should be called every time
// the application comes up before using the data store
func (rs *RethinkStore) Initialize() error {
	return rs.initializeTables(r.TableCreateOpts{Shards: 1, Replicas: 1})
}

func (rs *RethinkStore) initializeTables(opts r.TableCreateOpts) error {
	db := rs.db()

	err := multi(rs.session,
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

	_, err = rs.userTable().Insert(map[string]interface{}{"id": DemotedUser, "password": rs.dbpass}, r.InsertOpts{
		Conflict: "replace",
	}).RunWrite(rs.session)
	if err != nil {
		return err
	}

	err = rs.initIntegerPool()
	if err != nil {
		return err
	}

	rs.SugaredLogger.Infow("tables successfully initialized")

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
func (rs *RethinkStore) migrationTable() *r.Term {
	res := r.DB(rs.dbname).Table("migration")
	return &res
}
func (rs *RethinkStore) switchTable() *r.Term {
	res := r.DB(rs.dbname).Table("switch")
	return &res
}
func (rs *RethinkStore) waitTable() *r.Term {
	res := r.DB(rs.dbname).Table("wait")
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
func (rs *RethinkStore) integerTable() *r.Term {
	res := r.DB(rs.dbname).Table("integerpool")
	return &res
}
func (rs *RethinkStore) integerInfoTable() *r.Term {
	res := r.DB(rs.dbname).Table("integerpoolinfo")
	return &res
}
func (rs *RethinkStore) db() *r.Term {
	res := r.DB(rs.dbname)
	return &res
}
func (rs *RethinkStore) userTable() *r.Term {
	res := r.DB("rethinkdb").Table("users")
	return &res
}
func (rs *RethinkStore) statsTable() *r.Term {
	res := r.DB("rethinkdb").Table("stats")
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
	rs.Info("Rethinkstore disconnected")
	return nil
}

// Connect connects to the database. If there is an error, it will run until there is
// a connection.
func (rs *RethinkStore) Connect() error {
	rs.dbsession = retryConnect(rs.SugaredLogger, []string{rs.dbhost}, rs.dbname, rs.dbuser, rs.dbpass)
	rs.Info("Rethinkstore connected")
	rs.session = rs.dbsession
	return nil
}

// Demote connects to the database with the demoted metal runtime user. this enables
// putting the database in read-only mode during database migrations
func (rs *RethinkStore) Demote() error {
	rs.Info("Connecting with demoted runtime user")
	err := rs.Close()
	if err != nil {
		return err
	}
	rs.dbsession = retryConnect(rs.SugaredLogger, []string{rs.dbhost}, rs.dbname, DemotedUser, rs.dbpass)
	rs.Info("Rethinkstore connected with demoted user")
	rs.session = rs.dbsession
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
		return nil, fmt.Errorf("cannot connect to DB: %v", err)
	}

	err = r.DBList().Contains(dbname).Do(func(row r.Term) r.Term {
		return r.Branch(row, nil, r.DBCreate(dbname))
	}).Exec(session)
	if err != nil {
		return nil, fmt.Errorf("cannot create database: %v", err)
	}

	return session, nil
}

// retryConnect infinitely tries to establish a database connection.
// in case a connection could not be established, the function will
// wait for a short period of time and try again.
func retryConnect(log *zap.SugaredLogger, hosts []string, dbname, user, pwd string) *r.Session {
tryAgain:
	s, err := connect(hosts, dbname, user, pwd)
	if err != nil {
		log.Errorw("db connection error", "db", dbname, "hosts", hosts, "error", err)
		time.Sleep(3 * time.Second)
		goto tryAgain
	}
	return s
}

func (rs *RethinkStore) findEntityByID(table *r.Term, entity interface{}, id string) error {
	res, err := table.Get(id).Run(rs.session)
	if err != nil {
		return fmt.Errorf("cannot find %v with id %q in database: %v", getEntityName(entity), id, err)
	}
	defer res.Close()
	if res.IsNil() {
		return metal.NotFound("no %v with id %q found", getEntityName(entity), id)
	}
	err = res.One(entity)
	if err != nil {
		return fmt.Errorf("more than one %v with same id exists: %v", getEntityName(entity), err)
	}
	return nil
}

func (rs *RethinkStore) findEntity(query *r.Term, entity interface{}) error {
	res, err := query.Run(rs.session)
	if err != nil {
		return fmt.Errorf("cannot find %v in database: %v", getEntityName(entity), err)
	}
	defer res.Close()
	if res.IsNil() {
		return metal.NotFound("no %v with found", getEntityName(entity))
	}

	hasResult := res.Next(entity)
	if !hasResult {
		return fmt.Errorf("cannot find %v in database: %v", getEntityName(entity), err)
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
		return fmt.Errorf("cannot search %v in database: %v", getEntityName(entity), err)
	}
	defer res.Close()

	err = res.All(entity)
	if err != nil {
		return fmt.Errorf("cannot fetch all entities: %v", err)
	}
	return nil
}

func (rs *RethinkStore) listEntities(table *r.Term, entity interface{}) error {
	res, err := table.Run(rs.session)
	if err != nil {
		return fmt.Errorf("cannot list %v from database: %v", getEntityName(entity), err)
	}
	defer res.Close()

	err = res.All(entity)
	if err != nil {
		return fmt.Errorf("cannot fetch all entities: %v", err)
	}
	return nil
}

func (rs *RethinkStore) createEntity(table *r.Term, entity metal.Entity) error {
	now := time.Now()
	entity.SetCreated(now)
	entity.SetChanged(now)

	// TODO: Return metal.Conflict
	res, err := table.Insert(entity).RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot create %v in database: %v", getEntityName(entity), err)
	}

	if entity.GetID() == "" && len(res.GeneratedKeys) > 0 {
		entity.SetID(res.GeneratedKeys[0])
	}
	return nil
}

func (rs *RethinkStore) upsertEntity(table *r.Term, entity metal.Entity) error {
	now := time.Now()
	if entity.GetChanged().IsZero() {
		entity.SetChanged(now)
	}
	entity.SetChanged(now)

	res, err := table.Insert(entity, r.InsertOpts{
		Conflict: "replace",
	}).RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot upsert %v (%s) in database: %v", getEntityName(entity), entity.GetID(), err)
	}

	if entity.GetID() == "" && len(res.GeneratedKeys) > 0 {
		entity.SetID(res.GeneratedKeys[0])
	}
	return nil
}

func (rs *RethinkStore) deleteEntity(table *r.Term, entity metal.Entity) error {
	_, err := table.Get(entity.GetID()).Delete().RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot delete %v with id %q from database: %v", getEntityName(entity), entity.GetID(), err)
	}
	return nil
}

func (rs *RethinkStore) updateEntity(table *r.Term, newEntity metal.Entity, oldEntity metal.Entity) error {
	newEntity.SetChanged(time.Now())
	_, err := table.Get(oldEntity.GetID()).Replace(func(row r.Term) r.Term {
		return r.Branch(row.Field("changed").Eq(r.Expr(oldEntity.GetChanged())), newEntity, r.Error("the entity was changed from another, please retry"))
	}).RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot update %v (%s): %v", getEntityName(newEntity), oldEntity.GetID(), err)
	}
	return nil
}

func (rs *RethinkStore) listenForEntityChange(ctx context.Context, table *r.Term, entity metal.Entity, response interface{}) error {
	res, err := table.Get(entity.GetID()).Changes().Run(rs.session, r.RunOpts{Context: ctx})
	if err != nil {
		return fmt.Errorf("cannot listen for %v change with id %q in database", getEntityName(entity), entity.GetID())
	}
	defer res.Close()

	for res.Next(&response) {
		rs.SugaredLogger.Debugw("entity changed", "entity", getEntityName(entity), "id", entity.GetID())
		return nil
	}
	err = res.Err()
	if err != nil {
		return fmt.Errorf("error retrieving next %v (%s) from database: %v", getEntityName(entity), entity.GetID(), err)
	}

	return fmt.Errorf("%v (%s) database change event stream has closed without an error", getEntityName(entity), entity.GetID())
}

func getEntityName(entity interface{}) string {
	t := reflect.TypeOf(entity)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return strings.ToLower(t.Name())
}
