package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	restful "github.com/emicklei/go-restful/v3"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/metal-lib/pkg/pointer"
)

func TestRegisterSwitch(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	switchservice := NewSwitch(log, ds)
	container := restful.NewContainer().Add(switchservice)

	name := "switch999"
	createRequest := v1.SwitchRegisterRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: "switch999",
			},
			Describable: v1.Describable{
				Name: &name,
			},
		},
		PartitionID: "1",
		SwitchBase: v1.SwitchBase{
			RackID: "1",
		},
	}
	js, err := json.Marshal(createRequest)
	require.NoError(t, err)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/switch/register", body)
	req.Header.Add("Content-Type", "application/json")
	container = injectEditor(log, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode, w.Body.String())
	var result v1.SwitchResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, "switch999", result.ID)
	require.Equal(t, "switch999", *result.Name)
	require.Equal(t, "1", result.RackID)
	require.Equal(t, "1", result.Partition.ID)
	require.Empty(t, result.Connections)
}

func TestRegisterExistingSwitch(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	switchservice := NewSwitch(log, ds)
	container := restful.NewContainer().Add(switchservice)

	createRequest := v1.SwitchRegisterRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: testdata.Switch2.ID,
			},
		},
		PartitionID: testdata.Switch2.PartitionID,
		SwitchBase: v1.SwitchBase{
			RackID: testdata.Switch2.RackID,
		},
	}
	js, err := json.Marshal(createRequest)
	require.NoError(t, err)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/switch/register", body)
	req.Header.Add("Content-Type", "application/json")
	container = injectEditor(log, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.SwitchResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.Switch2.ID, result.ID)
	require.Equal(t, testdata.Switch2.Name, *result.Name)
	require.Equal(t, testdata.Switch2.RackID, result.RackID)
	require.Equal(t, testdata.Switch2.PartitionID, result.Partition.ID)
	require.Empty(t, result.Connections)
	// con := result.Connections[0]
	// require.Equal(t, testdata.Switch2.MachineConnections["1"][0].Nic.MacAddress, con.Nic.MacAddress)
}

func TestRegisterExistingSwitchErrorModifyingNics(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	switchservice := NewSwitch(log, ds)
	container := restful.NewContainer().Add(switchservice)

	createRequest := v1.SwitchRegisterRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: testdata.Switch1.ID,
			},
		},
		Nics:        v1.SwitchNics{},
		PartitionID: testdata.Switch1.PartitionID,
		SwitchBase: v1.SwitchBase{
			RackID: testdata.Switch1.RackID,
		},
	}
	js, err := json.Marshal(createRequest)
	require.NoError(t, err)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/switch/register", body)
	container = injectAdmin(log, container, req)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)
}

func TestReplaceSwitch(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	switchservice := NewSwitch(log, ds)
	container := restful.NewContainer().Add(switchservice)

	createRequest := v1.SwitchRegisterRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: testdata.Switch2.ID,
			},
		},
		PartitionID: testdata.Switch2.PartitionID,
		SwitchBase: v1.SwitchBase{
			RackID: testdata.Switch2.RackID,
		},
	}
	js, err := json.Marshal(createRequest)
	require.NoError(t, err)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/switch/register", body)
	req.Header.Add("Content-Type", "application/json")
	container = injectEditor(log, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.SwitchResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.Switch2.ID, result.ID)
	require.Equal(t, testdata.Switch2.Name, *result.Name)
	require.Equal(t, testdata.Switch2.RackID, result.RackID)
	require.Equal(t, testdata.Switch2.PartitionID, result.Partition.ID)
	require.Empty(t, result.Connections)
}

func TestSwitchMigrateConnectionsExistError(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	switchservice := NewSwitch(log, ds)
	container := restful.NewContainer().Add(switchservice)

	migrateRequest := v1.SwitchMigrateRequest{
		OldSwitchID: testdata.Switch2.ID,
		NewSwitchID: testdata.Switch1.ID,
	}
	js, err := json.Marshal(migrateRequest)
	require.NoError(t, err)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/switch/migrate", body)
	req.Header.Add("Content-Type", "application/json")
	container = injectEditor(log, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, w.Body.String())
	var errorResponse httperrors.HTTPErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errorResponse)
	require.NoError(t, err)
	require.Equal(t, "target switch already has machine connections", errorResponse.Message)
	require.Equal(t, http.StatusBadRequest, errorResponse.StatusCode)
}

func TestSwitchMigrateDifferentRacksError(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	switchservice := NewSwitch(log, ds)
	container := restful.NewContainer().Add(switchservice)

	migrateRequest := v1.SwitchMigrateRequest{
		OldSwitchID: testdata.Switch1.ID,
		NewSwitchID: testdata.Switch3.ID,
	}
	js, err := json.Marshal(migrateRequest)
	require.NoError(t, err)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/switch/migrate", body)
	req.Header.Add("Content-Type", "application/json")
	container = injectEditor(log, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, w.Body.String())
	var errorResponse httperrors.HTTPErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errorResponse)
	require.NoError(t, err)
	require.Equal(t, "new switch must be in the same rack as the old one", errorResponse.Message)
	require.Equal(t, http.StatusBadRequest, errorResponse.StatusCode)
}

