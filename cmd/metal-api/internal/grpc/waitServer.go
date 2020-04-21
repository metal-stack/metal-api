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
	queue     map[string]bool
}

func NewWaitServer(ds *datastore.RethinkStore) *WaitServer {
	return &WaitServer{
		ds:        ds,
		logger:    zapup.MustRootLogger().Sugar(),
		queueLock: new(sync.RWMutex),
		queue:     make(map[string]bool),
	}
}

func (s *WaitServer) NotifyAllocated(machineID string) {
	s.queueLock.Lock()
	_, ok := s.queue[machineID]
	if ok {
		s.queue[machineID] = true
	}
	s.queueLock.Unlock()
}

func (s *WaitServer) Wait(req *v1.WaitRequest, srv v1.Wait_WaitServer) error {
	s.logger.Infof("wait for allocation called by", "machineID", req.MachineID)
	machineID := req.MachineID

	s.queueLock.RLock()
	allocated, ok := s.queue[machineID]
	s.queueLock.RUnlock()

	if !ok || !allocated {
		m, err := s.ds.FindMachineByID(machineID)
		if err != nil {
			return err
		}
		allocated = m.Allocation != nil
	}

	if allocated {
		s.queueLock.Lock()
		delete(s.queue, machineID)
		s.queueLock.Unlock()
		return nil
	}

	s.queueLock.Lock()
	s.queue[machineID] = false
	s.queueLock.Unlock()

	ctx := srv.Context()
	ticker := time.NewTicker(500 * time.Millisecond)

	defer func() {
		ticker.Stop()

		s.queueLock.Lock()
		delete(s.queue, machineID)
		s.queueLock.Unlock()
	}()

	nextResponse := time.Now()
	for now := range ticker.C {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			s.queueLock.RLock()
			allocated = s.queue[machineID]
			s.queueLock.RUnlock()

			if allocated {
				return nil
			}

			if now.After(nextResponse) {
				err := srv.Send(&v1.WaitResponse{})
				if err != nil {
					s.logger.Errorw("failed to respond", "error", err)
					return err
				}
				nextResponse = now.Add(5 * time.Second)
			}
		}
	}

	return nil
}
