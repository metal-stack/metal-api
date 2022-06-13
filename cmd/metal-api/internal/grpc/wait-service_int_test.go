//go:build integration
// +build integration

package grpc

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

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

type test struct {
	*testing.T
	ss []*Server
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
			require.True(t, a > 0)
			require.True(t, m[0] > 0)
			require.True(t, m[1] > 0)
			require.True(t, m[0] >= m[1])
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

	ds, mock := datastore.InitMockDB()
	for i := 0; i < t.numberMachineInstances; i++ {
		machineID := strconv.Itoa(i)
		mock.On(r.DB("mockdb").Table("machine").Get(machineID)).Return(metal.Machine{
			Base: metal.Base{ID: machineID},
		}, nil)
		mock.On(r.DB("mockdb").Table("machine").Get(machineID).Replace(r.MockAnything())).Return(nil, nil)
	}

	t.startApiInstances(ds)
	t.startMachineInstances()
	t.notReadyMachines.Wait()

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

	t.allocateMachines()

	t.unallocatedMachines.Wait()

	require.Equal(t, t.numberAllocations, len(t.allocations))
	for _, allocated := range t.allocations {
		require.True(t, allocated)
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
		s.Stop()
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
	for i := 0; i < t.numberApiInstances; i++ {
		cfg := &ServerConfig{
			Store:            ds,
			Logger:           zaptest.NewLogger(t).Sugar(),
			GrpcPort:         50005 + i,
			TlsEnabled:       false,
			ResponseInterval: 2 * time.Millisecond,
			CheckInterval:    1 * time.Hour,
		}
		s, err := NewServer(cfg)
		require.NoError(t, err)
		t.ss = append(t.ss, s)
		go func() {
			err := s.Serve()
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
		grpc.WithBlock(),
	}
	for i := 0; i < t.numberMachineInstances; i++ {
		machineID := strconv.Itoa(i)
		port := 50005 + t.randNumber(t.numberApiInstances)
		ctx, cancel := context.WithCancel(context.Background())
		conn, err := grpc.DialContext(ctx, fmt.Sprintf("localhost:%d", port), opts...)
		require.NoError(t, err)
		t.cc = append(t.cc, &client{
			ClientConn: conn,
			cancel:     cancel,
		})
		go func() {
			waitClient := v1.NewWaitClient(conn)
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

func (t *test) waitForAllocation(machineID string, c v1.WaitClient, ctx context.Context) error {
	req := &v1.WaitRequest{
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
	for i := 0; i < t.numberAllocations; i++ {
		machineID := t.selectMachine(alreadyAllocated)
		alreadyAllocated = append(alreadyAllocated, machineID)
		t.mtx.Lock()
		t.allocations[machineID] = false
		t.mtx.Unlock()
		t.simulateNsqNotifyAllocated(machineID)
	}
}

func (t *test) selectMachine(except []string) string {
	machineID := strconv.Itoa(t.randNumber(t.numberMachineInstances))
	for _, id := range except {
		if id == machineID {
			return t.selectMachine(except)
		}
	}
	return machineID
}

func (t *test) simulateNsqNotifyAllocated(machineID string) {
	for _, s := range t.ss {
		s.WaitService().handleAllocation(machineID)
	}
}

func (t *test) randNumber(n int) int {
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	require.NoError(t, err)
	return int(nBig.Int64())
}
