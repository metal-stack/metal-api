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
			name: "valid transition from PXE booting to PXE booting (swallow repeated)",
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
			name: "valid transition from PXE booting to preparing",
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
			name: "valid transition from booting new kernel to phoned home",
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
			name: "valid transition from installing to crashing, going into crash loop",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  lastEventTime,
						Event: metal.ProvisioningEventInstalling,
					},
				},
				Liveliness: metal.MachineLivelinessAlive,
			},
			event: &metal.ProvisioningEvent{
				Time:  now,
				Event: metal.ProvisioningEventCrashed,
			},
			wantErr: nil,
			want: &metal.ProvisioningEventContainer{
				CrashLoop:            true,
				FailedMachineReclaim: false,
				Liveliness:           metal.MachineLivelinessAlive,
				LastEventTime:        &now,
				LastErrorEvent: &metal.ProvisioningEvent{
					Time:  now,
					Event: metal.ProvisioningEventCrashed,
				},
				Events: metal.ProvisioningEvents{
					{
						Time:  now,
						Event: metal.ProvisioningEventCrashed,
					},
					{
						Time:  lastEventTime,
						Event: metal.ProvisioningEventInstalling,
					},
				},
			},
		},
		{
			name: "valid transition from crashing to pxe booting, maintaing crash loop",
			container: &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Time:  lastEventTime,
						Event: metal.ProvisioningEventCrashed,
					},
					{
						Time:  lastEventTime.Add(-1 * time.Minute),
						Event: metal.ProvisioningEventInstalling,
					},
				},
				CrashLoop:  true,
				Liveliness: metal.MachineLivelinessAlive,
			},
			event: &metal.ProvisioningEvent{
				Time:  now,
				Event: metal.ProvisioningEventPXEBooting,
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
						Event: metal.ProvisioningEventPXEBooting,
					},
					{
						Time:  lastEventTime,
						Event: metal.ProvisioningEventCrashed,
					},
					{
						Time:  lastEventTime.Add(-1 * time.Minute),
						Event: metal.ProvisioningEventInstalling,
					},
				},
			},
		},
		{
			name: "invalid transition from registering to preparing",
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
				LastErrorEvent: &metal.ProvisioningEvent{
					Time:    now,
					Event:   metal.ProvisioningEventPreparing,
					Message: "[unexpectedly received in state registering]",
				},
				Events: metal.ProvisioningEvents{
					{
						Time:    now,
						Event:   metal.ProvisioningEventPreparing,
						Message: "[unexpectedly received in state registering]",
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
			name: "reset crash loop flag with phoned home event",
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
			name: "reset crash loop flag with planned reboot event",
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
		{
			name: "unexpected arrival of alive event",
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
				Time:    now,
				Event:   metal.ProvisioningEventAlive,
				Message: "sending alive",
			},
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
				LastErrorEvent: &metal.ProvisioningEvent{
					Time:    now,
					Event:   metal.ProvisioningEventAlive,
					Message: "[unexpectedly received in state phoned home]: sending alive",
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

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("HandleProvisioningEvent() diff = %s", diff)
			}

			if err = got.Validate(); err != nil {
				t.Errorf("HandleProvisioningEvent() Validate error = %s", err)
			}
		})
	}
}

func TestReactionToAllIncomingEvents(t *testing.T) {
	// this test ensures that for every incoming event we have a proper transition
	for e1 := range metal.AllProvisioningEventTypes {
		for e2 := range metal.AllProvisioningEventTypes {
			_, err := HandleProvisioningEvent(zaptest.NewLogger(t).Sugar(), &metal.ProvisioningEventContainer{
				Events: metal.ProvisioningEvents{
					{
						Event: e2,
					},
				},
			}, &metal.ProvisioningEvent{
				Event: e1,
			})
			if err != nil {
				t.Errorf("transitioning from state %s with event %s: %s", e2, e1, err)
			}
		}
	}
}
