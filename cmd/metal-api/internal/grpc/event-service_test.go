package grpc

import (
	"context"
	"log/slog"
	"os"
	"reflect"
	"testing"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"

	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
)

func TestEventService_Send(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name    string
		req     *v1.EventServiceSendRequest
		ds      *datastore.RethinkStore
		log     *slog.Logger
		want    *v1.EventServiceSendResponse
		wantErr bool
	}{
		{
			name: "simple",
			req: &v1.EventServiceSendRequest{
				Events: map[string]*v1.MachineProvisioningEvent{
					"m1": {
						Event:   string(metal.ProvisioningEventPreparing),
						Message: "starting metal-hammer",
					},
				},
			},
			ds:  ds,
			log: log,
			want: &v1.EventServiceSendResponse{
				Events: uint64(1),
				Failed: []string{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			e := &EventService{
				log: tt.log,
				ds:  tt.ds,
			}

			got, err := e.Send(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("EventService.Send() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EventService.Send() = %v, want %v", got, tt.want)
			}
		})
	}
}
