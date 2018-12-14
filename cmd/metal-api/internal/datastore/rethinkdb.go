package datastore

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
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

func (rs *RethinkStore) initializeTables(opts r.TableCreateOpts) {
	rs.db().TableCreate("image", opts).Exec(rs.session)
	rs.db().TableCreate("size", opts).Exec(rs.session)
	rs.db().TableCreate("site", opts).Exec(rs.session)
	rs.db().TableCreate("device", opts).Exec(rs.session)
	rs.db().TableCreate("switch", opts).Exec(rs.session)
	rs.db().TableCreate("wait", opts).Exec(rs.session)
	rs.db().TableCreate("ipmi", opts).Exec(rs.session)

	rs.deviceTable().IndexCreate("project").RunWrite(rs.session)
}

func (rs *RethinkStore) sizeTable() *r.Term {
	res := r.DB(rs.dbname).Table("size")
	return &res
}
func (rs *RethinkStore) imageTable() *r.Term {
	res := r.DB(rs.dbname).Table("image")
	return &res
}
func (rs *RethinkStore) siteTable() *r.Term {
	res := r.DB(rs.dbname).Table("site")
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
func (rs *RethinkStore) Connect() {
	rs.database, rs.dbsession = retryConnect(rs.SugaredLogger, []string{rs.dbhost}, rs.dbname, rs.dbuser, rs.dbpass)
	rs.Info("Rethinkstore connected")
	rs.session = rs.dbsession
	rs.initializeTables(r.TableCreateOpts{Shards: 1, Replicas: 1})
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
	// wenn DB schon existiert, fehler ignorieren ...
	r.DBCreate(dbname).Exec(session)
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
