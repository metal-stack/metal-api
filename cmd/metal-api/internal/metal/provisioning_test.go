package metal

import (
	"testing"
	"time"
)

func TestProvisioningEventContainer_Validate(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name      string
		container ProvisioningEventContainer
		wantErr   bool
	}{
		{
			name: "Validate empty container",
			container: ProvisioningEventContainer{
				Events: ProvisioningEvents{},
			},
			wantErr: false,
		},
		{
			name: "Validate sorted and consistent container",
			container: ProvisioningEventContainer{
				Events: ProvisioningEvents{
					ProvisioningEvent{
						Time: now.Add(-2 * time.Minute),
					},
					ProvisioningEvent{
						Time: now.Add(-3 * time.Minute),
					},
					ProvisioningEvent{
						Time: now.Add(-4 * time.Minute),
					},
					ProvisioningEvent{
						Time: now.Add(-5 * time.Minute),
					},
				},
				LastEventTime: &now,
			},
			wantErr: false,
		},
		{
			name: "Validate container with one event",
			container: ProvisioningEventContainer{
				Events: ProvisioningEvents{
					ProvisioningEvent{
						Time: now,
					},
				},
				LastEventTime: &now,
			},
			wantErr: false,
		},
		{
			name: "Validate container with empty last event time field",
			container: ProvisioningEventContainer{
				Events: ProvisioningEvents{
					ProvisioningEvent{
						Time: now,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Validate unsorted container",
			container: ProvisioningEventContainer{
				Events: ProvisioningEvents{
					ProvisioningEvent{
						Time: now.Add(-2 * time.Minute),
					},
					ProvisioningEvent{
						Time: now.Add(-4 * time.Minute),
					},
					ProvisioningEvent{
						Time: now.Add(-3 * time.Minute),
					},
					ProvisioningEvent{
						Time: now.Add(-5 * time.Minute),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Validate inconsistent last event times",
			container: ProvisioningEventContainer{
				Events: ProvisioningEvents{
					ProvisioningEvent{
						Time: now.Add(1 * time.Minute),
					},
					ProvisioningEvent{
						Time: now.Add(-3 * time.Minute),
					},
					ProvisioningEvent{
						Time: now.Add(-4 * time.Minute),
					},
					ProvisioningEvent{
						Time: now.Add(-5 * time.Minute),
					},
				},
				LastEventTime: &now,
			},
			wantErr: true,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.container.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("ProvisioningEventContainer.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
