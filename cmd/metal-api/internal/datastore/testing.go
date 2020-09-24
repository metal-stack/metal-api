package datastore

import (
	"context"
	"testing"

	"github.com/metal-stack/metal-lib/zapup"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
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
	vrfterm := rs.integerTable(VRFIntegerPool.String())
	asnterm := rs.integerTable(VRFIntegerPool.String())
	vrfPool := IntegerPool{tablename: VRFIntegerPool.String(), min: VRFPoolRangeMin, max: VRFPoolRangeMax, term: vrfterm, session: rs.session}
	asnPool := IntegerPool{tablename: ASNIntegerPool.String(), min: ASNPoolRangeMin, max: ASNPoolRangeMax, term: asnterm, session: rs.session}
	rs.integerPools[VRFIntegerPool] = &vrfPool
	rs.integerPools[ASNIntegerPool] = &asnPool
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
	VRFPoolRangeMin = 10000
	VRFPoolRangeMax = 10010
	err = rs.Connect()
	assert.NoError(t, err)
	return rs, c, ctx
}
