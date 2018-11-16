package service

import (
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"github.com/inconshreveable/log15"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

var (
	testlogger = log15.New()
)

func init() {
	testlogger.SetHandler(log15.DiscardHandler())
}

func initMockDB() (*datastore.RethinkStore, *r.Mock) {
	rs := datastore.New(
		testlogger,
		"db-addr",
		"mockdb",
		"db-user",
		"db-password",
	)
	mock := rs.Mock()

	return rs, mock
}
