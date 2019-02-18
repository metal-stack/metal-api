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

func TestGetPartitions(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	service := NewPartition(testdata.Testlogger, ds)
	container := restful.NewContainer().Add(service)
	req := httptest.NewRequest("GET", "/v1/partition", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result []metal.Partition
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Len(t, result, 3)
	require.Equal(t, testdata.Partition1.ID, result[0].ID)
	require.Equal(t, testdata.Partition1.Name, result[0].Name)
	require.Equal(t, testdata.Partition2.ID, result[1].ID)
	require.Equal(t, testdata.Partition2.Name, result[1].Name)
	require.Equal(t, testdata.Partition3.ID, result[2].ID)
	require.Equal(t, testdata.Partition3.Name, result[2].Name)
}

func TestGetPartition(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	service := NewPartition(testdata.Testlogger, ds)
	container := restful.NewContainer().Add(service)
	req := httptest.NewRequest("GET", "/v1/partition/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result metal.Partition
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, testdata.Partition1.ID, result.ID)
	require.Equal(t, testdata.Partition1.Name, result.Name)
}

func TestGetPartitionNotFound(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	service := NewPartition(testdata.Testlogger, ds)
	container := restful.NewContainer().Add(service)
	req := httptest.NewRequest("GET", "/v1/partition/999", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusNotFound, resp.StatusCode, w.Body.String())
}

func TestDeletePartition(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	service := NewPartition(testdata.Testlogger, ds)
	container := restful.NewContainer().Add(service)
	req := httptest.NewRequest("DELETE", "/v1/partition/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result metal.Partition
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, testdata.Partition1.ID, result.ID)
	require.Equal(t, testdata.Partition1.Name, result.Name)
}

func TestCreatePartition(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	service := NewPartition(testdata.Testlogger, ds)
	container := restful.NewContainer().Add(service)

	js, _ := json.Marshal(testdata.Partition1)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("PUT", "/v1/partition", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusCreated, resp.StatusCode, w.Body.String())
	var result metal.Partition
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, testdata.Partition1.ID, result.ID)
	require.Equal(t, testdata.Partition1.Name, result.Name)
	require.Equal(t, testdata.Partition1.Description, result.Description)
}

func TestUpdatePartition(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	service := NewPartition(testdata.Testlogger, ds)
	container := restful.NewContainer().Add(service)

	js, _ := json.Marshal(testdata.Partition1)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/partition", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result metal.Partition
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, testdata.Partition1.ID, result.ID)
	require.Equal(t, testdata.Partition1.Name, result.Name)
	require.Equal(t, testdata.Partition1.Description, result.Description)
}
