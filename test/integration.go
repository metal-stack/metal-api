package test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/metal-stack/metal-lib/bus"
	"github.com/stretchr/testify/require"
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
		Image:        "rethinkdb:2.4.4-bookworm-slim",
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
		Image:        "postgres:16-alpine",
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
			Image:        "getmeili/meilisearch:v1.3.4",
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

func StartNsqd(t *testing.T, log *slog.Logger) (testcontainers.Container, bus.Publisher, *bus.Consumer) {
	ctx := context.Background()

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "nsqio/nsq:v1.3.0",
			ExposedPorts: []string{"4150/tcp", "4151/tcp"},
			WaitingFor: wait.ForAll(
				wait.ForListeningPort("4150/tcp"),
				wait.ForListeningPort("4151/tcp"),
			),
			Cmd: []string{"nsqd"},
		},
		Started: true,
		Logger:  testcontainers.TestLogger(t),
	})
	require.NoError(t, err)

	ip, err := c.Host(ctx)
	require.NoError(t, err)

	tcpPort, err := c.MappedPort(ctx, "4150")
	require.NoError(t, err)
	httpPort, err := c.MappedPort(ctx, "4151")
	require.NoError(t, err)

	consumer, err := bus.NewConsumer(log, nil)
	require.NoError(t, err)

	tcpAddress := fmt.Sprintf("%s:%d", ip, tcpPort.Int())
	httpAddress := fmt.Sprintf("%s:%d", ip, httpPort.Int())

	consumer.With(bus.NSQDs(tcpAddress))

	publisher, err := bus.NewPublisher(log, &bus.PublisherConfig{
		TCPAddress:   tcpAddress,
		HTTPEndpoint: httpAddress,
	})
	require.NoError(t, err)

	return c, publisher, consumer
}
