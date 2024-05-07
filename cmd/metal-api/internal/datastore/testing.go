package datastore

import (
	"log/slog"
	"testing"

	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
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
func InitMockDB(t *testing.T) (*RethinkStore, *r.Mock) {
	rs := New(
		slog.Default(),
		"db-addr",
		"mockdb",
		"db-user",
		"db-password",
	)
	mock := rs.Mock()
	return rs, mock
}
