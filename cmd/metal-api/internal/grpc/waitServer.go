package grpc

import (
	"fmt"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"github.com/metal-stack/metal-lib/bus"
	"github.com/metal-stack/metal-lib/zapup"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"math/rand"
	"sync"
	"time"
)

const (
	receiverHandlerTimeout = 15 * time.Second
	allocationTopicTTL     = time.Duration(30) * time.Second
)

var allocationTopic = metal.TopicAllocation.GetFQN("alloc")

func timeoutHandler(err bus.TimeoutError) error {
	zapup.MustRootLogger().Sugar().Error("Timeout processing event", "event", err.Event())
	return nil
}

type WaitServer struct {
	bus.Publisher
	ds        *datastore.RethinkStore
	logger    *zap.SugaredLogger
	queueLock *sync.RWMutex
	queue     map[string]chan bool
}

func NewWaitServer(ds *datastore.RethinkStore, publisher bus.Publisher, partitions metal.Partitions) (*WaitServer, error) {
	tlsCfg := &bus.TLSConfig{
		CACertFile:     viper.GetString("nsqd-ca-cert-file"),
		ClientCertFile: viper.GetString("nsqd-client-cert-file"),
	}
	c, err := bus.NewConsumer(zapup.MustRootLogger(), tlsCfg, viper.GetString("nsqlookupd-http-addr"))
	if err != nil {
		return nil, err
	}

	s := &WaitServer{
		Publisher: publisher,
		ds:        ds,
		logger:    zapup.MustRootLogger().Sugar(),
		queueLock: new(sync.RWMutex),
		queue:     make(map[string]chan bool),
	}

	channel := fmt.Sprintf("alloc-%d", rand.Int())
	err = c.With(bus.LogLevel(bus.Debug)).
		MustRegister(allocationTopic, channel).
		Consume(metal.AllocationEvent{}, func(message interface{}) error {
			evt := message.(*metal.AllocationEvent)
			s.logger.Debugf("Got message", "topic", allocationTopic, "channel", channel, "machineID", evt.MachineID)
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
	err := s.Publish(allocationTopic, &metal.AllocationEvent{MachineID: machineID})
	if err != nil {
		s.logger.Errorf("failed to publish machine allocation event", "topic", allocationTopic, "machineID", machineID, "error", err)
	} else {
		s.logger.Debugf("published machine allocation event", "topic", allocationTopic, "machineID", machineID)
	}
	return err
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
