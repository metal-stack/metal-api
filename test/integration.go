package test

import (
	"context"
	"sync"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	rtOnce      sync.Once
	rtContainer testcontainers.Container
)

func init() {
	// prevent testcontainer logging mangle test and benchmark output
	// log.SetOutput(ioutil.Discard)
}

type ConnectionDetails struct {
	Port     string
	IP       string
	DB       string
	User     string
	Password string
}

func StartRethink() (container testcontainers.Container, c *ConnectionDetails, err error) {
	ctx := context.Background()
	rtOnce.Do(func() {
		var err error
		req := testcontainers.ContainerRequest{
			Image:        "rethinkdb:2.4.0",
			ExposedPorts: []string{"8080/tcp", "28015/tcp"},
			Env:          map[string]string{"RETHINKDB_PASSWORD": "rethink"},
			WaitingFor: wait.ForAll(
				wait.ForListeningPort("8080/tcp"),
			),
			Cmd: []string{"rethinkdb", "--bind", "all", "--directory", "/tmp", "--initial-password", "rethink"},
		}
		rtContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		if err != nil {
			panic(err.Error())
		}
	})
	ip, err := rtContainer.Host(ctx)
	if err != nil {
		return rtContainer, nil, err
	}
	port, err := rtContainer.MappedPort(ctx, "28015")
	if err != nil {
		return rtContainer, nil, err
	}

	c = &ConnectionDetails{
		IP:       ip,
		Port:     port.Port(),
		User:     "admin",
		DB:       "metal",
		Password: "rethink",
	}

	return rtContainer, c, err
}
