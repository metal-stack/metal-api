package fsm

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"go.uber.org/zap/zaptest"
)

func TestHandleProvisioningEvent(t *testing.T) {
	now := time.Now()
	lastEventTime := now.Add(-time.Minute * 4)
	exceedThresholdTime := now.Add(-time.Minute * 10)
	tests := []struct {
		event     *metal.ProvisioningEvent
		container *metal.ProvisioningEventContainer
		name      string
		wantErr   error
		want      *metal.ProvisioningEventContainer
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
			wantErr: nil,
			want: &metal.ProvisioningEventContainer{
				CrashLoop:            false,
				FailedMachineReclaim: false,
				Liveliness:           metal.MachineLivelinessAlive,
				LastEventTime:        &now,
				Events: metal.ProvisioningEvents{
					{
						Time:  now,
						Event: metal.ProvisioningEventPXEBooting,
					},
				},
			},
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
			wantErr: nil,
			want: &metal.ProvisioningEventContainer{
				CrashLoop:            false,
				FailedMachineReclaim: false,
				Liveliness:           metal.MachineLivelinessAlive,
				LastEventTime:        &now,
				Events: metal.ProvisioningEvents{
					{
						Time:  lastEventTime,
						Event: metal.ProvisioningEventPXEBooting,
					},
				},
			},
		},
		{
			name: "Transition from PXEBooting to Preparing",
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
				Event: metal.ProvisioningEventPreparing,
			},
			wantErr: nil,
			want: &metal.ProvisioningEventContainer{
				CrashLoop:            false,
				FailedMachineReclaim: false,
				Liveliness:           metal.MachineLivelinessAlive,
				LastEventTime:        &now,
				Events: metal.ProvisioningEvents{
					{
						Time:  now,
						Event: metal.ProvisioningEventPreparing,
					},
					{
						Time:  lastEventTime,
						Event: metal.ProvisioningEventPXEBooting,
					},
				},
			},
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
			wantErr: nil,
			want: &metal.ProvisioningEventContainer{
				CrashLoop:            false,
				FailedMachineReclaim: false,
				Liveliness:           metal.MachineLivelinessAlive,
				LastEventTime:        &now,
				Events: metal.ProvisioningEvents{
					{
						Time:  now,
						Event: metal.ProvisioningEventPhonedHome,
					},
					{
						Time:  lastEventTime,
						Event: metal.ProvisioningEventBootingNewKernel,
					},
				},
			},
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
			wantErr: nil,
			want: &metal.ProvisioningEventContainer{
				CrashLoop:            true,
				FailedMachineReclaim: false,
				Liveliness:           metal.MachineLivelinessAlive,
				LastEventTime:        &now,
				Events: metal.ProvisioningEvents{
					{
						Time:  now,
						Event: metal.ProvisioningEventPreparing,
					},
					{
						Time:  lastEventTime,
						Event: metal.ProvisioningEventRegistering,
					},
				},
			},
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
			wantErr: nil,
			want: &metal.ProvisioningEventContainer{
				CrashLoop:            false,
				FailedMachineReclaim: false,
				Liveliness:           metal.MachineLivelinessAlive,
				LastEventTime:        &now,
				Events: metal.ProvisioningEvents{
					{
						Time:  lastEventTime,
						Event: metal.ProvisioningEventRegistering,
					},
				},
			},
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
			wantErr: nil,
			want: &metal.ProvisioningEventContainer{
				CrashLoop:            false,
				FailedMachineReclaim: false,
				Liveliness:           metal.MachineLivelinessAlive,
				LastEventTime:        &now,
				Events: metal.ProvisioningEvents{
					{
						Time:  lastEventTime,
						Event: metal.ProvisioningEventPhonedHome,
					},
				},
			},
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
			wantErr: nil,
			want: &metal.ProvisioningEventContainer{
				CrashLoop:            false,
				FailedMachineReclaim: false,
				Liveliness:           metal.MachineLivelinessAlive,
				LastEventTime:        &lastEventTime,
				Events: metal.ProvisioningEvents{
					{
						Time:  lastEventTime,
						Event: metal.ProvisioningEventMachineReclaim,
					},
				},
			},
		},
		{
			name: "Failed Machine Reclaim",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  exceedThresholdTime,
						Event: metal.ProvisioningEventMachineReclaim,
					},
				},
				Liveliness:    metal.MachineLivelinessAlive,
				LastEventTime: &exceedThresholdTime,
			},
			event: &metal.ProvisioningEvent{
				Time:  now,
				Event: metal.ProvisioningEventPhonedHome,
			},
			wantErr: nil,
			want: &metal.ProvisioningEventContainer{
				CrashLoop:            false,
				FailedMachineReclaim: true,
				Liveliness:           metal.MachineLivelinessAlive,
				LastEventTime:        &now,
				Events: metal.ProvisioningEvents{
					{
						Time:  exceedThresholdTime,
						Event: metal.ProvisioningEventMachineReclaim,
					},
				},
			},
		},
		{
			name: "Reset Failed Reclaim flag with PXE Booting event",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  lastEventTime,
						Event: metal.ProvisioningEventMachineReclaim,
					},
				},
				Liveliness:           metal.MachineLivelinessAlive,
				LastEventTime:        &lastEventTime,
				FailedMachineReclaim: true,
			},
			event: &metal.ProvisioningEvent{
				Time:  now,
				Event: metal.ProvisioningEventPXEBooting,
			},
			wantErr: nil,
			want: &metal.ProvisioningEventContainer{
				CrashLoop:            false,
				FailedMachineReclaim: false,
				Liveliness:           metal.MachineLivelinessAlive,
				LastEventTime:        &now,
				Events: metal.ProvisioningEvents{
					{
						Time:  now,
						Event: metal.ProvisioningEventPXEBooting,
					},
					{
						Time:  lastEventTime,
						Event: metal.ProvisioningEventMachineReclaim,
					},
				},
			},
		},
		{
			name: "Reset Failed Reclaim with with Preparing event",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  lastEventTime,
						Event: metal.ProvisioningEventMachineReclaim,
					},
				},
				Liveliness:           metal.MachineLivelinessAlive,
				LastEventTime:        &lastEventTime,
				FailedMachineReclaim: true,
			},
			event: &metal.ProvisioningEvent{
				Time:  now,
				Event: metal.ProvisioningEventPreparing,
			},
			wantErr: nil,
			want: &metal.ProvisioningEventContainer{
				CrashLoop:            false,
				FailedMachineReclaim: false,
				Liveliness:           metal.MachineLivelinessAlive,
				LastEventTime:        &now,
				Events: metal.ProvisioningEvents{
					{
						Time:  now,
						Event: metal.ProvisioningEventPreparing,
					},
					{
						Time:  lastEventTime,
						Event: metal.ProvisioningEventMachineReclaim,
					},
				},
			},
		},
		{
			name: "Reset Crash Loop flag with Phoned Home event",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  lastEventTime,
						Event: metal.ProvisioningEventBootingNewKernel,
					},
				},
				Liveliness:    metal.MachineLivelinessAlive,
				LastEventTime: &lastEventTime,
				CrashLoop:     true,
			},
			event: &metal.ProvisioningEvent{
				Time:  now,
				Event: metal.ProvisioningEventPhonedHome,
			},
			wantErr: nil,
			want: &metal.ProvisioningEventContainer{
				CrashLoop:            false,
				FailedMachineReclaim: false,
				Liveliness:           metal.MachineLivelinessAlive,
				LastEventTime:        &now,
				Events: metal.ProvisioningEvents{
					{
						Time:  now,
						Event: metal.ProvisioningEventPhonedHome,
					},
					{
						Time:  lastEventTime,
						Event: metal.ProvisioningEventBootingNewKernel,
					},
				},
			},
		},
		{
			name: "Reset Crash Loop flag with Planned Reboot event",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  lastEventTime,
						Event: metal.ProvisioningEventRegistering,
					},
				},
				Liveliness:    metal.MachineLivelinessAlive,
				LastEventTime: &lastEventTime,
				CrashLoop:     true,
			},
			event: &metal.ProvisioningEvent{
				Time:  now,
				Event: metal.ProvisioningEventPlannedReboot,
			},
			wantErr: nil,
			want: &metal.ProvisioningEventContainer{
				CrashLoop:            false,
				FailedMachineReclaim: false,
				Liveliness:           metal.MachineLivelinessAlive,
				LastEventTime:        &now,
				Events: metal.ProvisioningEvents{
					{
						Time:  now,
						Event: metal.ProvisioningEventPlannedReboot,
					},
					{
						Time:  lastEventTime,
						Event: metal.ProvisioningEventRegistering,
					},
				},
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkProvisioningEvent(tt.event, tt.container, zaptest.NewLogger(t).Sugar())
			if diff := cmp.Diff(tt.wantErr, err); diff != "" {
				t.Errorf("HandleProvisioningEvent() diff = %s", diff)
			}

			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("HandleProvisioningEvent() diff = %s", diff)
			}

			if err = got.Validate(); err != nil {
				t.Errorf("HandleProvisioningEvent() Validate error = %s", err)
			}
		})
	}
}
