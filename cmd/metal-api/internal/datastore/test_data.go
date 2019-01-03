package datastore

import (
	"go.uber.org/zap"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

var testlogger = zap.NewNop()

func InitMockDB() (*RethinkStore, *r.Mock) {
	rs := New(
		testlogger,
		"db-addr",
		"mockdb",
		"db-user",
		"db-password",
	)
	mock := rs.Mock()

	return rs, mock
}
