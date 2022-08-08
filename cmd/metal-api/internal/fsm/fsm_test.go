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
			name: "pxe booting is first event in container",
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
			name: "alive is first event in container",
			container: &metal.ProvisioningEventContainer{
				Events:     metal.ProvisioningEvents{},
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
				Events:               metal.ProvisioningEvents{},
			},
		},
		{
			name: "transition from PXE booting to PXE booting",
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
			name: "transition from PXE booting to preparing",
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
			name: "transition from booting new kernel to phoned home",
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
			name: "transition from registering to preparing",
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
			name: "swallow alive event",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  lastEventTime,
						Event: metal.ProvisioningEventWaiting,
					},
					{
						Time:  lastEventTime.Add(-1 * time.Minute),
						Event: metal.ProvisioningEventRegistering,
					},
					{
						Time:  lastEventTime.Add(-2 * time.Minute),
						Event: metal.ProvisioningEventPreparing,
					},
					{
						Time:  lastEventTime.Add(-3 * time.Minute),
						Event: metal.ProvisioningEventPXEBooting,
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
						Event: metal.ProvisioningEventWaiting,
					},
					{
						Time:  lastEventTime.Add(-1 * time.Minute),
						Event: metal.ProvisioningEventRegistering,
					},
					{
						Time:  lastEventTime.Add(-2 * time.Minute),
						Event: metal.ProvisioningEventPreparing,
					},
					{
						Time:  lastEventTime.Add(-3 * time.Minute),
						Event: metal.ProvisioningEventPXEBooting,
					},
				},
			},
		},
		{
			name: "swallow repeated phoned home",
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
			name: "swallow phoned home after machine reclaim",
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
			name: "failed machine reclaim",
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
			name: "reset failed reclaim flag with PXE booting event",
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
			name: "reset failed reclaim with with preparing event",
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
			got, err := HandleProvisioningEvent(zaptest.NewLogger(t).Sugar(), tt.container, tt.event)
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
