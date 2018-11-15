package service

import (
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore/rethinkstore"
	"github.com/inconshreveable/log15"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

var (
	testlogger = log15.New()
)

func init() {
	testlogger.SetHandler(log15.DiscardHandler())
}

func initMockDB() (datastore.Datastore, *r.Mock) {
	rs := rethinkstore.New(
		testlogger,
		"db-addr",
		"mockdb",
		"db-user",
		"db-password",
	)
	mock := rs.Mock()

	return rs, mock
}
