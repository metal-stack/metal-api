package grpc

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"

	mathrand "math/rand"

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
	Logger           *zap.SugaredLogger
	ds               Datasource
	queue            sync.Map
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
		return nil, fmt.Errorf("cannot connect to NSQ: %w", err)
	}

	s := &WaitService{
		Publisher:        cfg.Publisher,
		consumer:         c,
		ds:               cfg.Datasource,
		Logger:           cfg.Logger,
		queue:            sync.Map{},
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
		s.Logger.Errorw("failed to publish machine allocation event", "topic", metal.TopicAllocation.Name, "machineID", machineID, "error", err)
	} else {
		s.Logger.Debugw("published machine allocation event", "topic", metal.TopicAllocation.Name, "machineID", machineID)
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
		s.Logger.Warnw("failed to generate crypto random number -> fallback to math random number", "error", err)
		mathrand.Seed(time.Now().UnixNano())
		// nolint
		r = mathrand.Uint64()
	}
	channel := fmt.Sprintf("alloc-%d#ephemeral", r)
	return s.consumer.With(bus.LogLevel(bus.Warning)).
		MustRegister(metal.TopicAllocation.Name, channel).
		Consume(metal.AllocationEvent{}, func(message interface{}) error {
			evt := message.(*metal.AllocationEvent)
			s.Logger.Debugw("got message", "topic", metal.TopicAllocation.Name, "channel", channel, "machineID", evt.MachineID)
			s.handleAllocation(evt.MachineID)
			return nil
		}, 5, bus.Timeout(receiverHandlerTimeout, s.timeoutHandler), bus.TTL(allocationTopicTTL))
}

func (s *WaitService) timeoutHandler(err bus.TimeoutError) error {
	s.Logger.Error("Timeout processing event", "event", err.Event())
	return nil
}

func (s *WaitService) Wait(req *v1.WaitRequest, srv v1.Wait_WaitServer) error {
	machineID := req.MachineID
	s.Logger.Infow("wait for allocation called by", "machineID", machineID)

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
			s.Logger.Errorw("unable to remove waiting flag from machine", "machineID", machineID, "error", err)
		}
	}()

	// we also create and listen to a channel that will be used as soon as the machine is allocated
	value, ok := s.queue.Load(machineID)

	var can chan bool
	if !ok {
		can = make(chan bool)
		s.queue.Store(machineID, can)
	} else {
		can, ok = value.(chan bool)
	}

	if !ok {
		return fmt.Errorf("unable to cast queue entry to a chan bool")
	}

	defer func() {
		s.queue.Delete(machineID)
		close(can)
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
	value, ok := s.queue.Load(machineID)
	can, okcast := value.(chan bool)
	if !okcast {
		s.Logger.Error("unable to cast queue entry to chan bool")
		return
	}
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