func TestConnectMachineWithSwitches(t *testing.T) {
	partitionID := "1"
	s1swp1 := metal.Nic{
		Name:       "swp1",
		MacAddress: "11:11:11:11:11:11",
	}
	s1 := metal.Switch{
		Base:               metal.Base{ID: "1"},
		PartitionID:        partitionID,
		OS:                 &metal.SwitchOS{Vendor: metal.SwitchOSVendorCumulus},
		MachineConnections: metal.ConnectionMap{},
		Nics: metal.Nics{
			s1swp1,
		},
	}
	s2swp1 := metal.Nic{
		Name:       "swp1",
		MacAddress: "21:11:11:11:11:11",
	}
	s2 := metal.Switch{
		Base:               metal.Base{ID: "2"},
		PartitionID:        partitionID,
		OS:                 &metal.SwitchOS{Vendor: metal.SwitchOSVendorCumulus},
		MachineConnections: metal.ConnectionMap{},
		Nics: metal.Nics{
			s2swp1,
		},
	}
	testSwitches := []metal.Switch{s1, s2}
	tests := []struct {
		name    string
		machine *metal.Machine
		wantErr bool
	}{
		{
			name: "Connect machine with uplinks to two distinct switches",
			machine: &metal.Machine{
				Base:        metal.Base{ID: "1"},
				PartitionID: partitionID,
				Hardware: metal.MachineHardware{
					Nics: metal.Nics{
						metal.Nic{
							Name: "lan0",
							Neighbors: metal.Nics{
								s1swp1,
							},
						},
						metal.Nic{
							Name: "lan1",
							Neighbors: metal.Nics{
								s2swp1,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Connect machine without neighbors on one interface",
			machine: &metal.Machine{
				Base:        metal.Base{ID: "2"},
				PartitionID: partitionID,
				Hardware: metal.MachineHardware{
					Nics: metal.Nics{
						metal.Nic{
							Name: "lan0",
							Neighbors: metal.Nics{
								s1swp1,
							},
						},
						metal.Nic{
							Name:      "lan1",
							Neighbors: metal.Nics{},
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for i := range tests {
		tt := tests[i]
		ds, mock := datastore.InitMockDB(t)
		mock.On(r.DB("mockdb").Table("switch")).Return(testSwitches, nil)
		mock.On(r.DB("mockdb").Table("switch").Get(r.MockAnything()).Replace(r.MockAnything())).Return(testdata.EmptyResult, nil)

		t.Run(tt.name, func(t *testing.T) {
			if err := ds.ConnectMachineWithSwitches(tt.machine); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.connectMachineWithSwitches() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
		// mock.AssertExpectations(t)
	}
}

func TestSetVrfAtSwitch(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	sw := metal.Switch{
		PartitionID: "1",
		Nics:        metal.Nics{metal.Nic{MacAddress: metal.MacAddress("11:11:11:11:11:11")}},
		MachineConnections: metal.ConnectionMap{
			"1": metal.Connections{
				metal.Connection{
					Nic: metal.Nic{
						MacAddress: metal.MacAddress("11:11:11:11:11:11"),
					},
					MachineID: "1",
				},
				metal.Connection{
					Nic: metal.Nic{
						MacAddress: metal.MacAddress("11:11:11:11:11:22"),
					},
					MachineID: "1",
				},
			},
		},
	}
	sws := []metal.Switch{sw}
	mock.On(r.DB("mockdb").Table("switch").Filter(r.MockAnything())).Return(sws, nil)
	mock.On(r.DB("mockdb").Table("switch").Get(r.MockAnything()).Replace(r.MockAnything())).Return(testdata.EmptyResult, nil)

	vrf := "123"
	m := &metal.Machine{
		Base:        metal.Base{ID: "1"},
		PartitionID: "1",
	}
	switches, err := ds.SetVrfAtSwitches(m, vrf)
	require.NoError(t, err, "no error was expected: got %v", err)
	require.Len(t, switches, 1)
	for _, s := range switches {
		require.Equal(t, vrf, s.Nics[0].Vrf)
	}
	mock.AssertExpectations(t)
}

func TestMakeBGPFilterFirewall(t *testing.T) {
	type args struct {
		machine metal.Machine
	}
	tests := []struct {
		name string
		args args
		want v1.BGPFilter
	}{
		{
			name: "valid firewall networks with underlay",
			args: args{
				machine: metal.Machine{
					Allocation: &metal.MachineAllocation{
						MachineNetworks: []*metal.MachineNetwork{
							{
								IPs: []string{},
								Vrf: 104010,
							},
							{
								IPs:      []string{"10.0.0.2", "10.0.0.1"},
								Vrf:      0,
								Underlay: true,
							},
							{
								IPs: []string{"212.89.42.1", "212.89.42.2"},
								Vrf: 104009,
							},
							{
								IPs: []string{"2001::", "fe80::"},
								Vrf: 104011,
							},
							{
								IPs:      []string{"2002::", "fe81::"},
								Underlay: true,
								Vrf:      104012,
							},
						},
					},
				},
			},
			want: v1.NewBGPFilter([]string{"104009", "104010", "104011"}, []string{"10.0.0.1/32", "10.0.0.2/32", "2002::/128", "fe81::/128"}),
		},
		{
			name: "no underlay firewall networks",
			args: args{
				machine: metal.Machine{
					Allocation: &metal.MachineAllocation{
						MachineNetworks: []*metal.MachineNetwork{
							{
								IPs:      []string{"10.0.0.1"},
								Vrf:      104010,
								Underlay: false,
							},
						},
					},
				},
			},
			want: v1.BGPFilter{
				VNIs:  []string{"104010"},
				CIDRs: []string{},
			},
		},
		{
			name: "empty firewall networks",
			args: args{
				machine: metal.Machine{
					Allocation: &metal.MachineAllocation{
						MachineNetworks: []*metal.MachineNetwork{},
					},
				},
			},
			want: v1.BGPFilter{
				VNIs:  []string{},
				CIDRs: []string{},
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			r := switchResource{}
			got, _ := r.makeBGPFilterFirewall(tt.args.machine)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("makeBGPFilterFirewall() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMakeBGPFilterMachine(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)

	type args struct {
		machine metal.Machine
		ipsMap  metal.IPsMap
		nws     metal.NetworkMap
	}
	tests := []struct {
		name string
		args args
		want v1.BGPFilter
	}{
		{
			name: "valid machine networks",
			args: args{
				ipsMap: metal.IPsMap{"project": metal.IPs{
					metal.IP{
						IPAddress: "212.89.42.1",
					},
					metal.IP{
						IPAddress: "212.89.42.2",
					},
					metal.IP{
						IPAddress: "100.127.1.1",
					},
					metal.IP{
						IPAddress: "10.1.0.1",
					},
					metal.IP{
						IPAddress: "2001::1",
					},
				}},
				nws: metal.NetworkMap{
					"tenant-super": metal.Network{
						PrivateSuper:               true,
						AdditionalAnnouncableCIDRs: []string{"10.240.0.0/12"},
					},
					"1": metal.Network{
						Base: metal.Base{ID: "1"},
						Prefixes: metal.Prefixes{
							{IP: "10.2.0.0", Length: "22"},
							{IP: "10.1.0.0", Length: "22"},
						},
						ParentNetworkID: "tenant-super",
					},
				},
				machine: metal.Machine{
					Allocation: &metal.MachineAllocation{
						Project: "project",
						MachineNetworks: []*metal.MachineNetwork{
							{
								NetworkID: "1",
								IPs:       []string{"10.1.0.1"},
								Prefixes:  []string{"10.2.0.0/22", "10.1.0.0/22"},
								Vrf:       1234,
								Private:   true,
							},
							{
								NetworkID: "2",
								IPs:       []string{"10.0.0.2", "10.0.0.1"},
								Vrf:       0,
								Underlay:  true,
							},
							{
								NetworkID: "3",
								IPs:       []string{"212.89.42.2", "212.89.42.1"},
								Vrf:       104009,
							},
							{
								NetworkID: "4",
								IPs:       []string{"2001::"},
								Vrf:       104010,
							},
						},
					},
				},
			},
			want: v1.NewBGPFilter([]string{}, []string{"10.1.0.0/22", "10.2.0.0/22", "100.127.1.1/32", "10.240.0.0/12", "2001::1/128", "212.89.42.1/32", "212.89.42.2/32"}),
		},
		{
			name: "allow only allocated ips",
			args: args{
				ipsMap: metal.IPsMap{"project": metal.IPs{
					metal.IP{
						IPAddress: "212.89.42.1",
					},
				}},
				machine: metal.Machine{
					Allocation: &metal.MachineAllocation{
						Project: "project",
						MachineNetworks: []*metal.MachineNetwork{
							{
								NetworkID: "5",
								IPs:       []string{"212.89.42.2", "212.89.42.1"},
								Vrf:       104009,
							},
						},
					},
				},
			},
			want: v1.NewBGPFilter([]string{}, []string{"212.89.42.1/32"}),
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			mock.On(r.DB("mockdb").Table("network").Get(r.MockAnything())).Return(testdata.Partition1PrivateSuperNetwork, nil)

			r := switchResource{webResource: webResource{ds: ds, log: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))}}

			got, _ := r.makeBGPFilterMachine(tt.args.machine, tt.args.nws, tt.args.ipsMap)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("makeBGPFilterMachine() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMakeSwitchNics(t *testing.T) {
	type args struct {
		s        *metal.Switch
		ips      metal.IPsMap
		nws      metal.NetworkMap
		machines metal.Machines
	}
	tests := []struct {
		name string
		args args
		want v1.SwitchNics
	}{
		{
			name: "machine and firewall bgp filter",
			args: args{
				s: &metal.Switch{
					MachineConnections: metal.ConnectionMap{
						"m1": metal.Connections{
							metal.Connection{
								MachineID: "m1",
								Nic: metal.Nic{
									Name: "swp1",
								},
							},
						},
						"fw1": metal.Connections{
							metal.Connection{
								MachineID: "fw1",
								Nic: metal.Nic{
									Name: "swp2",
								},
							},
						},
					},
					Nics: metal.Nics{
						metal.Nic{
							Name: "swp1",
							Vrf:  "vrf1",
							State: &metal.NicState{
								Actual: metal.SwitchPortStatusUp,
							},
						},
						metal.Nic{
							Name: "swp2",
							Vrf:  "default",
						},
					},
				},
				ips: metal.IPsMap{
					"project": metal.IPs{
						metal.IP{
							IPAddress: "212.89.1.1",
						},
					},
				},
				machines: metal.Machines{
					metal.Machine{
						Base: metal.Base{ID: "m1"},
						Allocation: &metal.MachineAllocation{
							Project: "project",
						},
					},
					metal.Machine{
						Base: metal.Base{ID: "fw1"},
						Allocation: &metal.MachineAllocation{
							Project: "p",
							Role:    metal.RoleFirewall,
							MachineNetworks: []*metal.MachineNetwork{
								{Vrf: 1},
								{Vrf: 2},
							},
						},
					},
				},
			},
			want: v1.SwitchNics{
				v1.SwitchNic{
					Name: "swp1",
					Vrf:  "vrf1",
					BGPFilter: &v1.BGPFilter{
						CIDRs: []string{"212.89.1.1/32"},
						VNIs:  []string{},
					},
					Actual: v1.SwitchPortStatusUp,
				},
				v1.SwitchNic{
					Name: "swp2",
					Vrf:  "default",
					BGPFilter: &v1.BGPFilter{
						CIDRs: []string{},
						VNIs:  []string{"1", "2"},
					},
					Actual: v1.SwitchPortStatusUnknown,
				},
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			r := switchResource{}
			got, _ := r.makeSwitchNics(tt.args.s, tt.args.nws, tt.args.ips, tt.args.machines)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("makeSwitchNics() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_adoptFromTwin(t *testing.T) {
	type args struct {
		old       *metal.Switch
		twin      *metal.Switch
		newSwitch *metal.Switch
	}
	tests := []struct {
		name    string
		args    args
		want    *metal.Switch
		wantErr bool
	}{
		{
			name: "adopt machine connections and nic configuration from twin",
			args: args{
				old: &metal.Switch{
					Mode: metal.SwitchReplace,
				},
				twin: &metal.Switch{
					OS: &metal.SwitchOS{Vendor: metal.SwitchOSVendorCumulus},
					Nics: metal.Nics{
						metal.Nic{
							Name:       "swp1s0",
							MacAddress: "aa:aa:aa:aa:aa:a1",
							Vrf:        "1",
						},
						metal.Nic{
							Name:       "swp1s1",
							MacAddress: "aa:aa:aa:aa:aa:a2",
						},
						metal.Nic{
							Name:       "swp1s2",
							MacAddress: "aa:aa:aa:aa:aa:a3",
						},
					},
					MachineConnections: metal.ConnectionMap{
						"m1": metal.Connections{
							metal.Connection{
								Nic: metal.Nic{
									Name:       "swp1s0",
									MacAddress: "aa:aa:aa:aa:aa:a1",
								},
							},
						},
						"fw1": metal.Connections{
							metal.Connection{
								Nic: metal.Nic{
									Name:       "swp1s1",
									MacAddress: "aa:aa:aa:aa:aa:a2",
								},
							},
						},
					},
				},
				newSwitch: &metal.Switch{
					OS: &metal.SwitchOS{Vendor: metal.SwitchOSVendorCumulus},
					Nics: metal.Nics{
						metal.Nic{
							Name:       "swp1s0",
							MacAddress: "bb:bb:bb:bb:bb:b1",
						},
						metal.Nic{
							Name:       "swp1s1",
							MacAddress: "bb:bb:bb:bb:bb:b2",
						},
						metal.Nic{
							Name:       "swp1s2",
							MacAddress: "bb:bb:bb:bb:bb:b3",
						},
						metal.Nic{
							Name:       "swp1s3",
							MacAddress: "bb:bb:bb:bb:bb:b4",
						},
					},
				},
			},
			want: &metal.Switch{
				Mode: metal.SwitchOperational,
				OS:   &metal.SwitchOS{Vendor: metal.SwitchOSVendorCumulus},
				Nics: metal.Nics{
					metal.Nic{
						Name:       "swp1s0",
						MacAddress: "bb:bb:bb:bb:bb:b1",
						Vrf:        "1",
					},
					metal.Nic{
						Name:       "swp1s1",
						MacAddress: "bb:bb:bb:bb:bb:b2",
					},
					metal.Nic{
						Name:       "swp1s2",
						MacAddress: "bb:bb:bb:bb:bb:b3",
					},
					metal.Nic{
						Name:       "swp1s3",
						MacAddress: "bb:bb:bb:bb:bb:b4",
					},
				},
				MachineConnections: metal.ConnectionMap{
					"m1": metal.Connections{
						metal.Connection{
							Nic: metal.Nic{
								Name:       "swp1s0",
								MacAddress: "bb:bb:bb:bb:bb:b1",
							},
						},
					},
					"fw1": metal.Connections{
						metal.Connection{
							Nic: metal.Nic{
								Name:       "swp1s1",
								MacAddress: "bb:bb:bb:bb:bb:b2",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "fail if partition differs",
			args: args{
				old: &metal.Switch{
					Mode:        metal.SwitchReplace,
					PartitionID: "1",
				},
				newSwitch: &metal.Switch{
					PartitionID: "2",
				},
			},
			wantErr: true,
		},
		{
			name: "fail if rack differs",
			args: args{
				old: &metal.Switch{
					Mode:        metal.SwitchReplace,
					PartitionID: "1",
					RackID:      "1",
				},
				newSwitch: &metal.Switch{
					PartitionID: "1",
					RackID:      "2",
				},
			},
			wantErr: true,
		},
		{
			name: "fail if twin switch is also in replace mode",
			args: args{
				old: &metal.Switch{
					Mode:        metal.SwitchReplace,
					PartitionID: "1",
					RackID:      "1",
				},
				twin: &metal.Switch{
					Mode:        metal.SwitchReplace,
					PartitionID: "1",
					RackID:      "1",
				},
				newSwitch: &metal.Switch{
					PartitionID: "1",
					RackID:      "1",
				},
			},
			wantErr: true,
		},
		{
			name: "new switch is directly useable if twin has no machine connections",
			args: args{
				old: &metal.Switch{
					Mode:        metal.SwitchReplace,
					PartitionID: "1",
					RackID:      "1",
				},
				twin: &metal.Switch{
					OS:          &metal.SwitchOS{Vendor: metal.SwitchOSVendorCumulus},
					PartitionID: "1",
					RackID:      "1",
				},
				newSwitch: &metal.Switch{
					PartitionID: "1",
					RackID:      "1",
				},
			},
			want: &metal.Switch{
				PartitionID: "1",
				RackID:      "1",
				Mode:        metal.SwitchOperational,
			},
			wantErr: false,
		},
		{
			name: "adopt machine connections and nic configuration from twin with different switch os",
			args: args{
				old: &metal.Switch{
					OS: &metal.SwitchOS{
						Vendor: metal.SwitchOSVendorCumulus,
					},
					Mode: metal.SwitchReplace,
				},
				twin: &metal.Switch{
					OS: &metal.SwitchOS{
						Vendor: metal.SwitchOSVendorCumulus,
					},
					Nics: metal.Nics{
						metal.Nic{
							Name:       "swp1s0",
							MacAddress: "aa:aa:aa:aa:aa:a1",
							Vrf:        "1",
						},
						metal.Nic{
							Name:       "swp1s1",
							MacAddress: "aa:aa:aa:aa:aa:a2",
						},
						metal.Nic{
							Name:       "swp1s2",
							MacAddress: "aa:aa:aa:aa:aa:a3",
						},
					},
					MachineConnections: metal.ConnectionMap{
						"m1": metal.Connections{
							metal.Connection{
								Nic: metal.Nic{
									Name:       "swp1s0",
									MacAddress: "aa:aa:aa:aa:aa:a1",
								},
							},
						},
						"fw1": metal.Connections{
							metal.Connection{
								Nic: metal.Nic{
									Name:       "swp1s1",
									MacAddress: "aa:aa:aa:aa:aa:a2",
								},
							},
						},
					},
				},
				newSwitch: &metal.Switch{
					OS: &metal.SwitchOS{
						Vendor: metal.SwitchOSVendorSonic,
					},
					Nics: metal.Nics{
						metal.Nic{
							Name:       "Ethernet0",
							MacAddress: "bb:bb:bb:bb:bb:b1",
						},
						metal.Nic{
							Name:       "Ethernet1",
							MacAddress: "bb:bb:bb:bb:bb:b2",
						},
						metal.Nic{
							Name:       "Ethernet2",
							MacAddress: "bb:bb:bb:bb:bb:b3",
						},
						metal.Nic{
							Name:       "Ethernet3",
							MacAddress: "bb:bb:bb:bb:bb:b4",
						},
					},
				},
			},
			want: &metal.Switch{
				Mode: metal.SwitchOperational,
				OS: &metal.SwitchOS{
					Vendor: metal.SwitchOSVendorSonic,
				},
				Nics: metal.Nics{
					metal.Nic{
						Name:       "Ethernet0",
						MacAddress: "bb:bb:bb:bb:bb:b1",
						Vrf:        "1",
					},
					metal.Nic{
						Name:       "Ethernet1",
						MacAddress: "bb:bb:bb:bb:bb:b2",
					},
					metal.Nic{
						Name:       "Ethernet2",
						MacAddress: "bb:bb:bb:bb:bb:b3",
					},
					metal.Nic{
						Name:       "Ethernet3",
						MacAddress: "bb:bb:bb:bb:bb:b4",
					},
				},
				MachineConnections: metal.ConnectionMap{
					"m1": metal.Connections{
						metal.Connection{
							Nic: metal.Nic{
								Name:       "Ethernet0",
								MacAddress: "bb:bb:bb:bb:bb:b1",
							},
						},
					},
					"fw1": metal.Connections{
						metal.Connection{
							Nic: metal.Nic{
								Name:       "Ethernet1",
								MacAddress: "bb:bb:bb:bb:bb:b2",
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got, err := adoptFromTwin(tt.args.old, tt.args.twin, tt.args.newSwitch)
			if (err != nil) != tt.wantErr {
				t.Errorf("adoptFromTwin() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(got, tt.want) {
				t.Errorf("adoptFromTwin() = %v", cmp.Diff(got, tt.want))
			}
		})
	}
}

func Test_adoptNics(t *testing.T) {
	type args struct {
		twin      *metal.Switch
		newSwitch *metal.Switch
	}
	tests := []struct {
		name    string
		args    args
		want    metal.Nics
		wantErr bool
	}{
		{
			name: "adopt vrf configuration, leaf underlay ports untouched, newSwitch might have additional ports",
			args: args{
				twin: &metal.Switch{
					OS: &metal.SwitchOS{Vendor: metal.SwitchOSVendorCumulus},
					Nics: metal.Nics{
						metal.Nic{
							Name:       "swp1s0",
							MacAddress: "aa:aa:aa:aa:aa:a1",
							Vrf:        "vrf1",
						},
						metal.Nic{
							Name:       "swp1s1",
							MacAddress: "aa:aa:aa:aa:aa:a2",
							Vrf:        "",
						},
					},
				},
				newSwitch: &metal.Switch{
					OS: &metal.SwitchOS{Vendor: metal.SwitchOSVendorCumulus},
					Nics: metal.Nics{
						metal.Nic{
							Name:       "swp1s0",
							MacAddress: "bb:bb:bb:bb:bb:b1",
						},
						metal.Nic{
							Name:       "swp1s1",
							MacAddress: "bb:bb:bb:bb:bb:b2",
						},
						metal.Nic{
							Name:       "swp99",
							MacAddress: "bb:bb:bb:bb:bb:b3",
						},
					},
				},
			},
			want: metal.Nics{
				metal.Nic{
					Name:       "swp1s0",
					MacAddress: "bb:bb:bb:bb:bb:b1",
					Vrf:        "vrf1",
				},
				metal.Nic{
					Name:       "swp1s1",
					MacAddress: "bb:bb:bb:bb:bb:b2",
					Vrf:        "",
				},
				metal.Nic{
					Name:       "swp99",
					MacAddress: "bb:bb:bb:bb:bb:b3",
				},
			},
			wantErr: false,
		},
		{
			name: "new switch misses nic",
			args: args{
				twin: &metal.Switch{
					OS: &metal.SwitchOS{Vendor: metal.SwitchOSVendorCumulus},
					Nics: metal.Nics{
						metal.Nic{
							Name:       "swp1s0",
							MacAddress: "aa:aa:aa:aa:aa:a1",
							Vrf:        "vrf1",
						},
					},
				},
				newSwitch: &metal.Switch{
					OS: &metal.SwitchOS{Vendor: metal.SwitchOSVendorCumulus},
					Nics: metal.Nics{
						metal.Nic{
							Name:       "swp1s1",
							MacAddress: "bb:bb:bb:bb:bb:b2",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "switch os from cumulus to sonic",
			args: args{
				twin: &metal.Switch{
					OS: &metal.SwitchOS{
						Vendor: metal.SwitchOSVendorCumulus,
					},
					Nics: metal.Nics{
						metal.Nic{
							Name:       "swp1s0",
							MacAddress: "aa:aa:aa:aa:aa:a1",
							Vrf:        "vrf1",
						},
						metal.Nic{
							Name:       "swp1s1",
							MacAddress: "aa:aa:aa:aa:aa:a2",
							Vrf:        "",
						},
						metal.Nic{
							Name:       "swp99",
							MacAddress: "aa:aa:aa:aa:aa:a3",
						},
					},
				},
				newSwitch: &metal.Switch{
					OS: &metal.SwitchOS{
						Vendor: metal.SwitchOSVendorSonic,
					},
					Nics: metal.Nics{
						metal.Nic{
							Name:       "Ethernet0",
							MacAddress: "bb:bb:bb:bb:bb:b2",
						},
						metal.Nic{
							Name:       "Ethernet1",
							MacAddress: "bb:bb:bb:bb:bb:b3",
						},
						metal.Nic{
							Name:       "Ethernet392",
							MacAddress: "bb:bb:bb:bb:bb:b4",
						},
					},
				},
			},
			want: metal.Nics{
				metal.Nic{
					Name:       "Ethernet0",
					MacAddress: "bb:bb:bb:bb:bb:b2",
					Vrf:        "vrf1",
				},
				metal.Nic{
					Name:       "Ethernet1",
					MacAddress: "bb:bb:bb:bb:bb:b3",
					Vrf:        "",
				},
				metal.Nic{
					Name:       "Ethernet392",
					MacAddress: "bb:bb:bb:bb:bb:b4",
				},
			},
			wantErr: false,
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got, err := adoptNics(tt.args.twin, tt.args.newSwitch)
			if (err != nil) != tt.wantErr {
				t.Errorf("adoptNics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("diff %v", diff)
			}
		})
	}
}

func Test_adoptMachineConnections(t *testing.T) {
	type args struct {
		twin      *metal.Switch
		newSwitch *metal.Switch
	}
	tests := []struct {
		name    string
		args    args
		want    metal.ConnectionMap
		wantErr bool
	}{
		{
			name: "adopt machine connections from twin",
			args: args{
				twin: &metal.Switch{
					OS: &metal.SwitchOS{Vendor: metal.SwitchOSVendorCumulus},
					MachineConnections: metal.ConnectionMap{
						"m1": metal.Connections{
							metal.Connection{
								Nic: metal.Nic{
									Name:       "swp1s0",
									MacAddress: "aa:aa:aa:aa:aa:a1",
								},
							},
						},
						"m2": metal.Connections{
							metal.Connection{
								Nic: metal.Nic{
									Name:       "swp1s1",
									MacAddress: "aa:aa:aa:aa:aa:a2",
								},
							},
						},
					},
				},
				newSwitch: &metal.Switch{
					OS: &metal.SwitchOS{Vendor: metal.SwitchOSVendorCumulus},
					Nics: metal.Nics{
						metal.Nic{
							Name:       "swp1s0",
							MacAddress: "bb:bb:bb:bb:bb:b1",
						},
						metal.Nic{
							Name:       "swp1s1",
							MacAddress: "bb:bb:bb:bb:bb:b2",
						},
					},
				},
			},
			want: metal.ConnectionMap{
				"m1": metal.Connections{
					metal.Connection{
						Nic: metal.Nic{
							Name:       "swp1s0",
							MacAddress: "bb:bb:bb:bb:bb:b1",
						},
					},
				},
				"m2": metal.Connections{
					metal.Connection{
						Nic: metal.Nic{
							Name:       "swp1s1",
							MacAddress: "bb:bb:bb:bb:bb:b2",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "new switch misses nic for existing machine connection at twin",
			args: args{
				twin: &metal.Switch{
					OS: &metal.SwitchOS{Vendor: metal.SwitchOSVendorCumulus},
					MachineConnections: metal.ConnectionMap{
						"m1": metal.Connections{
							metal.Connection{
								Nic: metal.Nic{
									Name:       "swp1s0",
									MacAddress: "aa:aa:aa:aa:aa:a1",
								},
							},
						},
					},
				},
				newSwitch: &metal.Switch{
					OS: &metal.SwitchOS{Vendor: metal.SwitchOSVendorCumulus},
					Nics: metal.Nics{
						metal.Nic{
							Name:       "swp1s1",
							MacAddress: "bb:bb:bb:bb:bb:b2",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "adopt from twin with different switch os",
			args: args{
				twin: &metal.Switch{
					OS: &metal.SwitchOS{Vendor: metal.SwitchOSVendorCumulus},
					MachineConnections: metal.ConnectionMap{
						"m1": metal.Connections{
							metal.Connection{
								Nic: metal.Nic{
									Name:       "swp1s0",
									MacAddress: "aa:aa:aa:aa:aa:a1",
								},
							},
						},
						"m2": metal.Connections{
							metal.Connection{
								Nic: metal.Nic{
									Name:       "swp1s1",
									MacAddress: "aa:aa:aa:aa:aa:a2",
								},
							},
						},
					},
				},
				newSwitch: &metal.Switch{
					OS: &metal.SwitchOS{Vendor: metal.SwitchOSVendorSonic},
					Nics: metal.Nics{
						metal.Nic{
							Name:       "Ethernet0",
							MacAddress: "bb:bb:bb:bb:bb:b1",
						},
						metal.Nic{
							Name:       "Ethernet1",
							MacAddress: "bb:bb:bb:bb:bb:b2",
						},
					},
				},
			},
			want: metal.ConnectionMap{
				"m1": metal.Connections{
					metal.Connection{
						Nic: metal.Nic{
							Name:       "Ethernet0",
							MacAddress: "bb:bb:bb:bb:bb:b1",
						},
					},
				},
				"m2": metal.Connections{
					metal.Connection{
						Nic: metal.Nic{
							Name:       "Ethernet1",
							MacAddress: "bb:bb:bb:bb:bb:b2",
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got, err := adoptMachineConnections(tt.args.twin, tt.args.newSwitch)
			if (err != nil) != tt.wantErr {
				t.Errorf("adoptMachineConnections() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("adoptMachineConnections() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_updateSwitchNics(t *testing.T) {
	type args struct {
		oldNics            map[string]*metal.Nic
		newNics            map[string]*metal.Nic
		currentConnections metal.ConnectionMap
	}
	tests := []struct {
		name    string
		args    args
		want    metal.Nics
		wantErr bool
	}{
		{
			name: "idempotence",
			args: args{
				oldNics: map[string]*metal.Nic{
					"11:11:11:11:11:11": {Name: "swp1", MacAddress: "11:11:11:11:11:11"},
				},
				newNics: map[string]*metal.Nic{
					"11:11:11:11:11:11": {Name: "swp1", MacAddress: "11:11:11:11:11:11"},
				},
				currentConnections: metal.ConnectionMap{},
			},
			want: metal.Nics{
				metal.Nic{Name: "swp1", MacAddress: "11:11:11:11:11:11"},
			},
			wantErr: false,
		},
		{
			name: "adding a nic",
			args: args{
				oldNics: map[string]*metal.Nic{
					"11:11:11:11:11:11": {Name: "swp1", MacAddress: "11:11:11:11:11:11"},
				},
				newNics: map[string]*metal.Nic{
					"11:11:11:11:11:11": {Name: "swp1", MacAddress: "11:11:11:11:11:11"},
					"11:11:11:11:11:12": {Name: "swp2", MacAddress: "11:11:11:11:11:12"},
				},
				currentConnections: metal.ConnectionMap{},
			},
			want: metal.Nics{
				metal.Nic{Name: "swp1", MacAddress: "11:11:11:11:11:11"},
				metal.Nic{Name: "swp2", MacAddress: "11:11:11:11:11:12"},
			},
			wantErr: false,
		},
		{
			name: "removing a nic",
			args: args{
				oldNics: map[string]*metal.Nic{
					"11:11:11:11:11:11": {Name: "swp1", MacAddress: "11:11:11:11:11:11"},
				},
				newNics:            map[string]*metal.Nic{},
				currentConnections: metal.ConnectionMap{},
			},
			want:    metal.Nics{},
			wantErr: false,
		},
		{
			name: "removing a nic 2",
			args: args{
				oldNics: map[string]*metal.Nic{
					"11:11:11:11:11:11": {Name: "swp1", MacAddress: "11:11:11:11:11:11"},
					"11:11:11:11:11:12": {Name: "swp2", MacAddress: "11:11:11:11:11:12"},
				},
				newNics: map[string]*metal.Nic{
					"11:11:11:11:11:11": {Name: "swp1", MacAddress: "11:11:11:11:11:11"},
				},
				currentConnections: metal.ConnectionMap{},
			},
			want: metal.Nics{
				metal.Nic{Name: "swp1", MacAddress: "11:11:11:11:11:11"},
			},
			wantErr: false,
		},
		{
			name: "removing a used nic",
			args: args{
				oldNics: map[string]*metal.Nic{
					"11:11:11:11:11:11": {Name: "swp1", MacAddress: "11:11:11:11:11:11"},
					"11:11:11:11:11:12": {Name: "swp2", MacAddress: "11:11:11:11:11:12"},
				},
				newNics: map[string]*metal.Nic{
					"11:11:11:11:11:11": {Name: "swp1", MacAddress: "11:11:11:11:11:11"},
				},
				currentConnections: metal.ConnectionMap{
					"machine-uuid-1": metal.Connections{metal.Connection{MachineID: "machine-uuid-1", Nic: metal.Nic{Name: "swp2", MacAddress: "11:11:11:11:11:12"}}},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "updating a nic",
			args: args{
				oldNics: map[string]*metal.Nic{
					"11:11:11:11:11:11": {Name: "swp1", MacAddress: "11:11:11:11:11:11"},
				},
				newNics: map[string]*metal.Nic{
					"11:11:11:11:11:11": {Name: "swp2", MacAddress: "11:11:11:11:11:11"},
				},
				currentConnections: metal.ConnectionMap{},
			},
			want: metal.Nics{
				metal.Nic{Name: "swp2", MacAddress: "11:11:11:11:11:11"},
			},
			wantErr: false,
		},
		{
			name: "updating a nic, vrf should not be altered",
			args: args{
				oldNics: map[string]*metal.Nic{
					"11:11:11:11:11:11": {Name: "swp1", Vrf: "vrf1", MacAddress: "11:11:11:11:11:11"},
				},
				newNics: map[string]*metal.Nic{
					"11:11:11:11:11:11": {Name: "swp2", Vrf: "vrf2", MacAddress: "11:11:11:11:11:11"},
				},
				currentConnections: metal.ConnectionMap{},
			},
			want: metal.Nics{
				metal.Nic{Name: "swp2", Vrf: "vrf1", MacAddress: "11:11:11:11:11:11"},
			},
			wantErr: false,
		},
		{
			name: "updating a nic name, which already has a connection",
			args: args{
				oldNics: map[string]*metal.Nic{
					"11:11:11:11:11:11": {Name: "swp1", MacAddress: "11:11:11:11:11:11"},
				},
				newNics: map[string]*metal.Nic{
					"11:11:11:11:11:11": {Name: "swp2", MacAddress: "11:11:11:11:11:11"},
				},
				currentConnections: metal.ConnectionMap{
					"machine-uuid-1": metal.Connections{metal.Connection{MachineID: "machine-uuid-1", Nic: metal.Nic{Name: "swp1", MacAddress: "11:11:11:11:11:11"}}},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got, err := updateSwitchNics(tt.args.oldNics, tt.args.newNics, tt.args.currentConnections)
			if (err != nil) != tt.wantErr {
				t.Errorf("updateSwitchNics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got.ByIdentifier(), tt.want.ByIdentifier()) {
				t.Errorf("updateSwitchNics() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateSwitch(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	switchservice := NewSwitch(log, ds)
	container := restful.NewContainer().Add(switchservice)

	desc := "test"
	updateRequest := v1.SwitchUpdateRequest{
		Common: v1.Common{
			Describable: v1.Describable{
				Description: &desc,
			},
			Identifiable: v1.Identifiable{
				ID: testdata.Switch1.ID,
			},
		},
		SwitchBase: v1.SwitchBase{
			Mode: string(metal.SwitchReplace),
		},
	}
	js, err := json.Marshal(updateRequest)
	require.NoError(t, err)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/switch", body)
	container = injectAdmin(log, container, req)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.SwitchResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.Switch1.ID, result.ID)
	require.Equal(t, testdata.Switch1.Name, *result.Name)
	require.Equal(t, desc, *result.Description)
	require.Equal(t, string(metal.SwitchReplace), result.Mode)
}

func TestNotifySwitch(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	switchservice := NewSwitch(log, ds)
	container := restful.NewContainer().Add(switchservice)

	d := time.Second * 10
	notifyRequest := v1.SwitchNotifyRequest{
		Duration: d,
	}
	js, err := json.Marshal(notifyRequest)
	require.NoError(t, err)
	body := bytes.NewBuffer(js)
	id := testdata.Switch1.ID
	req := httptest.NewRequest("POST", "/v1/switch/"+id+"/notify", body)
	container = injectEditor(log, container, req)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.SwitchNotifyResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, id, result.ID)
	require.NotNil(t, result.LastSync)
	require.Equal(t, d, result.LastSync.Duration)
	require.Nil(t, result.LastSyncError)
}

func TestNotifyErrorSwitch(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	switchservice := NewSwitch(log, ds)
	container := restful.NewContainer().Add(switchservice)

	d := time.Second * 10
	e := "failed to apply config"
	notifyRequest := v1.SwitchNotifyRequest{
		Duration: d,
		Error:    &e,
	}
	js, err := json.Marshal(notifyRequest)
	require.NoError(t, err)
	body := bytes.NewBuffer(js)
	id := testdata.Switch1.ID
	req := httptest.NewRequest("POST", "/v1/switch/"+id+"/notify", body)
	container = injectEditor(log, container, req)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.SwitchNotifyResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, id, result.ID)
	require.Equal(t, d, result.LastSyncError.Duration)
	require.Equal(t, e, *result.LastSyncError.Error)
}

func TestToggleSwitchWrongNic(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	switchservice := NewSwitch(log, ds)
	container := restful.NewContainer().Add(switchservice)

	updateRequest := v1.SwitchPortToggleRequest{
		NicName: "wrongname",
		Status:  v1.SwitchPortStatusDown,
	}
	js, err := json.Marshal(updateRequest)
	require.NoError(t, err)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/switch/"+testdata.Switch1.ID+"/port", body)
	container = injectAdmin(log, container, req)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode, w.Body.String())
	var result httperrors.HTTPErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, "the nic \"wrongname\" does not exist in this switch", result.Message)
}

func TestToggleSwitchWrongState(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	switchservice := NewSwitch(log, ds)
	container := restful.NewContainer().Add(switchservice)

	states := []v1.SwitchPortStatus{
		v1.SwitchPortStatusUnknown,
		v1.SwitchPortStatus("illegal"),
	}

	for _, s := range states {

		updateRequest := v1.SwitchPortToggleRequest{
			NicName: testdata.Switch1.Nics[0].Name,
			Status:  s,
		}
		js, err := json.Marshal(updateRequest)
		require.NoError(t, err)
		body := bytes.NewBuffer(js)
		req := httptest.NewRequest("POST", "/v1/switch/"+testdata.Switch1.ID+"/port", body)
		container = injectAdmin(log, container, req)
		req.Header.Add("Content-Type", "application/json")
		w := httptest.NewRecorder()
		container.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode, w.Body.String())
		var result httperrors.HTTPErrorResponse
		err = json.NewDecoder(resp.Body).Decode(&result)

		require.NoError(t, err)
		require.Equal(t, result.Message, fmt.Sprintf("the status %q must be concrete", s))
	}
}

func TestToggleSwitch(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	switchservice := NewSwitch(log, ds)
	container := restful.NewContainer().Add(switchservice)

	updateRequest := v1.SwitchPortToggleRequest{
		NicName: testdata.Switch1.Nics[0].Name,
		Status:  v1.SwitchPortStatusDown,
	}

	js, err := json.Marshal(updateRequest)
	require.NoError(t, err)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/switch/"+testdata.Switch1.ID+"/port", body)
	container = injectAdmin(log, container, req)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.SwitchResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.Switch1.ID, result.ID)
	require.Equal(t, testdata.Switch1.Name, *result.Name)
	require.Equal(t, v1.SwitchPortStatusDown, result.Nics[0].Actual)
	require.Equal(t, v1.SwitchPortStatusUnknown, result.Connections[0].Nic.Actual)
}

func TestToggleSwitchNicWithoutMachine(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	switchservice := NewSwitch(log, ds)
	container := restful.NewContainer().Add(switchservice)

	updateRequest := v1.SwitchPortToggleRequest{
		NicName: testdata.Switch1.Nics[1].Name,
		Status:  v1.SwitchPortStatusDown,
	}

	js, err := json.Marshal(updateRequest)
	require.NoError(t, err)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/switch/"+testdata.Switch1.ID+"/port", body)
	container = injectAdmin(log, container, req)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, w.Body.String())
	var result httperrors.HTTPErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, result.Message, fmt.Sprintf("switch %q does not have a connected machine at port %q", testdata.Switch1.ID, testdata.Switch1.Nics[1].Name))
}

func Test_adjustMachineNics(t *testing.T) {
	tests := []struct {
		name        string
		nics        metal.Nics
		connections metal.Connections
		nicMap      metal.NicMap
		want        metal.Nics
		wantErr     bool
	}{
		{
			name: "nothing to adjust",
			nics: []metal.Nic{
				{
					Name:       "eth0",
					MacAddress: "11:11:11:11:11:11",
					Neighbors: []metal.Nic{
						{
							Name:       "swp1",
							MacAddress: "aa:aa:aa:aa:aa:aa",
						},
					},
				},
				{
					Name:       "eth1",
					MacAddress: "11:11:11:11:11:22",
					Neighbors: []metal.Nic{
						{
							Name:       "swp1",
							MacAddress: "aa:aa:aa:aa:aa:bb",
						},
					},
				},
			},
			connections: []metal.Connection{
				{
					Nic: metal.Nic{
						Name:       "swp1",
						MacAddress: "cc:cc:cc:cc:cc:cc",
					},
				},
			},
			want: []metal.Nic{
				{
					Name:       "eth0",
					MacAddress: "11:11:11:11:11:11",
					Neighbors: []metal.Nic{
						{
							Name:       "swp1",
							MacAddress: "aa:aa:aa:aa:aa:aa",
						},
					},
				},
				{
					Name:       "eth1",
					MacAddress: "11:11:11:11:11:22",
					Neighbors: []metal.Nic{
						{
							Name:       "swp1",
							MacAddress: "aa:aa:aa:aa:aa:bb",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "unrealistic error case",
			nics: []metal.Nic{
				{
					Name:       "eth0",
					MacAddress: "11:11:11:11:11:11",
					Neighbors: []metal.Nic{
						{
							Name:       "swp2",
							MacAddress: "aa:aa:aa:aa:aa:aa",
						},
					},
				},
				{
					Name:       "eth1",
					MacAddress: "11:11:11:11:11:22",
					Neighbors: []metal.Nic{
						{
							Name:       "swp2",
							MacAddress: "aa:aa:aa:aa:aa:bb",
						},
					},
				},
			},
			connections: []metal.Connection{
				{
					Nic: metal.Nic{
						Name:       "swp2",
						MacAddress: "aa:aa:aa:aa:aa:aa",
					},
				},
			},
			nicMap: map[string]*metal.Nic{
				"swp1": {
					Name:       "Ethernet0",
					MacAddress: "dd:dd:dd:dd:dd:dd",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "adjust nics from cumulus to sonic",
			nics: []metal.Nic{
				{
					Name:       "eth0",
					MacAddress: "11:11:11:11:11:11",
					Neighbors: []metal.Nic{
						{
							Name:       "swp1",
							MacAddress: "aa:aa:aa:aa:aa:aa",
						},
					},
				},
				{
					Name:       "eth1",
					MacAddress: "11:11:11:11:11:22",
					Neighbors: []metal.Nic{
						{
							Name:       "swp1",
							MacAddress: "aa:aa:aa:aa:aa:bb",
						},
					},
				},
			},
			connections: []metal.Connection{
				{
					Nic: metal.Nic{
						Name:       "swp1",
						MacAddress: "aa:aa:aa:aa:aa:aa",
					},
				},
			},
			nicMap: map[string]*metal.Nic{
				"swp1": {
					Name:       "Ethernet0",
					MacAddress: "dd:dd:dd:dd:dd:dd",
				},
			},
			want: []metal.Nic{
				{
					Name:       "eth0",
					MacAddress: "11:11:11:11:11:11",
					Neighbors: []metal.Nic{
						{
							Name:       "Ethernet0",
							MacAddress: "dd:dd:dd:dd:dd:dd",
						},
					},
				},
				{
					Name:       "eth1",
					MacAddress: "11:11:11:11:11:22",
					Neighbors: []metal.Nic{
						{
							Name:       "swp1",
							MacAddress: "aa:aa:aa:aa:aa:bb",
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := adjustMachineNics(tt.nics, tt.connections, tt.nicMap)
			if (err != nil) != tt.wantErr {
				t.Errorf("adjustMachineNics() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("adjustMachineNics() diff = %v", diff)
			}
		})
	}
}

func Test_SwitchDelete(t *testing.T) {
	tests := []struct {
		name       string
		mockFn     func(mock *r.Mock)
		want       *v1.SwitchResponse
		wantErr    error
		wantStatus int
		force      bool
	}{
		{
			name: "delete switch",
			mockFn: func(mock *r.Mock) {
				mock.On(r.DB("mockdb").Table("switch").Get("switch-1")).Return(&metal.Switch{
					Base: metal.Base{
						ID: "switch-1",
					},
				}, nil)
				mock.On(r.DB("mockdb").Table("switch").Get("switch-1").Delete()).Return(testdata.EmptyResult, nil)
				mock.On(r.DB("mockdb").Table("switchstatus").Get("switch-1")).Return(nil, nil)
				mock.On(r.DB("mockdb").Table("ip")).Return(nil, nil)
				mock.On(r.DB("mockdb").Table("network")).Return(nil, nil)
			},
			want: &v1.SwitchResponse{
				Common: v1.Common{
					Identifiable: v1.Identifiable{
						ID: "switch-1",
					},
					Describable: v1.Describable{
						Name:        pointer.Pointer(""),
						Description: pointer.Pointer(""),
					},
				},
				Nics:        v1.SwitchNics{},
				Connections: []v1.SwitchConnection{},
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "delete switch does not work due to machine connections",
			mockFn: func(mock *r.Mock) {
				mock.On(r.DB("mockdb").Table("switch").Get("switch-1")).Return(&metal.Switch{
					Base: metal.Base{
						ID: "switch-1",
					},
					MachineConnections: metal.ConnectionMap{
						"port-a": metal.Connections{},
					},
				}, nil)
				mock.On(r.DB("mockdb").Table("switch").Get("switch-1").Delete()).Return(testdata.EmptyResult, nil)
			},
			wantErr: &httperrors.HTTPErrorResponse{
				StatusCode: http.StatusBadRequest,
				Message:    "cannot delete switch switch-1 while it still has machines connected to it",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "delete switch with force",
			mockFn: func(mock *r.Mock) {
				mock.On(r.DB("mockdb").Table("switch").Get("switch-1")).Return(&metal.Switch{
					Base: metal.Base{
						ID: "switch-1",
					},
					MachineConnections: metal.ConnectionMap{
						"port-a": metal.Connections{},
					},
				}, nil)
				mock.On(r.DB("mockdb").Table("switch").Get("switch-1").Delete()).Return(testdata.EmptyResult, nil)
				mock.On(r.DB("mockdb").Table("switchstatus").Get("switch-1")).Return(nil, nil)
				mock.On(r.DB("mockdb").Table("ip")).Return(nil, nil)
				mock.On(r.DB("mockdb").Table("network")).Return(nil, nil)
			},
			force: true,
			want: &v1.SwitchResponse{
				Common: v1.Common{
					Identifiable: v1.Identifiable{
						ID: "switch-1",
					},
					Describable: v1.Describable{
						Name:        pointer.Pointer(""),
						Description: pointer.Pointer(""),
					},
				},
				Nics:        v1.SwitchNics{},
				Connections: []v1.SwitchConnection{},
			},
			wantStatus: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var (
				ds, mock = datastore.InitMockDB(t)
				ws       = NewSwitch(slog.Default(), ds)
			)

			if tt.mockFn != nil {
				tt.mockFn(mock)
			}

			if tt.wantErr != nil {
				code, got := genericWebRequest[*httperrors.HTTPErrorResponse](t, ws, testAdminUser, nil, "DELETE", "/v1/switch/switch-1")
				assert.Equal(t, tt.wantStatus, code)

				if diff := cmp.Diff(tt.wantErr, got); diff != "" {
					t.Errorf("diff (-want +got):\n%s", diff)
				}

				return
			}

			force := ""
			if tt.force {
				force = "?force=true"
			}

			code, got := genericWebRequest[*v1.SwitchResponse](t, ws, testAdminUser, nil, "DELETE", "/v1/switch/switch-1"+force)
			assert.Equal(t, tt.wantStatus, code)

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCompactCidrs(t *testing.T) {
	// sample cidrs from a production cluster passed in to be added to the route map
	cidrs := []string{
		"10.4.0.31/32",
		"10.6.0.25/32",
		"10.64.28.0/22",
		"10.67.36.138/32",
		"10.76.20.12/32",
		"10.76.20.14/32",
		"10.76.20.15/32",
		"10.76.20.16/32",
		"10.76.20.17/32",
		"10.76.20.2/32",
		"10.76.20.3/32",
		"10.76.20.4/32",
		"10.76.20.5/32",
		"10.76.20.6/32",
		"10.76.20.7/32",
		"10.76.20.8/32",
		"10.76.20.9/32",
		"10.78.248.134/32",
		"2001:db8::7/128",
		"2001:db8::8/128",
		"2001:db8::20/128",
		"2001:db8::db/128",
		"100.127.130.178/32",
		"100.127.130.179/32",
		"100.127.130.180/32",
		"100.127.130.181/32",
		"100.127.130.182/32",
		"100.127.130.183/32",
		"100.153.67.112/32",
		"100.153.67.113/32",
		"100.153.67.114/32",
		"100.153.67.115/32",
		"100.153.67.116/32",
		"100.34.85.136/32",
		"100.34.85.17/32",
		"100.34.89.209/32",
		"2001:db8::9/128",
		"2001:db8::10/128",
		"100.34.89.210/32",
		"100.90.30.12/32",
		"100.90.30.13/32",
		"100.90.30.14/32",
		"100.90.30.15/32",
		"100.90.30.16/32",
		"100.90.30.32/32",
		"100.90.30.4/32",
		"100.90.30.7/32",
	}

	compactedCidrs := []string{
		"10.4.0.31/32",
		"10.6.0.25/32",
		"10.64.28.0/22",
		"10.67.36.138/32",
		"10.76.20.2/31",
		"10.76.20.4/30",
		"10.76.20.8/31",
		"10.76.20.12/32",
		"10.76.20.14/31",
		"10.76.20.16/31",
		"10.78.248.134/32",
		"100.90.30.4/32",
		"100.90.30.7/32",
		"100.90.30.12/30",
		"100.90.30.16/32",
		"100.90.30.32/32",
		"100.127.130.178/31",
		"100.127.130.180/30",
		"100.153.67.112/30",
		"100.153.67.116/32",
		"100.34.85.17/32",
		"100.34.85.136/32",
		"100.34.89.209/32",
		"100.34.89.210/32",
		"2001:db8::7/128",
		"2001:db8::8/127",
		"2001:db8::10/128",
		"2001:db8::20/128",
		"2001:db8::db/128",
	}

	compacted, err := compactCidrs(cidrs)
	require.NoError(t, err)
	require.Less(t, len(compacted), len(cidrs))
	require.Len(t, compacted, 29)

	t.Logf("aggregated cidrs:%s old count:%d new count:%d", compacted, len(cidrs), len(compacted))

	require.ElementsMatch(t, compactedCidrs, compacted)
}
