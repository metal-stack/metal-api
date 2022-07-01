package fsm

import (
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"go.uber.org/zap"
)

func TestHandleProvisioningEvent(t *testing.T) {
	now := time.Now()
	lastTimeEvent := now.Add(-time.Minute * 5)
	tests := []struct {
		event                *metal.ProvisioningEvent
		container            *metal.ProvisioningEventContainer
		name                 string
		wantErr              error
		wantIncompleteCycles int
		wantLiveliness       metal.MachineLiveliness
		wantNumberOfEvents   int
		wantLastEventTime    time.Time
		wantLastEvent        string
	}{
		{
			name: "Transitioning from Waiting to Installing",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  now.Add(-time.Minute * 5),
						Event: metal.ProvisioningEventWaiting,
					},
				},
				Liveliness: metal.MachineLivelinessAlive,
			},
			event: &metal.ProvisioningEvent{
				Time:  now,
				Event: metal.ProvisioningEventInstalling,
			},
			wantErr:              nil,
			wantIncompleteCycles: 0,
			wantLiveliness:       metal.MachineLivelinessAlive,
			wantNumberOfEvents:   2,
			wantLastEventTime:    now,
			wantLastEvent:        string(metal.ProvisioningEventInstalling),
		},
		{
			name: "Transitioning from PXEBooting to PXEBooting",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  now.Add(-time.Minute * 5),
						Event: metal.ProvisioningEventPXEBooting,
					},
				},
				Liveliness: metal.MachineLivelinessAlive,
			},
			event: &metal.ProvisioningEvent{
				Time:  now,
				Event: metal.ProvisioningEventPXEBooting,
			},
			wantErr:              nil,
			wantIncompleteCycles: 0,
			wantLiveliness:       metal.MachineLivelinessAlive,
			wantNumberOfEvents:   1,
			wantLastEventTime:    now,
			wantLastEvent:        string(metal.ProvisioningEventPXEBooting),
		},
		{
			name: "Transitioning from Registering to PXEBooting",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  now.Add(-time.Minute * 5),
						Event: metal.ProvisioningEventRegistering,
					},
				},
				Liveliness: metal.MachineLivelinessAlive,
			},
			event: &metal.ProvisioningEvent{
				Time:  now,
				Event: metal.ProvisioningEventPXEBooting,
			},
			wantErr:              fsm.InvalidEventError{Event: "PXE Booting", State: "Registering"},
			wantIncompleteCycles: 1,
			wantLiveliness:       metal.MachineLivelinessAlive,
			wantNumberOfEvents:   2,
			wantLastEventTime:    now,
			wantLastEvent:        string(metal.ProvisioningEventPXEBooting),
		},
		{
			name: "Registering to PXEBooting to Preparing",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  now.Add(-time.Minute * 5),
						Event: metal.ProvisioningEventRegistering,
					},
					{
						Time:  now.Add(-time.Minute * 2),
						Event: metal.ProvisioningEventPXEBooting,
					},
				},
				Liveliness: metal.MachineLivelinessAlive,
			},
			event: &metal.ProvisioningEvent{
				Time:  now,
				Event: metal.ProvisioningEventPreparing,
			},
			wantErr:              nil,
			wantIncompleteCycles: 1,
			wantLiveliness:       metal.MachineLivelinessAlive,
			wantNumberOfEvents:   3,
			wantLastEventTime:    now,
			wantLastEvent:        string(metal.ProvisioningEventPreparing),
		},
		{
			name: "Swallow repeated Phoned Home",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  now.Add(-time.Minute * 5),
						Event: metal.ProvisioningEventPhonedHome,
					},
				},
				Liveliness: metal.MachineLivelinessAlive,
			},
			event: &metal.ProvisioningEvent{
				Time:  now,
				Event: metal.ProvisioningEventPhonedHome,
			},
			wantErr:              nil,
			wantIncompleteCycles: 0,
			wantLiveliness:       metal.MachineLivelinessAlive,
			wantNumberOfEvents:   1,
			wantLastEventTime:    now,
			wantLastEvent:        string(metal.ProvisioningEventPhonedHome),
		},
		{
			name: "Swallow Phoned Home after Planned Reboot",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  now.Add(-time.Minute * 5),
						Event: metal.ProvisioningEventPlannedReboot,
					},
				},
				Liveliness:    metal.MachineLivelinessAlive,
				LastEventTime: &lastTimeEvent,
			},
			event: &metal.ProvisioningEvent{
				Time:  now,
				Event: metal.ProvisioningEventPhonedHome,
			},
			wantErr:              nil,
			wantIncompleteCycles: 0,
			wantLiveliness:       metal.MachineLivelinessAlive,
			wantNumberOfEvents:   1,
			wantLastEventTime:    now,
			wantLastEvent:        string(metal.ProvisioningEventPlannedReboot),
		},
		{
			name: "Liveliness unknown if Phoned Home after Planned Reboot timeout",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  now.Add(-time.Minute * 5),
						Event: metal.ProvisioningEventPlannedReboot,
					},
				},
				Liveliness:    metal.MachineLivelinessAlive,
				LastEventTime: &lastTimeEvent,
			},
			event: &metal.ProvisioningEvent{
				Time:  now.Add(time.Minute * 10),
				Event: metal.ProvisioningEventPhonedHome,
			},
			wantErr:              nil,
			wantIncompleteCycles: 0,
			wantLiveliness:       metal.MachineLivelinessUnknown,
			wantNumberOfEvents:   1,
			wantLastEventTime:    now.Add(time.Minute * 10),
			wantLastEvent:        string(metal.ProvisioningEventPlannedReboot),
		},
	}
	for _, tt := range tests {
		log := zap.NewExample().Sugar()
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			_, err := HandleProvisioningEvent(tt.event, tt.container, log)
			if diff := cmp.Diff(tt.wantErr, err); diff != "" {
				t.Errorf("HandleProvisioningEvent() diff = %s", diff)
			}

			incompleteCycles, err := parseContainerIncompleCycles(tt.container.IncompleteProvisioningCycles)
			if err != nil {
				t.Errorf(err.Error())
			}
			if incompleteCycles != tt.wantIncompleteCycles {
				t.Errorf("HandleProvisioningEvent() incomplete cycles got %d want %d", incompleteCycles, tt.wantIncompleteCycles)
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
		})
	}
}

func parseContainerIncompleCycles(cycles string) (int, error) {
	incompleteCycles, err := strconv.Atoi(cycles)
	if err != nil {
		return 0, err
	}

	return incompleteCycles, nil
}
