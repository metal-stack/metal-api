package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.f-i-ts.de/cloud-native/metallib/httperrors"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/ipam"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	v1 "git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/service/v1"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/testdata"
	restful "github.com/emicklei/go-restful"
	goipam "github.com/metal-pod/go-ipam"
	"github.com/stretchr/testify/require"
)

func TestGetNetworks(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	networkservice := NewNetwork(ds, ipam.New(goipam.New()))
	container := restful.NewContainer().Add(networkservice)
	req := httptest.NewRequest("GET", "/v1/network", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result []v1.NetworkResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Len(t, result, 4)
	require.Equal(t, testdata.Nw1.ID, result[0].ID)
	require.Equal(t, testdata.Nw1.Name, *result[0].Name)
	require.Equal(t, testdata.Nw2.ID, result[1].ID)
	require.Equal(t, testdata.Nw2.Name, *result[1].Name)
	require.Equal(t, testdata.Nw3.ID, result[2].ID)
	require.Equal(t, testdata.Nw3.Name, *result[2].Name)
}

func TestGetNetwork(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	networkservice := NewNetwork(ds, ipam.New(goipam.New()))
	container := restful.NewContainer().Add(networkservice)
	req := httptest.NewRequest("GET", "/v1/network/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.NetworkResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, testdata.Nw1.ID, result.ID)
	require.Equal(t, testdata.Nw1.Name, *result.Name)
}

func TestGetNetworkNotFound(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	networkservice := NewNetwork(ds, ipam.New(goipam.New()))
	container := restful.NewContainer().Add(networkservice)
	req := httptest.NewRequest("GET", "/v1/network/999", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusNotFound, resp.StatusCode, w.Body.String())
	var result httperrors.HTTPErrorResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Contains(t, result.Message, "999")
	require.Equal(t, 404, result.StatusCode)
}

func TestDeleteNetwork(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	ipamer, err := testdata.InitMockIpamData(mock, false)
	require.Nil(t, err)
	testdata.InitMockDBData(mock)

	networkservice := NewNetwork(ds, ipamer)
	container := restful.NewContainer().Add(networkservice)
	req := httptest.NewRequest("DELETE", "/v1/network/"+testdata.NwIPAM.ID, nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.NetworkResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, testdata.NwIPAM.ID, result.ID)
	require.Equal(t, testdata.NwIPAM.Name, *result.Name)
}

func TestDeleteNetworkIPInUse(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	ipamer, err := testdata.InitMockIpamData(mock, true)
	require.Nil(t, err)
	testdata.InitMockDBData(mock)

	networkservice := NewNetwork(ds, ipamer)
	container := restful.NewContainer().Add(networkservice)
	req := httptest.NewRequest("DELETE", "/v1/network/"+testdata.NwIPAM.ID, nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode, w.Body.String())
	var result httperrors.HTTPErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, 422, result.StatusCode)
	require.Contains(t, result.Message, "unable to delete network: prefix 10.0.0.0/16 has ip 10.0.0.1 in use")
}

func TestCreateNetwork(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	networkservice := NewNetwork(ds, ipam.New(goipam.New()))
	container := restful.NewContainer().Add(networkservice)

	vrfID := uint(1)
	createRequest := &v1.NetworkCreateRequest{
		Describeable:     v1.Describeable{Name: &testdata.Nw1.Name},
		NetworkBase:      v1.NetworkBase{PartitionID: &testdata.Nw1.PartitionID, ProjectID: &testdata.Nw1.ProjectID},
		NetworkImmutable: v1.NetworkImmutable{Prefixes: testdata.Nw1.Prefixes.String(), Vrf: &vrfID},
	}
	js, _ := json.Marshal(createRequest)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("PUT", "/v1/network", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusCreated, resp.StatusCode, w.Body.String())
	var result v1.NetworkResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, testdata.Nw1.Name, *result.Name)
	require.Equal(t, testdata.Nw1.PartitionID, *result.PartitionID)
	require.Equal(t, testdata.Nw1.ProjectID, *result.ProjectID)
}

func TestUpdateNetwork(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	networkservice := NewNetwork(ds, ipam.New(goipam.New()))
	container := restful.NewContainer().Add(networkservice)

	newName := "new"
	updateRequest := &v1.NetworkUpdateRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{ID: testdata.Nw1.GetID()},
			Describeable: v1.Describeable{Name: &newName}},
	}
	js, _ := json.Marshal(updateRequest)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/network", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result metal.Partition
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, testdata.Nw1.ID, result.ID)
	require.Equal(t, newName, result.Name)
}
