package test

import (
	"context"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func init() {
	// prevent testcontainer logging mangle test and benchmark output
	// log.SetOutput(io.Discard)
}

type ConnectionDetails struct {
	Port     string
	IP       string
	DB       string
	User     string
	Password string
}

func StartRethink(t testing.TB) (container testcontainers.Container, c *ConnectionDetails, err error) {
	ctx := context.Background()
	var log testcontainers.Logging
	if t != nil {
		log = testcontainers.TestLogger(t)
	}
	req := testcontainers.ContainerRequest{
		Image:        "rethinkdb:2.4.0",
		ExposedPorts: []string{"8080/tcp", "28015/tcp"},
		Env:          map[string]string{"RETHINKDB_PASSWORD": "rethink"},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("28015/tcp"),
		),
		Cmd: []string{"rethinkdb", "--bind", "all", "--directory", "/tmp", "--initial-password", "rethink", "--io-threads", "500"},
	}
	rtContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
		Logger:           log,
	})
	if err != nil {
		panic(err.Error())
	}
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

func StartPostgres() (container testcontainers.Container, c *ConnectionDetails, err error) {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "postgres:13-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env:          map[string]string{"POSTGRES_PASSWORD": "password"},
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections"),
			wait.ForListeningPort("5432/tcp"),
		),
		Cmd: []string{"postgres", "-c", "max_connections=500"},
	}
	pgContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		panic(err.Error())
	}
	ip, err := pgContainer.Host(ctx)
	if err != nil {
		return pgContainer, nil, err
	}
	port, err := pgContainer.MappedPort(ctx, "5432")
	if err != nil {
		return pgContainer, nil, err
	}
	c = &ConnectionDetails{
		IP:       ip,
		Port:     port.Port(),
		User:     "postgres",
		DB:       "postgres",
		Password: "password",
	}

	return pgContainer, c, err
}

func StartMeilisearch(t testing.TB) (container testcontainers.Container, c *ConnectionDetails, err error) {
	meilisearchMasterKey := "meili"

	ctx := context.Background()
	var log testcontainers.Logging
	if t != nil {
		log = testcontainers.TestLogger(t)
	}

	meiliContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "getmeili/meilisearch:v1.1.0",
			ExposedPorts: []string{"7700/tcp"},
			Env: map[string]string{
				"MEILI_MASTER_KEY":   meilisearchMasterKey,
				"MEILI_NO_ANALYTICS": "true",
			},
			WaitingFor: wait.ForAll(
				wait.ForListeningPort("7700/tcp"),
			),
		},
		Started: true,
		Logger:  log,
	})
	if err != nil {
		panic(err.Error())
	}

	host, err := meiliContainer.Host(ctx)
	if err != nil {
		return meiliContainer, nil, err
	}
	port, err := meiliContainer.MappedPort(ctx, "7700")
	if err != nil {
		return meiliContainer, nil, err
	}

	conn := &ConnectionDetails{
		IP:       host,
		Port:     port.Port(),
		Password: meilisearchMasterKey,
	}

	return meiliContainer, conn, err
}
