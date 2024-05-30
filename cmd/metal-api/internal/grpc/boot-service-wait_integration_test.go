//go:build integration
// +build integration

package grpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"github.com/undefinedlabs/go-mpatch"

	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

type testCase int

const (
	happyPath testCase = iota
	serverFailure
	clientFailure
)

type client struct {
	*grpc.ClientConn
	cancel func()
}

type server struct {
	cancel   context.CancelFunc
	allocate chan string
}

type test struct {
	*testing.T
	ss []*server
	cc []*client

	numberApiInstances     int
	numberMachineInstances int
	numberAllocations      int
	testCase               testCase

	notReadyMachines    *sync.WaitGroup
	unallocatedMachines *sync.WaitGroup
	mtx                 *sync.Mutex
	allocations         map[string]bool
}

func TestWaitServer(t *testing.T) {
	var tt []*test
	aa := []int{1, 10}
	mm := [][]int{{10, 7}}
	for _, a := range aa {
		for _, m := range mm {
			require.Positive(t, a)
			require.Positive(t, m[0])
			require.Positive(t, m[1])
			require.GreaterOrEqual(t, m[0], m[1])
			tt = append(tt, &test{
				numberApiInstances:     a,
				numberMachineInstances: m[0],
				numberAllocations:      m[1],
			})
		}
	}
	for _, test := range tt {
		test.T = t
		test.testCase = happyPath
		test.run()
		test.testCase = serverFailure
		test.run()
		test.testCase = clientFailure
		test.run()

	}
}

func (t *test) run() {
	defer t.shutdown()

	time.Sleep(20 * time.Millisecond)

	t.notReadyMachines = new(sync.WaitGroup)
	t.notReadyMachines.Add(t.numberMachineInstances)
	t.unallocatedMachines = new(sync.WaitGroup)
	t.unallocatedMachines.Add(t.numberAllocations)
	t.mtx = new(sync.Mutex)
	t.allocations = make(map[string]bool)

	now := time.Now()
	_, _ = mpatch.PatchMethod(time.Now, func() time.Time {
		return now
	})

	wait := make(map[string]bool)

	insertMock := func(w bool, id string) r.Term {
		return r.DB("mockdb").Table("machine").Get(id).Replace(func(row r.Term) r.Term {
			return r.Branch(row.Field("changed").Eq(r.MockAnything()), metal.Machine{
				Base:    metal.Base{ID: id, Changed: now},
				Waiting: w,
			}, r.MockAnything())
		})
	}
	returnMock := func(w bool, id string) func() []interface{} {
		return func() []interface{} {
			t.mtx.Lock()
			defer t.mtx.Unlock()
			wait[id] = w
			return []interface{}{r.WriteResponse{}}
		}
	}

	ds, mock := datastore.InitMockDB(t.T)
	for i := range t.numberMachineInstances {
		machineID := strconv.Itoa(i)
		mock.On(r.DB("mockdb").Table("machine").Get(machineID)).Return(metal.Machine{Base: metal.Base{ID: machineID}}, nil)
		mock.On(insertMock(true, machineID)).Return(returnMock(true, machineID), nil)
		mock.On(insertMock(false, machineID)).Return(returnMock(false, machineID), nil)
	}

	t.startApiInstances(ds)
	t.startMachineInstances()
	t.notReadyMachines.Wait()

	require.Len(t, wait, t.numberMachineInstances)
	for _, wait := range wait {
		require.True(t, wait)
	}

	switch t.testCase {
	case happyPath:
	case serverFailure:
		t.notReadyMachines.Add(t.numberMachineInstances)
		t.stopApiInstances()
		t.startApiInstances(ds)
		t.notReadyMachines.Wait()
	case clientFailure:
		t.notReadyMachines.Add(t.numberMachineInstances)
		t.stopMachineInstances()
		t.startMachineInstances()
		t.notReadyMachines.Wait()
	}

	require.Len(t, wait, t.numberMachineInstances)
	for _, wait := range wait {
		require.True(t, wait)
	}

	t.allocateMachines()

	t.unallocatedMachines.Wait()

	require.Len(t, t.allocations, t.numberAllocations)
	for _, allocated := range t.allocations {
		require.True(t, allocated)
	}

	require.Len(t, wait, t.numberMachineInstances)
	for key, wait := range wait {
		require.Equal(t, !containsKey(t.allocations, key), wait)
	}
}

