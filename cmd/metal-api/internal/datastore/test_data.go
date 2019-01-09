package datastore

import (
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

var testlogger = zap.NewNop()
var testloggerSugar = zapup.MustRootLogger().Sugar()
var (
	RethinkStore1 = RethinkStore{
		SugaredLogger: zapup.MustRootLogger().Sugar(),
		dbhost:        "dbhost",
		dbname:        "dbname",
		dbuser:        "dbuser",
		dbpass:        "password",
	}
)

/*
InitMockDB ...

Description:
This Function initializes the Mocked rethink DB.
It is recommented to execute metal.InitMockDBData() to fill it with moks

Return Values:
- RethinkStore 	// The Database
- Mock 			// The Mock endpoint (Used for mocks)
*/
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
