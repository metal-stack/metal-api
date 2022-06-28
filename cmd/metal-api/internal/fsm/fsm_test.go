package fsm

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"go.uber.org/zap"
)

// this test shouldn't pass yet
func TestHandleProvisioningEvent(t *testing.T) {
	now := time.Now()
	tests := []struct {
		event     *metal.ProvisioningEvent
		container *metal.ProvisioningEventContainer
		name      string
		wantErr   error
	}{
		{
			name: "Transitioning from Waiting to Installing",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  now,
						Event: metal.ProvisioningEventWaiting,
					},
				},
			},
			event: &metal.ProvisioningEvent{
				Event: metal.ProvisioningEventInstalling,
			},
			wantErr: nil,
		},
		{
			name: "Transitioning from PXEBooting to PXEBooting",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  now,
						Event: metal.ProvisioningEventPXEBooting,
					},
				},
			},
			event: &metal.ProvisioningEvent{
				Event: metal.ProvisioningEventPXEBooting,
			},
			wantErr: nil,
		},
		{
			name: "Transitioning from Registering to PXEBooting",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  now,
						Event: metal.ProvisioningEventRegistering,
					},
				},
			},
			event: &metal.ProvisioningEvent{
				Event: metal.ProvisioningEventPXEBooting,
			},
			wantErr: errors.New("event PXE Booting inappropriate in current state Registering"), // errors look identical but test fails
		},
	}
	for _, tt := range tests {
		log := zap.NewExample().Sugar()
		t.Run(tt.name, func(t *testing.T) {
			_, err := HandleProvisioningEvent(tt.event, tt.container, log)
			if diff := cmp.Diff(tt.wantErr, err); diff != "" {
				t.Errorf("HandleProvisioningEvent() error = %v, wantErr: %v", err, tt.wantErr)
			}
		})
	}
}
