package grpc

import (
	"context"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/test/bufconn"
	"io"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"
)

type client struct {
	*grpc.ClientConn
	cancel func()
}

type test struct {
	*testing.T
	ss []*WaitServer
	ll []*bufconn.Listener
	cc []*client

	numberApiInstances     int
	numberMachineInstances int
	numberAllocations      int

	notReadyMachines    *sync.WaitGroup
	unallocatedMachines *sync.WaitGroup
	allocationsMutex    *sync.RWMutex
	allocations         map[string]bool
}

func TestWaitServer(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	var tt []*test
	aa := []int{1, 10}
	mm := [][]int{{1, 1}, {3, 1}, {10, 7}}
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
		test.run()
	}
}

type datasource struct {
	mutex *sync.RWMutex
	wait  map[string]bool
}

func (ds *datasource) FindMachineByID(machineID string) (*metal.Machine, error) {
	return &metal.Machine{
		Base: metal.Base{
			ID: machineID,
		},
	}, nil
}

func (ds *datasource) UpdateMachine(old, new *metal.Machine) error {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()
	ds.wait[new.ID] = new.Waiting
	return nil
}

func (t *test) run() {
	defer t.shutdown()

	t.notReadyMachines = new(sync.WaitGroup)
	t.notReadyMachines.Add(t.numberMachineInstances)
	t.unallocatedMachines = new(sync.WaitGroup)
	t.unallocatedMachines.Add(t.numberAllocations)
	t.allocationsMutex = new(sync.RWMutex)
	t.allocations = make(map[string]bool)

	ds := &datasource{
		mutex: new(sync.RWMutex),
		wait:  make(map[string]bool),
	}

	t.startApiInstances(ds)
	t.startMachineInstances()
	t.notReadyMachines.Wait()

	require.Equal(t, t.numberMachineInstances, len(ds.wait))
	ds.mutex.RLock()
	for _, wait := range ds.wait {
		require.True(t, wait)
	}
	ds.mutex.RUnlock()

	require.Equal(t, t.numberMachineInstances, len(ds.wait))
	ds.mutex.RLock()
	for _, wait := range ds.wait {
		require.True(t, wait)
	}
	ds.mutex.RUnlock()

	t.allocateMachines()

	t.unallocatedMachines.Wait()

	t.allocationsMutex.RLock()
	require.Equal(t, t.numberAllocations, len(t.allocations))
	for _, allocated := range t.allocations {
		require.True(t, allocated)
	}

	ds.mutex.RLock()
	require.Equal(t, t.numberMachineInstances, len(ds.wait))
	for key, wait := range ds.wait {
		require.Equal(t, !containsKey(t.allocations, key), wait)
	}
	ds.mutex.RUnlock()

	t.allocationsMutex.RUnlock()
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
		t.ll = t.ll[:0]
	}()
	for i, s := range t.ss {
		t.ll[i].Close()
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
	wg := new(sync.WaitGroup)
	wg.Add(t.numberApiInstances)
	for i := 0; i < t.numberApiInstances; i++ {
		s := &WaitServer{
			server:           grpc.NewServer(),
			ds:               ds,
			queueLock:        new(sync.RWMutex),
			queue:            make(map[string]chan bool),
			logger:           zap.NewNop().Sugar(),
			responseInterval: 2 * time.Millisecond,
			checkInterval:    time.Hour,
		}
		t.ss = append(t.ss, s)

		l := bufconn.Listen(1024)
		t.ll = append(t.ll, l)
		v1.RegisterWaitServer(s.server, s)
		go func() {
			wg.Done()
			_ = s.server.Serve(l)
		}()
	}
	wg.Wait()
}

func (t *test) startMachineInstances() {
	kacp := keepalive.ClientParameters{
		Time:                5 * time.Millisecond,
		Timeout:             20 * time.Millisecond,
		PermitWithoutStream: true,
	}
	bufDialer := func(l *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
		return func(context.Context, string) (net.Conn, error) {
			return l.Dial()
		}
	}
	for i := 0; i < t.numberMachineInstances; i++ {
		machineID := strconv.Itoa(i)
		ctx, cancel := context.WithCancel(context.Background())
		l := t.ll[rand.Intn(t.numberApiInstances)]
		opts := []grpc.DialOption{
			grpc.WithContextDialer(bufDialer(l)),
			grpc.WithKeepaliveParams(kacp),
			grpc.WithInsecure(),
			grpc.WithBlock(),
		}
		conn, err := grpc.DialContext(ctx, "bufnet", opts...)
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
			t.allocationsMutex.Lock()
			t.allocations[machineID] = true
			t.allocationsMutex.Unlock()
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
		if err != nil {
			time.Sleep(2 * time.Millisecond)
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
		t.allocationsMutex.Lock()
		t.allocations[machineID] = false
		t.allocationsMutex.Unlock()
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