func containsKey(m map[string]bool, key string) bool {
	for k := range m {
		if k == key {
			return true
		}
	}
	return false
}

func (t *test) shutdown() {
	t.stopMachineInstances()
	t.stopApiInstances()
}

func (t *test) stopApiInstances() {
	defer func() {
		t.ss = t.ss[:0]
	}()
	for _, s := range t.ss {
		s.cancel()
		time.Sleep(50 * time.Millisecond)
	}
}

func (t *test) stopMachineInstances() {
	defer func() {
		t.cc = t.cc[:0]
	}()
	for _, c := range t.cc {
		c.cancel()
		_ = c.Close()
	}
}

func (t *test) startApiInstances(ds *datastore.RethinkStore) {
	for i := range t.numberApiInstances {
		ctx, cancel := context.WithCancel(context.Background())
		allocate := make(chan string)

		cfg := &ServerConfig{
			Context:          ctx,
			Store:            ds,
			Logger:           slog.Default(),
			GrpcPort:         50005 + i,
			TlsEnabled:       false,
			ResponseInterval: 2 * time.Millisecond,
			CheckInterval:    1 * time.Hour,

			integrationTestAllocator: allocate,
		}

		t.ss = append(t.ss, &server{
			cancel:   cancel,
			allocate: allocate,
		})
		go func() {
			err := Run(cfg)
			require.NoError(t, err)
		}()
	}
}

func (t *test) startMachineInstances() {
	kacp := keepalive.ClientParameters{
		Time:                5 * time.Millisecond,
		Timeout:             20 * time.Millisecond,
		PermitWithoutStream: true,
	}
	opts := []grpc.DialOption{
		grpc.WithKeepaliveParams(kacp),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	for i := range t.numberMachineInstances {
		machineID := strconv.Itoa(i)
		// golangci-lint has an issue with math/rand/v2
		// here it provides sufficient randomness though because it's not used for cryptographic purposes
		port := 50005 + rand.N(t.numberApiInstances) //nolint:gosec
		ctx, cancel := context.WithCancel(context.Background())
		conn, err := grpc.NewClient(fmt.Sprintf("localhost:%d", port), opts...)
		require.NoError(t, err)
		t.cc = append(t.cc, &client{
			ClientConn: conn,
			cancel:     cancel,
		})
		go func() {
			waitClient := v1.NewBootServiceClient(conn)
			err := t.waitForAllocation(machineID, waitClient, ctx)
			if err != nil {
				return
			}
			t.mtx.Lock()
			t.allocations[machineID] = true
			t.mtx.Unlock()
			t.unallocatedMachines.Done()
		}()
	}
}

func (t *test) waitForAllocation(machineID string, c v1.BootServiceClient, ctx context.Context) error {
	req := &v1.BootServiceWaitRequest{
		MachineId: machineID,
	}

	for {
		stream, err := c.Wait(ctx, req)
		time.Sleep(5 * time.Millisecond)
		if err != nil {
			continue
		}

		receivedResponse := false

		for {
			_, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				if !receivedResponse {
					break
				}
				return nil
			}
			if err != nil {
				if !receivedResponse {
					break
				}
				if t.testCase == clientFailure {
					return err
				}
				break
			}
			if !receivedResponse {
				receivedResponse = true
				t.notReadyMachines.Done()
			}
		}
	}
}

func (t *test) allocateMachines() {
	var alreadyAllocated []string
	for range t.numberAllocations {
		machineID := t.selectMachine(alreadyAllocated)
		alreadyAllocated = append(alreadyAllocated, machineID)
		t.mtx.Lock()
		t.allocations[machineID] = false
		t.mtx.Unlock()
		t.simulateNsqNotifyAllocated(machineID)
	}
}

func (t *test) selectMachine(except []string) string {
	// golangci-lint has an issue with math/rand/v2
	// here it provides sufficient randomness though because it's not used for cryptographic purposes
	machineID := strconv.Itoa(rand.N(t.numberMachineInstances)) //nolint:gosec
	for _, id := range except {
		if id == machineID {
			return t.selectMachine(except)
		}
	}
	return machineID
}

func (t *test) simulateNsqNotifyAllocated(machineID string) {
	for _, s := range t.ss {
		s.allocate <- machineID
	}
}
