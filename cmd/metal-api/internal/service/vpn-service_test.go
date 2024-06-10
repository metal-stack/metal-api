package service

import (
	"context"
	"log/slog"
	"testing"

	"github.com/google/go-cmp/cmp"
	headscalev1 "github.com/juanfont/headscale/gen/go/headscale/v1"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
	"github.com/metal-stack/metal-lib/pkg/testcommon"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

func Test_EvaluateVPNConnected(t *testing.T) {
	tests := []struct {
		name              string
		mockFn            func(mock *r.Mock)
		headscaleMachines []*headscalev1.Machine
		wantErr           error
	}{
		{
			name: "machines are correctly evaluated",
			mockFn: func(mock *r.Mock) {
				mock.On(r.DB("mockdb").Table("machine")).Return(metal.Machines{
					{
						Base: metal.Base{
							ID: "toggle",
						},
						Allocation: &metal.MachineAllocation{
							Project: "p1",
							VPN: &metal.MachineVPN{
								Connected: false,
							},
						},
					},
					{
						Base: metal.Base{
							ID: "already-connected",
						},
						Allocation: &metal.MachineAllocation{
							Project: "p2",
							VPN: &metal.MachineVPN{
								Connected: true,
							},
						},
					},
					{
						Base: metal.Base{
							ID: "no-vpn",
						},
						Allocation: &metal.MachineAllocation{
							Project: "p3",
						},
					},
				}, nil)

				// unfortunately, it's too hard to check the replace exactly for specific fields...
				mock.On(r.DB("mockdb").Table("machine").Get("toggle").Replace(r.MockAnything())).Return(testdata.EmptyResult, nil)
			},
			headscaleMachines: []*headscalev1.Machine{
				{
					Name: "toggle",
					User: &headscalev1.User{
						Name: "previous-allocation",
					},
					Online: false,
				},
				{
					Name: "toggle",
					User: &headscalev1.User{
						Name: "p1",
					},
					Online: true,
				},
				{
					Name: "already-connected",
					User: &headscalev1.User{
						Name: "p2",
					},
					Online: true,
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds, mock := datastore.InitMockDB(t)
			if tt.mockFn != nil {
				tt.mockFn(mock)
			}

			err := EvaluateVPNConnected(slog.Default(), ds, &headscaleTest{ms: tt.headscaleMachines})
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (-want +got):\n%s", diff)
			}

			mock.AssertExpectations(t)
		})
	}
}

type headscaleTest struct {
	ms []*headscalev1.Machine
}

func (h *headscaleTest) MachinesConnected(ctx context.Context) ([]*headscalev1.Machine, error) {
	return h.ms, nil
}
