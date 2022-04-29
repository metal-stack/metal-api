package grpc

import (
	"context"
	"errors"
	"fmt"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"go.uber.org/zap"
)

type EventService struct {
	log *zap.SugaredLogger
	ds  Datasource
}

func NewEventService(cfg *ServerConfig) *EventService {
	return &EventService{
		ds:  cfg.Datasource,
		log: cfg.Logger.Named("event-service"),
	}
}
func (e *EventService) Send(ctx context.Context, req *v1.EventServiceSendRequest) (*v1.EventServiceSendResponse, error) {
	e.log.Infow("send", "event", req)
	if req == nil {
		return nil, fmt.Errorf("no event send")
	}
	m, err := e.ds.FindMachineByID(req.MachineId)
	if err != nil && !metal.IsNotFound(err) {
		return nil, err
	}

	// an event can actually create an empty machine. This enables us to also catch the very first PXE Booting event
	// in a machine lifecycle
	if m == nil {
		m = &metal.Machine{
			Base: metal.Base{
				ID: req.MachineId,
			},
		}
		err = e.ds.CreateMachine(m)
		if err != nil {
			return nil, err
		}
	}

	ok := metal.AllProvisioningEventTypes[metal.ProvisioningEventType(req.Event)]
	if !ok {
		return nil, errors.New("unknown provisioning event")
	}
	_, err = e.ds.ProvisioningEventForMachine(req.MachineId, req.Event, req.Message)
	if err != nil {
		return nil, err
	}
	return &v1.EventServiceSendResponse{}, nil
}
