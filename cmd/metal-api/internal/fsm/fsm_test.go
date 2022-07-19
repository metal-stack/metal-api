package fsm

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestHandleProvisioningEvent(t *testing.T) {
	now := time.Now()
	lastEventTime := now.Add(-time.Minute * 4)
	tests := []struct {
		event              *metal.ProvisioningEvent
		container          *metal.ProvisioningEventContainer
		name               string
		wantErr            error
		wantCrashLoop      bool
		wantFailedReclaim  bool
		wantLiveliness     metal.MachineLiveliness
		wantNumberOfEvents int
		wantLastEventTime  time.Time
		wantLastEvent      string
	}{
		{
			name: "First Event in container",
			container: &metal.ProvisioningEventContainer{
				Events:     metal.ProvisioningEvents{},
				Liveliness: metal.MachineLivelinessAlive,
			},
			event: &metal.ProvisioningEvent{
				Time:  now,
				Event: metal.ProvisioningEventPXEBooting,
			},
			wantErr:            nil,
			wantCrashLoop:      false,
			wantFailedReclaim:  false,
			wantLiveliness:     metal.MachineLivelinessAlive,
			wantNumberOfEvents: 1,
			wantLastEventTime:  now,
			wantLastEvent:      metal.ProvisioningEventPXEBooting.String(),
		},
		{
			name: "Transition from PXEBooting to PXEBooting",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  lastEventTime,
						Event: metal.ProvisioningEventPXEBooting,
					},
				},
				Liveliness: metal.MachineLivelinessAlive,
			},
			event: &metal.ProvisioningEvent{
				Time:  now,
				Event: metal.ProvisioningEventPXEBooting,
			},
			wantErr:            nil,
			wantCrashLoop:      false,
			wantFailedReclaim:  false,
			wantLiveliness:     metal.MachineLivelinessAlive,
			wantNumberOfEvents: 1,
			wantLastEventTime:  now,
			wantLastEvent:      metal.ProvisioningEventPXEBooting.String(),
		},
		{
			name: "Transition from PXEBooting to Preparing",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  lastEventTime,
						Event: metal.ProvisioningEventPXEBooting,
					},
					{
						Time:  lastEventTime.Add(time.Minute * 2),
						Event: metal.ProvisioningEventPXEBooting,
					},
				},
				Liveliness: metal.MachineLivelinessAlive,
			},
			event: &metal.ProvisioningEvent{
				Time:  now,
				Event: metal.ProvisioningEventPreparing,
			},
			wantErr:            nil,
			wantCrashLoop:      false,
			wantFailedReclaim:  false,
			wantLiveliness:     metal.MachineLivelinessAlive,
			wantNumberOfEvents: 3,
			wantLastEventTime:  now,
			wantLastEvent:      metal.ProvisioningEventPreparing.String(),
		},
		{
			name: "Transition from Booting New Kernel to Phoned Home",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  lastEventTime,
						Event: metal.ProvisioningEventBootingNewKernel,
					},
				},
				Liveliness: metal.MachineLivelinessAlive,
			},
			event: &metal.ProvisioningEvent{
				Time:  now,
				Event: metal.ProvisioningEventPhonedHome,
			},
			wantErr:            nil,
			wantCrashLoop:      false,
			wantFailedReclaim:  false,
			wantLiveliness:     metal.MachineLivelinessAlive,
			wantNumberOfEvents: 2,
			wantLastEventTime:  now,
			wantLastEvent:      metal.ProvisioningEventPhonedHome.String(),
		},
		{
			name: "Transition from Registering to Preparing",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  lastEventTime,
						Event: metal.ProvisioningEventRegistering,
					},
				},
				Liveliness: metal.MachineLivelinessAlive,
			},
			event: &metal.ProvisioningEvent{
				Time:  now,
				Event: metal.ProvisioningEventPreparing,
			},
			wantErr:            nil,
			wantCrashLoop:      true,
			wantFailedReclaim:  false,
			wantLiveliness:     metal.MachineLivelinessAlive,
			wantNumberOfEvents: 2,
			wantLastEventTime:  now,
			wantLastEvent:      metal.ProvisioningEventPreparing.String(),
		},
		{
			name: "Swallow Alive event",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  lastEventTime,
						Event: metal.ProvisioningEventRegistering,
					},
				},
				Liveliness: metal.MachineLivelinessAlive,
			},
			event: &metal.ProvisioningEvent{
				Time:  now,
				Event: metal.ProvisioningEventAlive,
			},
			wantErr:            nil,
			wantCrashLoop:      false,
			wantFailedReclaim:  false,
			wantLiveliness:     metal.MachineLivelinessAlive,
			wantNumberOfEvents: 1,
			wantLastEventTime:  now,
			wantLastEvent:      metal.ProvisioningEventRegistering.String(),
		},
		{
			name: "Swallow repeated Phoned Home",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  lastEventTime,
						Event: metal.ProvisioningEventPhonedHome,
					},
				},
				Liveliness: metal.MachineLivelinessAlive,
			},
			event: &metal.ProvisioningEvent{
				Time:  now,
				Event: metal.ProvisioningEventPhonedHome,
			},
			wantErr:            nil,
			wantCrashLoop:      false,
			wantFailedReclaim:  false,
			wantLiveliness:     metal.MachineLivelinessAlive,
			wantNumberOfEvents: 1,
			wantLastEventTime:  now,
			wantLastEvent:      metal.ProvisioningEventPhonedHome.String(),
		},
		{
			name: "Swallow Phoned Home after Machine Reclaim",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  lastEventTime,
						Event: metal.ProvisioningEventMachineReclaim,
					},
				},
				Liveliness:    metal.MachineLivelinessAlive,
				LastEventTime: &lastEventTime,
			},
			event: &metal.ProvisioningEvent{
				Time:  now,
				Event: metal.ProvisioningEventPhonedHome,
			},
			wantErr:            nil,
			wantCrashLoop:      false,
			wantFailedReclaim:  false,
			wantLiveliness:     metal.MachineLivelinessAlive,
			wantNumberOfEvents: 1,
			wantLastEventTime:  lastEventTime,
			wantLastEvent:      metal.ProvisioningEventMachineReclaim.String(),
		},
		{
			name: "Failed Machine Reclaim",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  lastEventTime,
						Event: metal.ProvisioningEventMachineReclaim,
					},
				},
				Liveliness:    metal.MachineLivelinessAlive,
				LastEventTime: &lastEventTime,
			},
			event: &metal.ProvisioningEvent{
				Time:  now.Add(time.Minute * 10),
				Event: metal.ProvisioningEventPhonedHome,
			},
			wantErr:            nil,
			wantCrashLoop:      false,
			wantFailedReclaim:  true,
			wantLiveliness:     metal.MachineLivelinessAlive,
			wantNumberOfEvents: 1,
			wantLastEventTime:  now.Add(time.Minute * 10),
			wantLastEvent:      metal.ProvisioningEventMachineReclaim.String(),
		},
	}
	for _, tt := range tests {
		tt := tt
		log := zap.NewExample().Sugar()
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkProvisioningEvent(tt.event, tt.container, log)
			if diff := cmp.Diff(tt.wantErr, err); diff != "" {
				t.Errorf("HandleProvisioningEvent() diff = %s", diff)
			}

			assert.Equal(t, tt.wantCrashLoop, got.CrashLoop, "got unexpected value for crash loop")
			assert.Equal(t, tt.wantFailedReclaim, got.FailedMachineReclaim, "got unexpected value for failed machine reclaim")
			assert.Equal(t, tt.wantLiveliness, got.Liveliness, "got unexpected value for liveliness")
			assert.Equal(t, tt.wantNumberOfEvents, len(got.Events), "got unexpected number of events")
			assert.WithinDuration(t, tt.wantLastEventTime, *got.LastEventTime, 0, "got unexpected last event time")
			assert.Equal(t, tt.wantLastEvent, got.Events[len(got.Events)-1].Event.String(), "got unexpected last event")
		})
	}
}
