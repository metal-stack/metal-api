package datastore

import (
	"context"
	"testing"

	"git.f-i-ts.de/cloud-native/metallib/zapup"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/testdata"
	"github.com/stretchr/testify/assert"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
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

// InitTestDB create a docker container whith rethinkdb for real integration tests.
func InitTestDB(t *testing.T) (*RethinkStore, testcontainers.Container, context.Context) {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "rethinkdb",
		ExposedPorts: []string{"28015/tcp"},
		WaitingFor:   wait.ForLog("Server ready"),
	}
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	assert.NoError(t, err)
	ip, err := c.Host(ctx)
	assert.NoError(t, err)
	port, err := c.MappedPort(ctx, "28015/tcp")
	assert.NoError(t, err)
	rs := &RethinkStore{
		SugaredLogger: zapup.MustRootLogger().Sugar(),
		dbhost:        ip + ":" + port.Port(),
		dbname:        "testdb",
		dbuser:        "",
		dbpass:        "",
	}
	IntegerPoolRangeMin = 10000
	IntegerPoolRangeMax = 10010
	err = rs.Connect()
	assert.NoError(t, err)
	return rs, c, ctx
}
