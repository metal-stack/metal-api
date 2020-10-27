package grpc

import (
	"context"
	"fmt"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"io"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"
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
	ss []*WaitServer
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
	rand.Seed(time.Now().UnixNano())

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

type datasource struct {
	mtx  *sync.Mutex
	wait map[string]bool
}

func (ds *datasource) FindMachineByID(machineID string) (*metal.Machine, error) {
	return &metal.Machine{
		Base: metal.Base{
			ID: machineID,
		},
	}, nil
}

func (ds *datasource) UpdateMachine(old, new *metal.Machine) error {
	ds.mtx.Lock()
	defer ds.mtx.Unlock()
	ds.wait[new.ID] = new.Waiting
	return nil
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

	ds := &datasource{
		mtx:  new(sync.Mutex),
		wait: make(map[string]bool),
	}

	t.startApiInstances(ds)
	t.startMachineInstances()
	t.notReadyMachines.Wait()

	require.Equal(t, t.numberMachineInstances, len(ds.wait))
	for _, wait := range ds.wait {
		require.True(t, wait)
	}

	switch t.testCase {
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

	require.Equal(t, t.numberMachineInstances, len(ds.wait))
	for _, wait := range ds.wait {
		require.True(t, wait)
	}

	t.allocateMachines()

	t.unallocatedMachines.Wait()

	require.Equal(t, t.numberAllocations, len(t.allocations))
	for _, allocated := range t.allocations {
		require.True(t, allocated)
	}

	require.Equal(t, t.numberMachineInstances, len(ds.wait))
	for key, wait := range ds.wait {
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
		s.server.Stop()
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

func (t *test) startApiInstances(ds Datasource) {
	for i := 0; i < t.numberApiInstances; i++ {
		s := &WaitServer{
			ds:               ds,
			queueLock:        new(sync.RWMutex),
			queue:            make(map[string]chan bool),
			grpcPort:         50005 + i,
			logger:           zap.NewNop().Sugar(),
			responseInterval: 2 * time.Millisecond,
		}
		t.ss = append(t.ss, s)
		go func() {
			err := s.Serve()
			require.Nil(t, err)
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
		grpc.WithInsecure(),
		grpc.WithBlock(),
	}
	for i := 0; i < t.numberMachineInstances; i++ {
		machineID := strconv.Itoa(i)
		port := 50005 + rand.Intn(t.numberApiInstances)
		ctx, cancel := context.WithCancel(context.Background())
		conn, err := grpc.DialContext(ctx, fmt.Sprintf("localhost:%d", port), opts...)
		require.Nil(t, err)
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
		MachineID: machineID,
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
			if err == io.EOF {
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
	machineID := strconv.Itoa(rand.Intn(t.numberMachineInstances))
	for _, id := range except {
		if id == machineID {
			return t.selectMachine(except)
		}
	}
	return machineID
}

func (t *test) simulateNsqNotifyAllocated(machineID string) {
	for _, s := range t.ss {
		s.handleAllocation(machineID)
	}
}
