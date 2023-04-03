package grpc

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"github.com/metal-stack/metal-lib/bus"
)

const (
	receiverHandlerTimeout = 15 * time.Second
	allocationTopicTTL     = time.Duration(30) * time.Second
)

func (b *BootService) Wait(req *v1.BootServiceWaitRequest, srv v1.BootService_WaitServer) error {
	machineID := req.MachineId
	b.log.Infow("wait for allocation called by", "machineID", machineID)

	m, err := b.ds.FindMachineByID(machineID)
	if err != nil {
		return err
	}
	allocated := m.Allocation != nil
	if allocated {
		return nil
	}

	// machine is not yet allocated, so we set the waiting flag
	err = b.updateWaitingFlag(machineID, true)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			return
		}
		err := b.updateWaitingFlag(machineID, false)
		if err != nil {
			b.log.Errorw("unable to remove waiting flag from machine", "machineID", machineID, "error", err)
		}
	}()

	// we also create and listen to a channel that will be used as soon as the machine is allocated
	value, ok := b.queue.Load(machineID)

	var can chan bool
	if !ok {
		can = make(chan bool)
		b.queue.Store(machineID, can)
	} else {
		can, ok = value.(chan bool)
		if !ok {
			return fmt.Errorf("unable to cast queue entry to a chan bool")
		}
	}

	defer func() {
		b.queue.Delete(machineID)
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
		case now := <-time.After(b.responseInterval):
			if now.After(nextCheck) {
				m, err = b.ds.FindMachineByID(machineID)
				if err != nil {
					return err
				}
				allocated := m.Allocation != nil
				if allocated {
					return nil
				}
				nextCheck = now.Add(b.checkInterval)
			}
			err = sendKeepPatientResponse(srv)
			if err != nil {
				return err
			}
		}
	}
}

func (b *BootService) initWaitEndpoint() error {
	if b.publisher == nil || b.consumer == nil {
		return nil
	}
	channel := fmt.Sprintf("alloc-%s#ephemeral", uuid.NewString())
	return b.consumer.With(bus.LogLevel(bus.Warning)).
		MustRegister(metal.TopicAllocation.Name, channel).
		Consume(metal.AllocationEvent{}, func(message interface{}) error {
			evt := message.(*metal.AllocationEvent)
			b.log.Debugw("got message", "topic", metal.TopicAllocation.Name, "channel", channel, "machineID", evt.MachineID)
			b.handleAllocation(evt.MachineID)
			return nil
		}, 5, bus.Timeout(receiverHandlerTimeout, b.timeoutHandler), bus.TTL(allocationTopicTTL))
}

func (b *BootService) timeoutHandler(err bus.TimeoutError) error {
	b.log.Error("Timeout processing event", "event", err.Event())
	return nil
}

// https://github.com/grpc/grpc-go/issues/1229#issuecomment-302755717
func sendKeepPatientResponse(srv v1.BootService_WaitServer) error {
	errChan := make(chan error, 1)
	ctx := srv.Context()
	go func() {
		errChan <- srv.Send(&v1.BootServiceWaitResponse{})
		close(errChan)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}

func (b *BootService) handleAllocation(machineID string) {
	value, ok := b.queue.Load(machineID)
	if !ok {
		return
	}
	can, ok := value.(chan bool)
	if !ok {
		b.log.Error("handleAllocation: unable to cast queue entry to chan bool")
		return
	}
	can <- true
}

func (b *BootService) updateWaitingFlag(machineID string, flag bool) error {
	m, err := b.ds.FindMachineByID(machineID)
	if err != nil {
		return err
	}
	old := *m
	m.Waiting = flag
	return b.ds.UpdateMachine(&old, m)
}
