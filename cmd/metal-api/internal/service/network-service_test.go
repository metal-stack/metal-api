package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/metal-stack/metal-lib/httperrors"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/ipam"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"

	restful "github.com/emicklei/go-restful/v3"
	goipam "github.com/metal-stack/go-ipam"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
	"github.com/stretchr/testify/require"
)

func TestGetNetworks(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	networkservice := NewNetwork(ds, ipam.New(goipam.New()), nil)
	container := restful.NewContainer().Add(networkservice)
	req := httptest.NewRequest("GET", "/v1/network", nil)
	container = injectViewer(container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result []v1.NetworkResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
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

	networkservice := NewNetwork(ds, ipam.New(goipam.New()), nil)
	container := restful.NewContainer().Add(networkservice)
	req := httptest.NewRequest("GET", "/v1/network/1", nil)
	container = injectViewer(container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.NetworkResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.Nw1.ID, result.ID)
	require.Equal(t, testdata.Nw1.Name, *result.Name)
}

func TestGetNetworkNotFound(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	networkservice := NewNetwork(ds, ipam.New(goipam.New()), nil)
	container := restful.NewContainer().Add(networkservice)
	req := httptest.NewRequest("GET", "/v1/network/999", nil)
	container = injectViewer(container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode, w.Body.String())
	var result httperrors.HTTPErrorResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Contains(t, result.Message, "999")
	require.Equal(t, 404, result.StatusCode)
}

func TestDeleteNetwork(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	mock.On(r.DB("mockdb").Table("network").Filter(r.MockAnything())).Return([]interface{}{}, nil)
	ipamer, err := testdata.InitMockIpamData(mock, false)
	require.NoError(t, err)
	testdata.InitMockDBData(mock)

	networkservice := NewNetwork(ds, ipamer, nil)
	container := restful.NewContainer().Add(networkservice)
	req := httptest.NewRequest("DELETE", "/v1/network/"+testdata.NwIPAM.ID, nil)
	container = injectAdmin(container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.NetworkResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.NwIPAM.ID, result.ID)
	require.Equal(t, testdata.NwIPAM.Name, *result.Name)
}

func TestDeleteNetworkIPInUse(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	mock.On(r.DB("mockdb").Table("network").Filter(r.MockAnything())).Return([]interface{}{}, nil)
	ipamer, err := testdata.InitMockIpamData(mock, true)
	require.NoError(t, err)
	testdata.InitMockDBData(mock)

	networkservice := NewNetwork(ds, ipamer, nil)
	container := restful.NewContainer().Add(networkservice)
	req := httptest.NewRequest("DELETE", "/v1/network/"+testdata.NwIPAM.ID, nil)
	container = injectAdmin(container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode, w.Body.String())
	var result httperrors.HTTPErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, 422, result.StatusCode)
	require.Contains(t, result.Message, "unable to delete network: prefix 10.0.0.0/16 has ip 10.0.0.1 in use")
}

func TestCreateNetwork(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	ipamer, err := testdata.InitMockIpamData(mock, false)
	require.NoError(t, err)
	testdata.InitMockDBData(mock)

	networkservice := NewNetwork(ds, ipamer, nil)
	container := restful.NewContainer().Add(networkservice)

	prefixes := []string{"172.0.0.0/24"}
	destPrefixes := []string{"0.0.0.0/0"}
	vrf := uint(10000)
	createRequest := &v1.NetworkCreateRequest{
		Describable:      v1.Describable{Name: &testdata.Nw1.Name},
		NetworkBase:      v1.NetworkBase{PartitionID: &testdata.Nw1.PartitionID, ProjectID: &testdata.Nw1.ProjectID},
		NetworkImmutable: v1.NetworkImmutable{Prefixes: prefixes, DestinationPrefixes: destPrefixes, Vrf: &vrf},
	}
	js, err := json.Marshal(createRequest)
	require.NoError(t, err)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("PUT", "/v1/network", body)
	req.Header.Add("Content-Type", "application/json")
	container = injectAdmin(container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode, w.Body.String())
	var result v1.NetworkResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.Nw1.Name, *result.Name)
	require.Equal(t, testdata.Nw1.PartitionID, *result.PartitionID)
	require.Equal(t, testdata.Nw1.ProjectID, *result.ProjectID)
	require.Equal(t, destPrefixes, result.DestinationPrefixes)
}

func TestUpdateNetwork(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	networkservice := NewNetwork(ds, ipam.New(goipam.New()), nil)
	container := restful.NewContainer().Add(networkservice)

	newName := "new"
	updateRequest := &v1.NetworkUpdateRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{ID: testdata.Nw1.GetID()},
			Describable:  v1.Describable{Name: &newName},
		},
	}
	js, err := json.Marshal(updateRequest)
	require.NoError(t, err)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/network", body)
	req.Header.Add("Content-Type", "application/json")
	container = injectAdmin(container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result metal.Partition
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.Nw1.ID, result.ID)
	require.Equal(t, newName, result.Name)
}

func TestSearchNetwork(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	mock.On(r.DB("mockdb").Table("network").Filter(r.MockAnything())).Return([]interface{}{testdata.Nw1}, nil)
	testdata.InitMockDBData(mock)

	networkService := NewNetwork(ds, ipam.New(goipam.New()), nil)
	container := restful.NewContainer().Add(networkService)
	requestJSON := fmt.Sprintf("{%q:%q}", "partitionid", "1")
	req := httptest.NewRequest("POST", "/v1/network/find", bytes.NewBufferString(requestJSON))
	req.Header.Add("Content-Type", "application/json")
	container = injectViewer(container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var results []v1.NetworkResponse
	err := json.NewDecoder(resp.Body).Decode(&results)

	require.NoError(t, err)
	require.Len(t, results, 1)
	result := results[0]
	require.Equal(t, testdata.Nw1.ID, result.ID)
	require.Equal(t, testdata.Nw1.PartitionID, *result.PartitionID)
	require.Equal(t, testdata.Nw1.Name, *result.Name)
}
