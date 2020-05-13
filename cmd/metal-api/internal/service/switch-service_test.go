package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	restful "github.com/emicklei/go-restful"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
	"github.com/stretchr/testify/require"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

func TestRegisterSwitch(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	switchservice := NewSwitch(ds)
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
	js, _ := json.Marshal(createRequest)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/switch/register", body)
	req.Header.Add("Content-Type", "application/json")
	container = injectEditor(container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusCreated, resp.StatusCode, w.Body.String())
	var result v1.SwitchResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, "switch999", result.ID)
	require.Equal(t, "switch999", *result.Name)
	require.Equal(t, "1", result.RackID)
	require.Equal(t, "1", result.Partition.ID)
	require.Len(t, result.Connections, 0)
}

func TestRegisterExistingSwitch(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	switchservice := NewSwitch(ds)
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
	js, _ := json.Marshal(createRequest)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/switch/register", body)
	req.Header.Add("Content-Type", "application/json")
	container = injectEditor(container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.SwitchResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, testdata.Switch2.ID, result.ID)
	require.Equal(t, testdata.Switch2.Name, *result.Name)
	require.Equal(t, testdata.Switch2.RackID, result.RackID)
	require.Equal(t, testdata.Switch2.PartitionID, result.Partition.ID)
	require.Len(t, result.Connections, 0)
	// con := result.Connections[0]
	// require.Equal(t, testdata.Switch2.MachineConnections["1"][0].Nic.MacAddress, con.Nic.MacAddress)
}

func TestRegisterExistingSwitchErrorModifyingNics(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	switchservice := NewSwitch(ds)
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
	js, _ := json.Marshal(createRequest)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/switch/register", body)
	container = injectAdmin(container, req)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)
}

func TestNotifySwitch(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	switchservice := NewSwitch(ds)
	container := restful.NewContainer().Add(switchservice)

	d := time.Second * 10
	notifyRequest := v1.SwitchNotifyRequest{
		Duration: d,
	}
	js, _ := json.Marshal(notifyRequest)
	body := bytes.NewBuffer(js)
	id := testdata.Switch1.ID
	req := httptest.NewRequest("POST", "/v1/switch/"+id+"/notify", body)
	container = injectEditor(container, req)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.SwitchResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, id, result.ID)
	require.Equal(t, d, result.LastSync.Duration)
	require.Nil(t, result.LastSyncError)
}

func TestNotifyErrorSwitch(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	switchservice := NewSwitch(ds)
	container := restful.NewContainer().Add(switchservice)

	d := time.Second * 10
	e := "failed to apply config"
	notifyRequest := v1.SwitchNotifyRequest{
		Duration: d,
		Error:    &e,
	}
	js, _ := json.Marshal(notifyRequest)
	body := bytes.NewBuffer(js)
	id := testdata.Switch1.ID
	req := httptest.NewRequest("POST", "/v1/switch/"+id+"/notify", body)
	container = injectEditor(container, req)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.SwitchResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, id, result.ID)
	require.Equal(t, d, result.LastSyncError.Duration)
	require.Equal(t, e, *result.LastSyncError.Error)
}

