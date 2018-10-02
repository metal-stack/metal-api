package rethinkstore

import (
	"fmt"
	"time"

	"github.com/inconshreveable/log15"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

type RethinkStore struct {
	log15.Logger
	session       *r.Session
	database      *r.Term
	imageTable    r.Term
	sizeTable     r.Term
	facilityTable r.Term
	deviceTable   r.Term
	waitTable     r.Term
}

func New(log log15.Logger, host, dbname string) (*RethinkStore, error) {
	db, s := RetryConnect([]string{host}, dbname, "", "")
	res := &RethinkStore{Logger: log, database: db, session: s}
	res.initializeTables(r.TableCreateOpts{Shards: 1, Replicas: 1})
	return res, nil
}

func (rs *RethinkStore) initializeTables(opts r.TableCreateOpts) {
	rs.database.TableCreate("image", opts).Exec(rs.session)
	rs.database.TableCreate("size", opts).Exec(rs.session)
	rs.database.TableCreate("facility", opts).Exec(rs.session)
	rs.database.TableCreate("device", opts).Exec(rs.session)
	rs.database.TableCreate("wait", opts).Exec(rs.session)

	rs.imageTable = rs.database.Table("image")
	rs.sizeTable = rs.database.Table("size")
	rs.facilityTable = rs.database.Table("facility")
	rs.waitTable = rs.database.Table("wait")
	rs.deviceTable = rs.database.Table("device")
	rs.deviceTable.IndexCreate("project").RunWrite(rs.session)
}

func (rs *RethinkStore) Close() {
	log15.Info("Rethinkstore disconnected")
}

func (rs *RethinkStore) Connect() {
	log15.Info("Rethinkstore connected")
}

func Connect(hosts []string, dbname, user, pwd string) (*r.Term, *r.Session, error) {
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

// MustConnect versucht eine DB Verbindung herszustellen. Wenn es nicht
// funktioniert kommt es zu einem panic.
func MustConnect(hosts []string, dbname, username, pwd string) (*r.Term, *r.Session) {
	d, s, e := Connect(hosts, dbname, username, pwd)
	if e != nil {
		panic(e)
	}
	return d, s
}

// RetryConnect versucht endlos eine Verbindung zur DB herzustellen. Wenn
// die Verbindung nicht klappt wird eine zeit lange gewartet und erneut
// versucht.
func RetryConnect(hosts []string, dbname, user, pwd string) (*r.Term, *r.Session) {
tryAgain:
	db, s, err := Connect(hosts, dbname, user, pwd)
	if err != nil {
		log15.Error("db connection error", "db", dbname, "hosts", hosts, "error", err)
		time.Sleep(3 * time.Second)
		goto tryAgain
	}
	return db, s
}
