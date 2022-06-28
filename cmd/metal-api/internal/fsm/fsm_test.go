package fsm

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
	"gopkg.in/rethinkdb/rethinkdb-go.v6"
)

var store = func(mock *rethinkdb.Mock) {
	mock.On(rethinkdb.DB("mockdb").Table("event").Insert(rethinkdb.MockAnything(), rethinkdb.InsertOpts{
		Conflict: "replace",
	})).Return(testdata.EmptyResult, nil)
}

func TestHandleProvisioningEvent(t *testing.T) {
	now := time.Now()
	tests := []struct {
		event     *metal.ProvisioningEvent
		container *metal.ProvisioningEventContainer
		ds        func(mock *rethinkdb.Mock)
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
			ds:      store,
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
			ds:      store,
			wantErr: nil,
		},
		{
			name: "Transitioning from Planned Reboot to Phoned Home",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  now,
						Event: metal.ProvisioningEventPlannedReboot,
					},
				},
			},
			event: &metal.ProvisioningEvent{
				Event: metal.ProvisioningEventPhonedHome,
			},
			ds:      nil,
			wantErr: nil,
		},
		{
			name: "Transitioning from Planned Reboot to Phoned Home after too long a period of time",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  now,
						Event: metal.ProvisioningEventPlannedReboot,
					},
				},
			},
			event: &metal.ProvisioningEvent{
				Time:  now.Add(timeOutAfterPlannedReboot),
				Event: metal.ProvisioningEventPhonedHome,
			},
			ds:      nil,
			wantErr: fsm.CanceledError{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds, mock := datastore.InitMockDB()
			if tt.ds != nil {
				tt.ds(mock)
			}

			err := HandleProvisioningEvent(tt.event, tt.container, ds)
			if diff := cmp.Diff(tt.wantErr, err); diff != "" {
				t.Errorf("HandleProvisioningEvent() error = %v, wantErr %v", err, tt.wantErr)
			}
			mock.AssertExpectations(t)
		})
	}
}
