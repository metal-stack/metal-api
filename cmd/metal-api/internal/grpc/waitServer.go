package grpc

import (
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"github.com/metal-stack/metal-lib/zapup"
	"go.uber.org/zap"
	"time"
)

type WaitServer struct {
	ds     *datastore.RethinkStore
	logger *zap.SugaredLogger
}

func NewWaitServer(ds *datastore.RethinkStore) *WaitServer {
	return &WaitServer{
		ds:     ds,
		logger: zapup.MustRootLogger().Sugar(),
	}
}

func (s *WaitServer) Wait(req *v1.WaitRequest, srv v1.Wait_WaitServer) error {
	s.logger.Infof("wait for allocation called by", "machineID", req.Uuid)
	ctx := srv.Context()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			id := req.Uuid

			m, err := s.ds.FindMachineByID(id)
			if err != nil {
				return err
			}

			allocated := m.Allocation != nil
			resp := &v1.WaitResponse{
				Uuid:      id,
				Allocated: allocated,
			}
			err = srv.Send(resp)
			if err != nil {
				s.logger.Errorw("send error", "error", err)
				return err
			}

			if allocated {
				// Close stream regularly
				return nil
			}

			timeout := time.Now().Add(5 * time.Second)
			ticker := time.NewTicker(500 * time.Millisecond)
			for now := range ticker.C {
				m, err := s.ds.FindMachineByID(id)
				if err != nil {
					return err
				}
				if m.Allocation != nil || now.After(timeout) {
					break
				}
			}
			ticker.Stop()
		}
	}
}
