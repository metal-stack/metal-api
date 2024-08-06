//go:build integration
// +build integration

package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	v1grpc "github.com/metal-stack/metal-api/pkg/grpc"
	"github.com/metal-stack/metal-lib/bus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	integrationtest "github.com/metal-stack/metal-api/test"
)

type (
	test struct {
		log *slog.Logger

		ss []*server
		cc []*client

		t *testing.T

		ds        *datastore.RethinkStore
		publisher bus.Publisher
		consumer  *bus.Consumer

		numberApiInstances     int
		numberMachineInstances int
		numberAllocations      int
	}

	client struct {
		machineID string
		conn      *grpc.ClientConn
		c         v1.BootServiceClient
		cancel    func()
	}

	server struct {
		cfg      *ServerConfig
		cancel   context.CancelFunc
		allocate chan string
	}
)

func TestWaitServer(t *testing.T) {
	var (
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		ctx = context.Background()
	)

	// starting a rethinkdb rethinkContainer
	rethinkContainer, c, err := integrationtest.StartRethink(t)
	require.NoError(t, err)
	defer func() {
		_ = rethinkContainer.Terminate(ctx)
	}()

	ds := datastore.New(log, c.IP+":"+c.Port, c.DB, c.User, c.Password)
	ds.VRFPoolRangeMax = 1000
	ds.ASNPoolRangeMax = 1000

	err = ds.Connect()
	require.NoError(t, err)
	err = ds.Initialize()
	require.NoError(t, err)

	// starting nsqd container
	nsqContainer, publisher, consumer := integrationtest.StartNsqd(t, slog.Default())
	require.NoError(t, err)
	defer func() {
		_ = nsqContainer.Terminate(ctx)
	}()

	te := &test{
		log:                    log,
		t:                      t,
		ds:                     ds,
		publisher:              publisher,
		consumer:               consumer,
		numberApiInstances:     3,
		numberMachineInstances: 10,
		numberAllocations:      7,
	}

	te.run(ctx)
}

func (te *test) run(ctx context.Context) {
	defer te.shutdown()

	var (
		allocations = make(chan string)
		machines    []string
	)

	te.startApiInstances(ctx)
	te.startMachineInstances(ctx, allocations)

	for i := range te.numberAllocations {
		client := te.cc[i%len(te.cc)]
		te.t.Logf("sending nsq allocation event for machine %s", client.machineID)

		err := te.publisher.Publish(metal.TopicAllocation.Name, &metal.AllocationEvent{MachineID: client.machineID})
		require.NoError(te.t, err)

		machines = append(machines, client.machineID)
	}

	waitCtx, waitCancel := context.WithTimeout(ctx, 10*time.Second)
	defer waitCancel()

	for {
		select {
		case <-waitCtx.Done():
			te.t.Errorf("not all machines left wait for allocation loop in time")
		case machineID := <-allocations:
			require.Contains(te.t, machines, machineID, "unexpected machine left wait loop")

			te.t.Logf("machine %q left wait for allocation loop", machineID)

			machines = slices.DeleteFunc(machines, func(id string) bool {
				return id == machineID
			})
		}

		if len(machines) == 0 {
			te.t.Logf("all expected machines left wait for allocation loop")
			break
		}
	}

	ms, err := te.ds.ListMachines()
	require.NoError(te.t, err)

	waiting := 0
	for _, m := range ms {
		if m.Waiting {
			waiting++
		}
	}

	assert.Equal(te.t, te.numberMachineInstances-te.numberAllocations, waiting)
	assert.Len(te.t, ms, te.numberMachineInstances)
}

