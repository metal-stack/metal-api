package datastore

import (
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
)

/*
InitMockDB ...

Description:
This Function initializes the Mocked rethink DB.
It is recommended to execute metal.InitMockDBData() to fill it with mocks

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
