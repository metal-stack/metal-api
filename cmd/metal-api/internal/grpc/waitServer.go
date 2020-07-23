package grpc

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"github.com/metal-stack/metal-lib/bus"
	"github.com/metal-stack/metal-lib/zapup"
	"go.uber.org/zap"
)

const (
	receiverHandlerTimeout = 15 * time.Second
	allocationTopicTTL     = time.Duration(30) * time.Second
)

func timeoutHandler(err bus.TimeoutError) error {
	zapup.MustRootLogger().Sugar().Error("Timeout processing event", "event", err.Event())
	return nil
}

type WaitServerConfig struct {
	Publisher             bus.Publisher
	Datasource            *datastore.RethinkStore
	Logger                *zap.SugaredLogger
	NsqTlsConfig          *bus.TLSConfig
	NsqlookupdHttpAddress string
	GrpcPort              int
	TlsEnabled            bool
	CaCertFile            string
	ServerCertFile        string
	ServerKeyFile         string
}

type WaitServer struct {
	bus.Publisher
	ds             *datastore.RethinkStore
	logger         *zap.SugaredLogger
	queueLock      *sync.RWMutex
	queue          map[string]chan bool
	GrpcPort       int
	TlsEnabled     bool
	CaCertFile     string
	ServerCertFile string
	ServerKeyFile  string
}

func NewWaitServer(cfg *WaitServerConfig) (*WaitServer, error) {
	c, err := bus.NewConsumer(zapup.MustRootLogger(), cfg.NsqTlsConfig, cfg.NsqlookupdHttpAddress)
	if err != nil {
		return nil, err
	}

	s := &WaitServer{
		Publisher:      cfg.Publisher,
		ds:             cfg.Datasource,
		logger:         cfg.Logger,
		queueLock:      new(sync.RWMutex),
		queue:          make(map[string]chan bool),
		GrpcPort:       cfg.GrpcPort,
		TlsEnabled:     cfg.TlsEnabled,
		CaCertFile:     cfg.CaCertFile,
		ServerCertFile: cfg.ServerCertFile,
		ServerKeyFile:  cfg.ServerKeyFile,
	}

	rand.Seed(time.Now().Unix())
	channel := fmt.Sprintf("alloc-%d#ephemeral", rand.Int())
	err = c.With(bus.LogLevel(bus.Warning)).
		MustRegister(metal.TopicAllocation.Name, channel).
		Consume(metal.AllocationEvent{}, func(message interface{}) error {
			evt := message.(*metal.AllocationEvent)
			s.logger.Debugw("got message", "topic", metal.TopicAllocation.Name, "channel", channel, "machineID", evt.MachineID)
			s.queueLock.Lock()
			s.queue[evt.MachineID] <- true
			s.queueLock.Unlock()
			return nil
		}, 5, bus.Timeout(receiverHandlerTimeout, timeoutHandler), bus.TTL(allocationTopicTTL))
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *WaitServer) NotifyAllocated(machineID string) error {
	err := s.Publish(metal.TopicAllocation.Name, &metal.AllocationEvent{MachineID: machineID})
	if err != nil {
		s.logger.Errorw("failed to publish machine allocation event", "topic", metal.TopicAllocation.Name, "machineID", machineID, "error", err)
	} else {
		s.logger.Debugw("published machine allocation event", "topic", metal.TopicAllocation.Name, "machineID", machineID)
	}
	return err
}

func (s *WaitServer) Wait(req *v1.WaitRequest, srv v1.Wait_WaitServer) error {
	s.logger.Infow("wait for allocation called by", "machineID", req.MachineID)
	machineID := req.MachineID

	err := s.updateWaitingFlag(machineID, true)
	if err != nil {
		return err
	}
	defer func() {
		err := s.updateWaitingFlag(machineID, false)
		if err != nil {
			s.logger.Errorw("unable to remove waiting flag from machine", "machineID", machineID, "error", err)
		}
	}()

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

	nextCheck := time.Now()
	ctx := srv.Context()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case allocated := <-can:
			if !allocated {
				continue
			}
			s.queueLock.Lock()
			close(can)
			delete(s.queue, machineID)
			s.queueLock.Unlock()
			return nil
		case now := <-time.After(5 * time.Second):
			if now.After(nextCheck) {
				m, err := s.ds.FindMachineByID(machineID)
				if err != nil {
					return err
				}
				allocated := m.Allocation != nil
				if allocated {
					return nil
				}
				nextCheck = now.Add(60 * time.Second)
			}
			err := srv.Send(&v1.WaitResponse{})
			if err != nil {
				return err
			}
		}
	}
}

func (s *WaitServer) updateWaitingFlag(machineID string, flag bool) error {
	m, err := s.ds.FindMachineByID(machineID)
	if err != nil {
		return err
	}
	old := *m
	m.Waiting = flag
	return s.ds.UpdateMachine(&old, m)
}