func (te *test) startApiInstances(ctx context.Context) {
	var (
		timeoutCtx, cancel = context.WithTimeout(ctx, 5*time.Second)
		g, gCtx            = errgroup.WithContext(timeoutCtx)
	)

	defer cancel()

	for range te.numberApiInstances {
		instanceCtx, cancel := context.WithCancel(ctx)
		allocate := make(chan string)

		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(te.t, err)

		server := &server{
			cancel:   cancel,
			allocate: allocate,
			cfg: &ServerConfig{
				Context:    instanceCtx,
				Store:      te.ds,
				Logger:     te.log.WithGroup("grpc-server"),
				Listener:   listener,
				Publisher:  te.publisher,
				Consumer:   te.consumer,
				TlsEnabled: false,
			},
		}

		te.ss = append(te.ss, server)

		go func() {
			addr := server.cfg.Listener.Addr().String()

			te.t.Logf("running grpc server on %s", addr)

			if err := Run(server.cfg); err != nil {
				te.t.Logf("error running grpc server: %s", err)
			} else {
				te.t.Logf("grpc server on %s stopped successfully", addr)
			}
		}()

		g.Go(func() error {
			conn, err := server.newClient()
			if err != nil {
				return err
			}

			conn.Connect()
			defer conn.Close()

			return retry.Do(func() error {
				state := conn.GetState()
				if state == connectivity.Ready {
					te.t.Logf("grpc server on %s is accepting client connections", server.cfg.Listener.Addr().String())
					return nil
				}
				return fmt.Errorf("client cannot connect, still in state %s", state)
			}, retry.Context(gCtx))
		})
	}

	if err := g.Wait(); err != nil {
		te.t.Errorf("grpc servers did not come up: %s", err)
	}

	te.t.Logf("all %d grpc servers started successfully", len(te.ss))
}

func (te *test) startMachineInstances(ctx context.Context, allocations chan string) {
	var (
		timeoutCtx, cancel = context.WithTimeout(ctx, 5*time.Second)
		g, gCtx            = errgroup.WithContext(timeoutCtx)
	)

	defer cancel()

	for i := range te.numberMachineInstances {
		var (
			machineID   = strconv.Itoa(i)
			server      = te.ss[i%len(te.ss)]
			ctx, cancel = context.WithCancel(ctx)
		)

		conn, err := server.newClient()
		require.NoError(te.t, err)

		client := &client{
			machineID: machineID,
			conn:      conn,
			c:         v1.NewBootServiceClient(conn),
			cancel:    cancel,
		}

		te.cc = append(te.cc, client)

		err = te.ds.CreateMachine(&metal.Machine{
			Base: metal.Base{
				ID: client.machineID,
			},
		})
		require.NoError(te.t, err)

		te.t.Logf("created machine %q in rethinkdb store", client.machineID)

		go func() {
			client.conn.Connect()

			err := v1grpc.WaitForAllocation(ctx, te.log.WithGroup("grpc-client").With("machine-id", client.machineID), client.c, client.machineID, 2*time.Second)
			if err != nil {
				return
			}

			allocations <- machineID
		}()

		g.Go(func() error {
			return retry.Do(func() error {
				m, err := te.ds.FindMachineByID(client.machineID)
				require.NoError(te.t, err)

				if m.Waiting {
					return nil
				}

				return fmt.Errorf("machine %s is not yet waiting", m.ID)
			}, retry.Context(gCtx))
		})
	}

	if err := g.Wait(); err != nil {
		te.t.Errorf("grpc clients did not come up: %s", err)
	}

	te.t.Logf("all %d grpc clients are now waiting", len(te.cc))
}

func (te *test) shutdown() {
	te.stopMachineInstances()
	te.stopApiInstances()
}

func (te *test) stopApiInstances() {
	for _, s := range te.ss {
		te.t.Logf("stopping grpc server on %s", s.cfg.Listener.Addr().String())
		s.cancel()
	}
	te.ss = nil
}

func (te *test) stopMachineInstances() {
	for _, c := range te.cc {
		te.t.Logf("stopping grpc client for machine %s", c.machineID)
		c.cancel()
		err := c.conn.Close()
		require.NoError(te.t, err, "unable to shutdown grpc client conn")
	}
	te.cc = nil
}

func (s *server) newClient() (*grpc.ClientConn, error) {
	kacp := keepalive.ClientParameters{
		Time:                1 * time.Second,
		Timeout:             500 * time.Millisecond,
		PermitWithoutStream: true,
	}
	opts := []grpc.DialOption{
		grpc.WithKeepaliveParams(kacp),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.NewClient(s.cfg.Listener.Addr().String(), opts...)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
