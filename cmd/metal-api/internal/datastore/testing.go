package datastore

import (
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/testdata"
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
		testdata.Testlogger,
		"db-addr",
		"mockdb",
		"db-user",
		"db-password",
	)
	mock := rs.Mock()
	return rs, mock
}

var (
	rethinkStore1 = RethinkStore{
		SugaredLogger: zapup.MustRootLogger().Sugar(),
		dbhost:        "dbhost",
		dbname:        "dbname",
		dbuser:        "dbuser",
		dbpass:        "password",
	}
)
