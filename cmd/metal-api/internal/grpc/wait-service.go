package grpc

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"

	mathrand "math/rand"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"github.com/metal-stack/metal-lib/bus"
)

const (
	receiverHandlerTimeout = 15 * time.Second
	allocationTopicTTL     = time.Duration(30) * time.Second
)

type WaitService struct {
	bus.Publisher
	consumer         *bus.Consumer
	logger           *zap.SugaredLogger
	ds               Datasource
	queueLock        *sync.RWMutex
	queue            map[string]chan bool
	responseInterval time.Duration
	checkInterval    time.Duration
}

type Datasource interface {
	FindMachineByID(machineID string) (*metal.Machine, error)
	UpdateMachine(old, new *metal.Machine) error
}

func NewWaitService(cfg *ServerConfig) (*WaitService, error) {
	c, err := bus.NewConsumer(cfg.Logger.Desugar(), cfg.NsqTlsConfig, cfg.NsqlookupdHttpAddress)
	if err != nil {
		return nil, errors.Wrap(err, "cannot connect to NSQ")
	}

	s := &WaitService{
		Publisher:        cfg.Publisher,
		consumer:         c,
		ds:               cfg.Datasource,
		logger:           cfg.Logger,
		queueLock:        new(sync.RWMutex),
		queue:            make(map[string]chan bool),
		responseInterval: cfg.ResponseInterval,
		checkInterval:    cfg.CheckInterval,
	}

	err = s.initWaitEndpoint()
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *WaitService) NotifyAllocated(machineID string) error {
	err := s.Publish(metal.TopicAllocation.Name, &metal.AllocationEvent{MachineID: machineID})
	if err != nil {
		s.logger.Errorw("failed to publish machine allocation event", "topic", metal.TopicAllocation.Name, "machineID", machineID, "error", err)
	} else {
		s.logger.Debugw("published machine allocation event", "topic", metal.TopicAllocation.Name, "machineID", machineID)
	}
	return err
}

func (s *WaitService) initWaitEndpoint() error {
	if s.Publisher == nil {
		return nil
	}
	var r uint64
	b, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err == nil {
		r = b.Uint64()
	} else {
		s.logger.Warnw("failed to generate crypto random number -> fallback to math random number", "error", err)
		mathrand.Seed(time.Now().UnixNano())
		r = mathrand.Uint64()
	}
	channel := fmt.Sprintf("alloc-%d#ephemeral", r)
	return s.consumer.With(bus.LogLevel(bus.Warning)).
		MustRegister(metal.TopicAllocation.Name, channel).
		Consume(metal.AllocationEvent{}, func(message interface{}) error {
			evt := message.(*metal.AllocationEvent)
			s.logger.Debugw("got message", "topic", metal.TopicAllocation.Name, "channel", channel, "machineID", evt.MachineID)
			s.handleAllocation(evt.MachineID)
			return nil
		}, 5, bus.Timeout(receiverHandlerTimeout, s.timeoutHandler), bus.TTL(allocationTopicTTL))
}

func (s *WaitService) timeoutHandler(err bus.TimeoutError) error {
	s.logger.Error("Timeout processing event", "event", err.Event())
	return nil
}

func (s *WaitService) Wait(req *v1.WaitRequest, srv v1.Wait_WaitServer) error {
	machineID := req.MachineID
	s.logger.Infow("wait for allocation called by", "machineID", machineID)

	m, err := s.ds.FindMachineByID(machineID)
	if err != nil {
		return err
	}
	allocated := m.Allocation != nil
	if allocated {
		return nil
	}

	// machine is not yet allocated, so we set the waiting flag
	err = s.updateWaitingFlag(machineID, true)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			return
		}
		err := s.updateWaitingFlag(machineID, false)
		if err != nil {
			s.logger.Errorw("unable to remove waiting flag from machine", "machineID", machineID, "error", err)
		}
	}()

	// we also create and listen to a channel that will be used as soon as the machine is allocated
	s.queueLock.RLock()
	can, ok := s.queue[machineID]
	s.queueLock.RUnlock()
	if !ok {
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
			err = ctx.Err()
			return err
		case allocated := <-can:
			if allocated {
				return nil
			}
		case now := <-time.After(s.responseInterval):
			if now.After(nextCheck) {
				m, err = s.ds.FindMachineByID(machineID)
				if err != nil {
					return err
				}
				allocated := m.Allocation != nil
				if allocated {
					return nil
				}
				nextCheck = now.Add(s.checkInterval)
			}
			err = sendKeepPatientResponse(srv)
			if err != nil {
				return err
			}
		}
	}
}

// https://github.com/grpc/grpc-go/issues/1229#issuecomment-302755717
func sendKeepPatientResponse(srv v1.Wait_WaitServer) error {
	errChan := make(chan error, 1)
	ctx := srv.Context()
	go func() {
		errChan <- srv.Send(&v1.KeepPatientResponse{})
		close(errChan)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}

func (s *WaitService) handleAllocation(machineID string) {
	s.queueLock.RLock()
	defer s.queueLock.RUnlock()

	can, ok := s.queue[machineID]
	if ok {
		can <- true
	}
}

func (s *WaitService) updateWaitingFlag(machineID string, flag bool) error {
	m, err := s.ds.FindMachineByID(machineID)
	if err != nil {
		return err
	}
	old := *m
	m.Waiting = flag
	return s.ds.UpdateMachine(&old, m)
}
