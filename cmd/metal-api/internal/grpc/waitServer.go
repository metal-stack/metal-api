package grpc

import (
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"github.com/metal-stack/metal-lib/zapup"
	"go.uber.org/zap"
	"sync"
	"time"
)

type WaitServer struct {
	ds        *datastore.RethinkStore
	logger    *zap.SugaredLogger
	queueLock *sync.RWMutex
	queue     map[string]chan bool
}

func NewWaitServer(ds *datastore.RethinkStore) *WaitServer {
	return &WaitServer{
		ds:        ds,
		logger:    zapup.MustRootLogger().Sugar(),
		queueLock: new(sync.RWMutex),
		queue:     make(map[string]chan bool),
	}
}

func (s *WaitServer) NotifyAllocated(machineID string) {
	s.queueLock.Lock()
	can, ok := s.queue[machineID]
	if ok {
		can <- true
	}
	s.queueLock.Unlock()
}

func (s *WaitServer) Wait(req *v1.WaitRequest, srv v1.Wait_WaitServer) error {
	s.logger.Infof("wait for allocation called by", "machineID", req.MachineID)
	machineID := req.MachineID

	s.queueLock.RLock()
	can, ok := s.queue[machineID]
	s.queueLock.RUnlock()

	if !ok {
		m, err := s.ds.FindMachineByID(machineID)
		if err != nil {
			return err
		}
		allocated := m.Allocation != nil
		if allocated {
			return nil
		}

		can = make(chan bool)
		s.queueLock.Lock()
		s.queue[machineID] = can
		s.queueLock.Unlock()
	}

	defer func() {
		s.queueLock.Lock()
		delete(s.queue, machineID)
		close(can)
		s.queueLock.Unlock()
	}()

	nextCheck := time.Now()
	ctx := srv.Context()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case allocated := <-can:
			if allocated {
				return nil
			}
		case now := <-time.After(500 * time.Millisecond):
			if now.After(nextCheck) {
				m, err := s.ds.FindMachineByID(machineID)
				if err != nil {
					return err
				}
				allocated := m.Allocation != nil
				if allocated {
					return nil
				}
				err = srv.Send(&v1.WaitResponse{})
				if err != nil {
					return err
				}
				nextCheck = now.Add(10 * time.Second)
			}
		}
	}
}
