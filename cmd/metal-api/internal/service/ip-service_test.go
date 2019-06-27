package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/ipam"
	v1 "git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/service/v1"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/testdata"

	"git.f-i-ts.de/cloud-native/metallib/httperrors"

	goipam "github.com/metal-pod/go-ipam"
	"github.com/stretchr/testify/require"

	restful "github.com/emicklei/go-restful"
)

func TestGetIPs(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	ipservice := NewIP(ds, ipam.New(goipam.New()))
	container := restful.NewContainer().Add(ipservice)
	req := httptest.NewRequest("GET", "/v1/ip", nil)
	container = injectViewer(container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result []v1.IPResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Len(t, result, 3)
	require.Equal(t, testdata.IP1.IPAddress, result[0].IPAddress)
	require.Equal(t, testdata.IP1.Name, *result[0].Name)
	require.Equal(t, testdata.IP2.IPAddress, result[1].IPAddress)
	require.Equal(t, testdata.IP2.Name, *result[1].Name)
	require.Equal(t, testdata.IP3.IPAddress, result[2].IPAddress)
	require.Equal(t, testdata.IP3.Name, *result[2].Name)
}

func TestGetIP(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	ipservice := NewIP(ds, ipam.New(goipam.New()))
	container := restful.NewContainer().Add(ipservice)
	req := httptest.NewRequest("GET", "/v1/ip/1.2.3.4", nil)
	container = injectViewer(container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.IPResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, testdata.IP1.IPAddress, result.IPAddress)
	require.Equal(t, testdata.IP1.Name, *result.Name)
}

func TestGetIPNotFound(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	ipservice := NewIP(ds, ipam.New(goipam.New()))
	container := restful.NewContainer().Add(ipservice)
	req := httptest.NewRequest("GET", "/v1/ip/9.9.9.9", nil)
	container = injectViewer(container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusNotFound, resp.StatusCode, w.Body.String())
	var result httperrors.HTTPErrorResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Contains(t, result.Message, "9.9.9.9")
	require.Equal(t, 404, result.StatusCode)
}

func TestDeleteIP(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	ipamer, err := testdata.InitMockIpamData(mock, true)
	require.Nil(t, err)
	testdata.InitMockDBData(mock)

	ipservice := NewIP(ds, ipamer)
	container := restful.NewContainer().Add(ipservice)

	req := httptest.NewRequest("DELETE", "/v1/ip/"+testdata.IPAMIP.IPAddress, nil)
	container = injectEditor(container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.IPResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, testdata.IPAMIP.IPAddress, result.IPAddress)
	require.Equal(t, testdata.IPAMIP.Name, *result.Name)
}

func TestAllocateIP(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	ipamer, err := testdata.InitMockIpamData(mock, false)
	require.Nil(t, err)
	testdata.InitMockDBData(mock)

	ipservice := NewIP(ds, ipamer)
	container := restful.NewContainer().Add(ipservice)

	name := "testip1"
	allocateRequest := v1.IPAllocateRequest{
		Describable: v1.Describable{Name: &name},
		IPBase:      v1.IPBase{ProjectID: "123", NetworkID: testdata.NwIPAM.ID},
	}
	js, _ := json.Marshal(allocateRequest)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/ip/allocate", body)
	container = injectEditor(container, req)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusCreated, resp.StatusCode, w.Body.String())
	var result v1.IPResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, "10.0.0.1", result.IPAddress)
	require.Equal(t, name, *result.Name)
	require.Equal(t, allocateRequest.ProjectID, result.ProjectID)
}

func TestUpdateIP(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	ipservice := NewIP(ds, ipam.New(goipam.New()))
	container := restful.NewContainer().Add(ipservice)

	updateRequest := v1.IPUpdateRequest{
		Describable: v1.Describable{
			Name:        &testdata.IP2.Name,
			Description: &testdata.IP2.Description,
		},
		IPIdentifiable: v1.IPIdentifiable{
			IPAddress: testdata.IP1.IPAddress,
		},
	}
	js, _ := json.Marshal(updateRequest)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/ip", body)
	container = injectEditor(container, req)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.IPResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, testdata.IP1.IPAddress, result.IPAddress)
	require.Equal(t, testdata.IP2.Name, *result.Name)
}
