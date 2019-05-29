package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/testdata"

	"github.com/stretchr/testify/require"

	restful "github.com/emicklei/go-restful"
)

func TestCreateSwitch(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	switchservice := NewSwitch(ds)
	container := restful.NewContainer().Add(switchservice)

	js, _ := json.Marshal(metal.RegisterSwitch{
		ID:          "switch999",
		PartitionID: "1",
		RackID:      "1",
	})
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/switch/register", body)
	req.Header.Add("Content-Type", "application/json")
	container = injectEditor(container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusCreated, resp.StatusCode, w.Body.String())
	var result metal.Switch
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, "switch999", result.ID)
	require.Equal(t, "switch999", result.Name)
	require.Equal(t, "1", result.RackID)
	require.Equal(t, "1", result.PartitionID)
	require.Len(t, result.Connections, 0)
}

func TestUpdateSwitch(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	switchservice := NewSwitch(ds)
	container := restful.NewContainer().Add(switchservice)

	js, _ := json.Marshal(metal.RegisterSwitch{
		ID:          testdata.Switch1.ID,
		PartitionID: testdata.Switch1.PartitionID,
		RackID:      testdata.Switch1.RackID,
	})
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/switch/register", body)
	req.Header.Add("Content-Type", "application/json")
	container = injectEditor(container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result metal.Switch
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, testdata.Switch1.ID, result.ID)
	require.Equal(t, testdata.Switch1.ID, result.Name)
	require.Equal(t, testdata.Switch1.RackID, result.RackID)
	require.Equal(t, testdata.Switch1.PartitionID, result.PartitionID)
	require.Len(t, result.Connections, 2)
	con := result.Connections[0]
	require.Equal(t, testdata.Switch1.MachineConnections["1"][0].Nic.MacAddress, con.Nic.MacAddress)
}
