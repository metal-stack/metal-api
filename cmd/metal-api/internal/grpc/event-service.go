package grpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"go.uber.org/zap"
)

type EventService struct {
	log *zap.SugaredLogger
	ds  *datastore.RethinkStore
}

func NewEventService(cfg *ServerConfig) *EventService {
	return &EventService{
		ds:  cfg.Store,
		log: cfg.Logger.Named("event-service"),
	}
}
func (e *EventService) Send(ctx context.Context, req *v1.EventServiceSendRequest) (*v1.EventServiceSendResponse, error) {
	e.log.Debugw("send", "event", req)
	if req == nil {
		return nil, fmt.Errorf("no event send")
	}

	failed := []string{}
	processed := uint64(0)
	var processErrs []error
	for machineID, event := range req.Events {

		m, err := e.ds.FindMachineByID(machineID)
		if err != nil && !metal.IsNotFound(err) {
			processErrs = append(processErrs, fmt.Errorf("machine with ID:%s not found %w", machineID, err))
			failed = append(failed, machineID)
			continue
		}

		// an event can actually create an empty machine. This enables us to also catch the very first PXE Booting event
		// in a machine lifecycle
		if m == nil {
			m = &metal.Machine{
				Base: metal.Base{
					ID: machineID,
				},
			}
			err = e.ds.CreateMachine(m)
			if err != nil {
				processErrs = append(processErrs, err)
				failed = append(failed, machineID)
				continue
			}
		}

		ok := metal.AllProvisioningEventTypes[metal.ProvisioningEventType(event.Event)]
		if !ok {
			processErrs = append(processErrs, err)
			failed = append(failed, machineID)
			continue
		}

		ev := metal.ProvisioningEvent{
			Time:    time.Now(),
			Event:   metal.ProvisioningEventType(event.Event),
			Message: event.Message,
		}

		_, err = e.ds.ProvisioningEventForMachine(e.log, &ev, machineID)
		if err != nil {
			processErrs = append(processErrs, err)
			failed = append(failed, machineID)
			continue
		}
		processed++
	}

	return &v1.EventServiceSendResponse{
		Events: processed,
		Failed: failed,
	}, errors.Join(processErrs...)
}
