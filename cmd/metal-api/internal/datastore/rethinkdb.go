package datastore

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

var (
	tables = []string{"image", "size", "partition", "device", "switch", "wait", "ipmi", "vrf"}
)

// A RethinkStore is the database access layer for rethinkdb.
type RethinkStore struct {
	*zap.SugaredLogger
	session   r.QueryExecutor
	dbsession *r.Session
	database  *r.Term

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

func (rs *RethinkStore) initializeTables(opts r.TableCreateOpts) error {
	db := rs.db()

	return multi(rs.session,
		// create our tables
		r.Expr(tables).Difference(db.TableList()).ForEach(func(r r.Term) r.Term {
			return db.TableCreate(r, opts)
		}),
		// create indices
		db.Table("device").IndexList().Contains("project").Do(func(i r.Term) r.Term {
			return r.Branch(i, nil, db.Table("device").IndexCreate("project"))
		}),
	)
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
func (rs *RethinkStore) deviceTable() *r.Term {
	res := r.DB(rs.dbname).Table("device")
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
func (rs *RethinkStore) ipmiTable() *r.Term {
	res := r.DB(rs.dbname).Table("ipmi")
	return &res
}
func (rs *RethinkStore) vrfTable() *r.Term {
	res := r.DB(rs.dbname).Table("vrf")
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
	rs.Info("Rethinkstore disconnected")
	return nil
}

// Connect connects to the database. If there is an error, it will run until there is
// a connection.
func (rs *RethinkStore) Connect() error {
	rs.database, rs.dbsession = retryConnect(rs.SugaredLogger, []string{rs.dbhost}, rs.dbname, rs.dbuser, rs.dbpass)
	rs.Info("Rethinkstore connected")
	rs.session = rs.dbsession
	err := rs.initializeTables(r.TableCreateOpts{Shards: 1, Replicas: 1})
	return err
}

func connect(hosts []string, dbname, user, pwd string) (*r.Term, *r.Session, error) {
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
		return nil, nil, fmt.Errorf("cannot connect to DB: %v", err)
	}

	err = r.DBList().Contains(dbname).Do(func(row r.Term) r.Term {
		return r.Branch(row, nil, r.DBCreate(dbname))
	}).Exec(session)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create database: %v", err)
	}

	db := r.DB(dbname)
	return &db, session, nil
}

// mustConnect versucht eine DB Verbindung herszustellen. Wenn es nicht
// funktioniert kommt es zu einem panic.
func mustConnect(hosts []string, dbname, username, pwd string) (*r.Term, *r.Session) {
	d, s, e := connect(hosts, dbname, username, pwd)
	if e != nil {
		panic(e)
	}
	return d, s
}

// retryConnect versucht endlos eine Verbindung zur DB herzustellen. Wenn
// die Verbindung nicht klappt wird eine zeit lang gewartet und erneut
// versucht.
func retryConnect(log *zap.SugaredLogger, hosts []string, dbname, user, pwd string) (*r.Term, *r.Session) {
tryAgain:
	db, s, err := connect(hosts, dbname, user, pwd)
	if err != nil {
		log.Errorw("db connection error", "db", dbname, "hosts", hosts, "error", err)
		time.Sleep(3 * time.Second)
		goto tryAgain
	}
	return db, s
}
