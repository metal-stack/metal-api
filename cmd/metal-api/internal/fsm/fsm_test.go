package fsm

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"go.uber.org/zap"
)

func TestHandleProvisioningEvent(t *testing.T) {
	now := time.Now()
	lastTimeEvent := now.Add(-time.Minute * 4)
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
						Time:  lastTimeEvent,
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
						Time:  lastTimeEvent,
						Event: metal.ProvisioningEventPXEBooting,
					},
					{
						Time:  lastTimeEvent.Add(time.Minute * 2),
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
						Time:  lastTimeEvent,
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
						Time:  lastTimeEvent,
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
						Time:  lastTimeEvent,
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
			name: "Swallow repeated Phoned Home",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  lastTimeEvent,
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
						Time:  lastTimeEvent,
						Event: metal.ProvisioningEventMachineReclaim,
					},
				},
				Liveliness:    metal.MachineLivelinessAlive,
				LastEventTime: &lastTimeEvent,
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
			wantLastEventTime:  lastTimeEvent,
			wantLastEvent:      metal.ProvisioningEventMachineReclaim.String(),
		},
		{
			name: "Failed Machine Reclaim",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  lastTimeEvent,
						Event: metal.ProvisioningEventMachineReclaim,
					},
				},
				Liveliness:    metal.MachineLivelinessAlive,
				LastEventTime: &lastTimeEvent,
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
		log := zap.NewExample().Sugar()
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			_, err := handleProvisioningEvent(tt.event, tt.container, log)
			if diff := cmp.Diff(tt.wantErr, err); diff != "" {
				t.Errorf("HandleProvisioningEvent() diff = %s", diff)
			}

			if tt.container.CrashLoop != tt.wantCrashLoop {
				t.Errorf("HandleProvisioningEvent() machine crash loop got %v want %v", tt.container.CrashLoop, tt.wantCrashLoop)
			}

			if tt.container.FailedMachineReclaim != tt.wantFailedReclaim {
				t.Errorf("HandleProvisioningEvent() failed machine reclaim got %v want %v", tt.container.FailedMachineReclaim, tt.wantFailedReclaim)
			}

			if tt.container.Liveliness != tt.wantLiveliness {
				t.Errorf("HandleProvisioningEvent() machine liveliness got %v want %v", tt.container.Liveliness, tt.wantLiveliness)
			}

			if len(tt.container.Events) != tt.wantNumberOfEvents {
				t.Errorf("HandleProvisioningEvent() number of events got %d want %d", len(tt.container.Events), tt.wantNumberOfEvents)
			}

			if !tt.container.LastEventTime.Equal(tt.wantLastEventTime) {
				t.Errorf("HandleProvisioningEvent() last time event got %v want %v", tt.container.LastEventTime, tt.wantLastEventTime)
			}

			if tt.container.Events[len(tt.container.Events)-1].Event.String() != tt.wantLastEvent {
				t.Errorf("HandleProvisioningEvent() last event got %v want %v", tt.container.Events[len(tt.container.Events)-1].Event.String(), tt.wantLastEvent)
			}
		})
	}
}