func TestConnectMachineWithSwitches(t *testing.T) {
	tests := []struct {
		name    string
		machine *metal.Machine
		wantErr bool
	}{
		{
			name: "Test 1",
			machine: &metal.Machine{
				Base:        metal.Base{ID: "1"},
				PartitionID: "1",
			},
			wantErr: false,
		},
		{
			name: "Test 2",
			machine: &metal.Machine{
				Base:        metal.Base{ID: "1"},
				PartitionID: "1",
			}, wantErr: false,
		},
	}
	for _, tt := range tests {
		ds, mock := datastore.InitMockDB()
		mock.On(r.DB("mockdb").Table("switch")).Return(testdata.TestSwitches, nil)
		mock.On(r.DB("mockdb").Table("switch").Get(r.MockAnything()).Replace(r.MockAnything())).Return(testdata.EmptyResult, nil)

		t.Run(tt.name, func(t *testing.T) {
			if err := connectMachineWithSwitches(ds, tt.machine); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.connectMachineWithSwitches() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
		mock.AssertExpectations(t)
	}
}

func TestSetVrfAtSwitch(t *testing.T) {
	ds, mock := datastore.InitMockDB()
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
	mock.On(r.DB("mockdb").Table("switch")).Return(sws, nil)
	mock.On(r.DB("mockdb").Table("switch").Get(r.MockAnything()).Replace(r.MockAnything())).Return(testdata.EmptyResult, nil)

	vrf := "123"
	m := &metal.Machine{
		Base:        metal.Base{ID: "1"},
		PartitionID: "1",
	}
	switches, err := setVrfAtSwitches(ds, m, vrf)
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
							&metal.MachineNetwork{
								IPs: []string{},
								Vrf: 104010,
							},
							&metal.MachineNetwork{
								IPs:      []string{"10.0.0.2", "10.0.0.1"},
								Vrf:      0,
								Underlay: true,
							},
							&metal.MachineNetwork{
								IPs: []string{"212.89.42.1", "212.89.42.2"},
								Vrf: 104009,
							},
						},
					},
				},
			},
			want: v1.NewBGPFilter([]string{"104009", "104010"}, []string{"10.0.0.1/32", "10.0.0.2/32"}),
		},
		{
			name: "no underlay firewall networks",
			args: args{
				machine: metal.Machine{
					Allocation: &metal.MachineAllocation{
						MachineNetworks: []*metal.MachineNetwork{
							&metal.MachineNetwork{
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := makeBGPFilterFirewall(tt.args.machine)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("makeBGPFilterFirewall() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMakeBGPFilterMachine(t *testing.T) {
	type args struct {
		machine metal.Machine
		ipsMap  metal.IPsMap
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
				}},
				machine: metal.Machine{
					Allocation: &metal.MachineAllocation{
						Project: "project",
						MachineNetworks: []*metal.MachineNetwork{
							&metal.MachineNetwork{
								IPs:      []string{"10.1.0.1"},
								Prefixes: []string{"10.2.0.0/22", "10.1.0.0/22"},
								Vrf:      1234,
								Private:  true,
							},
							&metal.MachineNetwork{
								IPs:      []string{"10.0.0.2", "10.0.0.1"},
								Vrf:      0,
								Underlay: true,
							},
							&metal.MachineNetwork{
								IPs: []string{"212.89.42.2", "212.89.42.1"},
								Vrf: 104009,
							},
						},
					},
				},
			},
			want: v1.NewBGPFilter([]string{}, []string{"10.1.0.0/22", "10.2.0.0/22", "100.127.1.1/32", "212.89.42.1/32", "212.89.42.2/32"}),
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
							&metal.MachineNetwork{
								IPs: []string{"212.89.42.2", "212.89.42.1"},
								Vrf: 104009,
							},
						},
					},
				},
			},
			want: v1.NewBGPFilter([]string{}, []string{"212.89.42.1/32"}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := makeBGPFilterMachine(tt.args.machine, tt.args.ipsMap)
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
		images   metal.ImageMap
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
						},
						metal.Nic{
							Name: "swp2",
							Vrf:  "default",
						},
					},
				},
				ips: metal.IPsMap{"project": metal.IPs{
					metal.IP{
						IPAddress: "212.89.1.1",
					},
				},
				},
				images: metal.ImageMap{
					"fwimg": metal.Image{
						Base:     metal.Base{ID: "fwimg"},
						Features: map[metal.ImageFeatureType]bool{metal.ImageFeatureFirewall: true},
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
							ImageID: "fwimg",
							MachineNetworks: []*metal.MachineNetwork{
								&metal.MachineNetwork{Vrf: 1},
								&metal.MachineNetwork{Vrf: 2},
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
				},
				v1.SwitchNic{
					Name: "swp2",
					Vrf:  "default",
					BGPFilter: &v1.BGPFilter{
						CIDRs: []string{},
						VNIs:  []string{"1", "2"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := makeSwitchNics(tt.args.s, tt.args.ips, tt.args.images, tt.args.machines)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("makeSwitchNics() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_updateSwitchNics(t *testing.T) {
	type args struct {
		oldNics            metal.NicMap
		newNics            metal.NicMap
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
				oldNics: metal.NicMap{
					"11:11:11:11:11:11": &metal.Nic{Name: "swp1", MacAddress: "11:11:11:11:11:11"},
				},
				newNics: metal.NicMap{
					"11:11:11:11:11:11": &metal.Nic{Name: "swp1", MacAddress: "11:11:11:11:11:11"},
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
				oldNics: metal.NicMap{
					"11:11:11:11:11:11": &metal.Nic{Name: "swp1", MacAddress: "11:11:11:11:11:11"},
				},
				newNics: metal.NicMap{
					"11:11:11:11:11:11": &metal.Nic{Name: "swp1", MacAddress: "11:11:11:11:11:11"},
					"11:11:11:11:11:12": &metal.Nic{Name: "swp2", MacAddress: "11:11:11:11:11:12"},
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
				oldNics: metal.NicMap{
					"11:11:11:11:11:11": &metal.Nic{Name: "swp1", MacAddress: "11:11:11:11:11:11"},
				},
				newNics:            metal.NicMap{},
				currentConnections: metal.ConnectionMap{},
			},
			want:    metal.Nics{},
			wantErr: false,
		},
		{
			name: "removing a nic 2",
			args: args{
				oldNics: metal.NicMap{
					"11:11:11:11:11:11": &metal.Nic{Name: "swp1", MacAddress: "11:11:11:11:11:11"},
					"11:11:11:11:11:12": &metal.Nic{Name: "swp2", MacAddress: "11:11:11:11:11:12"},
				},
				newNics: metal.NicMap{
					"11:11:11:11:11:11": &metal.Nic{Name: "swp1", MacAddress: "11:11:11:11:11:11"},
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
				oldNics: metal.NicMap{
					"11:11:11:11:11:11": &metal.Nic{Name: "swp1", MacAddress: "11:11:11:11:11:11"},
					"11:11:11:11:11:12": &metal.Nic{Name: "swp2", MacAddress: "11:11:11:11:11:12"},
				},
				newNics: metal.NicMap{
					"11:11:11:11:11:11": &metal.Nic{Name: "swp1", MacAddress: "11:11:11:11:11:11"},
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
				oldNics: metal.NicMap{
					"11:11:11:11:11:11": &metal.Nic{Name: "swp1", MacAddress: "11:11:11:11:11:11"},
				},
				newNics: metal.NicMap{
					"11:11:11:11:11:11": &metal.Nic{Name: "swp2", MacAddress: "11:11:11:11:11:11"},
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
				oldNics: metal.NicMap{
					"11:11:11:11:11:11": &metal.Nic{Name: "swp1", Vrf: "vrf1", MacAddress: "11:11:11:11:11:11"},
				},
				newNics: metal.NicMap{
					"11:11:11:11:11:11": &metal.Nic{Name: "swp2", Vrf: "vrf2", MacAddress: "11:11:11:11:11:11"},
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
				oldNics: metal.NicMap{
					"11:11:11:11:11:11": &metal.Nic{Name: "swp1", MacAddress: "11:11:11:11:11:11"},
				},
				newNics: metal.NicMap{
					"11:11:11:11:11:11": &metal.Nic{Name: "swp2", MacAddress: "11:11:11:11:11:11"},
				},
				currentConnections: metal.ConnectionMap{
					"machine-uuid-1": metal.Connections{metal.Connection{MachineID: "machine-uuid-1", Nic: metal.Nic{Name: "swp1", MacAddress: "11:11:11:11:11:11"}}},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := updateSwitchNics(tt.args.oldNics, tt.args.newNics, tt.args.currentConnections)
			if (err != nil) != tt.wantErr {
				t.Errorf("updateSwitchNics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got.ByMac(), tt.want.ByMac()) {
				t.Errorf("updateSwitchNics() = %v, want %v", got, tt.want)
			}
		})
	}
}
