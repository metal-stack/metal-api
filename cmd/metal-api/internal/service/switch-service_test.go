package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.f-i-ts.de/cloud-native/metallib/httperrors"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	v1 "git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/service/v1"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/testdata"
	restful "github.com/emicklei/go-restful"
	"github.com/stretchr/testify/require"
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
			Describeable: v1.Describeable{
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
		Nics:        v1.SwitchNicsExtended{},
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

	resp := w.Result()
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode, w.Body.String())
	var result httperrors.HTTPErrorResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, http.StatusUnprocessableEntity, result.StatusCode)
	require.Regexp(t, "nic with mac address 11:11:11:11:11:11 gets removed but the machine with id \"1\" is already connected to this nic", result.Message)
}

func TestConnectMachineWithSwitches(t *testing.T) {
	type args struct {
		dev *metal.Machine
	}
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
