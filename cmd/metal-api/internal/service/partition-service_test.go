package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"

	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/stretchr/testify/require"
)

type nopTopicCreater struct {
}

func (n nopTopicCreater) CreateTopic(topicFQN string) error {
	return nil
}

type expectingTopicCreater struct {
	t              *testing.T
	expectedTopics []string
}

func (n expectingTopicCreater) CreateTopic(topicFQN string) error {
	ass := assert.New(n.t)
	ass.NotEmpty(topicFQN)
	ass.Contains(n.expectedTopics, topicFQN, "Expectation %v contains %s failed.", n.expectedTopics, topicFQN)
	return nil
}

func TestGetPartitions(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)

	service := NewPartition(zaptest.NewLogger(t).Sugar(), ds, &nopTopicCreater{})
	container := restful.NewContainer().Add(service)
	req := httptest.NewRequest("GET", "/v1/partition", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result []v1.PartitionResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Len(t, result, 3)
	require.Equal(t, testdata.Partition1.ID, result[0].ID)
	require.Equal(t, testdata.Partition1.Name, *result[0].Name)
	require.Equal(t, testdata.Partition1.Description, *result[0].Description)
	require.Equal(t, testdata.Partition2.ID, result[1].ID)
	require.Equal(t, testdata.Partition2.Name, *result[1].Name)
	require.Equal(t, testdata.Partition2.Description, *result[1].Description)
	require.Equal(t, testdata.Partition3.ID, result[2].ID)
	require.Equal(t, testdata.Partition3.Name, *result[2].Name)
	require.Equal(t, testdata.Partition3.Description, *result[2].Description)
}

func TestGetPartition(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)

	service := NewPartition(zaptest.NewLogger(t).Sugar(), ds, &nopTopicCreater{})
	container := restful.NewContainer().Add(service)
	req := httptest.NewRequest("GET", "/v1/partition/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.PartitionResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.Partition1.ID, result.ID)
	require.Equal(t, testdata.Partition1.Name, *result.Name)
	require.Equal(t, testdata.Partition1.Description, *result.Description)
}

func TestGetPartitionNotFound(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := zaptest.NewLogger(t).Sugar()

	service := NewPartition(log, ds, &nopTopicCreater{})
	container := restful.NewContainer().Add(service)
	req := httptest.NewRequest("GET", "/v1/partition/999", nil)
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

func TestDeletePartition(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := zaptest.NewLogger(t).Sugar()

	service := NewPartition(log, ds, &nopTopicCreater{})
	container := restful.NewContainer().Add(service)
	req := httptest.NewRequest("DELETE", "/v1/partition/1", nil)
	container = injectAdmin(log, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.PartitionResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.Partition1.ID, result.ID)
	require.Equal(t, testdata.Partition1.Name, *result.Name)
	require.Equal(t, testdata.Partition1.Description, *result.Description)
}

func TestCreatePartition(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := zaptest.NewLogger(t).Sugar()

	topicCreater := expectingTopicCreater{
		t:              t,
		expectedTopics: []string{"1-switch", "1-machine"},
	}
	service := NewPartition(log, ds, topicCreater)
	container := restful.NewContainer().Add(service)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "I am a downloadable content")
	}))
	defer ts.Close()

	downloadableFile := ts.URL

	createRequest := v1.PartitionCreateRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: testdata.Partition1.ID,
			},
			Describable: v1.Describable{
				Name:        &testdata.Partition1.Name,
				Description: &testdata.Partition1.Description,
			},
		},
		PartitionBootConfiguration: v1.PartitionBootConfiguration{
			ImageURL:  &downloadableFile,
			KernelURL: &downloadableFile,
		},
	}
	js, err := json.Marshal(createRequest)
	require.NoError(t, err)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("PUT", "/v1/partition", body)
	req.Header.Add("Content-Type", "application/json")
	container = injectAdmin(log, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode, w.Body.String())
	var result v1.PartitionResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.Partition1.ID, result.ID)
	require.Equal(t, testdata.Partition1.Name, *result.Name)
	require.Equal(t, testdata.Partition1.Description, *result.Description)
}

func TestUpdatePartition(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := zaptest.NewLogger(t).Sugar()

	service := NewPartition(log, ds, &nopTopicCreater{})
	container := restful.NewContainer().Add(service)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "I am a downloadable content")
	}))
	defer ts.Close()

	mgmtService := "mgmt"
	downloadableFile := ts.URL
	updateRequest := v1.PartitionUpdateRequest{
		Common: v1.Common{
			Describable: v1.Describable{
				Name:        &testdata.Partition2.Name,
				Description: &testdata.Partition2.Description,
			},
			Identifiable: v1.Identifiable{
				ID: testdata.Partition1.ID,
			},
		},
		MgmtServiceAddress: &mgmtService,
		PartitionBootConfiguration: &v1.PartitionBootConfiguration{
			ImageURL: &downloadableFile,
		},
	}
	js, err := json.Marshal(updateRequest)
	require.NoError(t, err)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/partition", body)
	req.Header.Add("Content-Type", "application/json")
	container = injectAdmin(log, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.PartitionResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.Partition1.ID, result.ID)
	require.Equal(t, testdata.Partition2.Name, *result.Name)
	require.Equal(t, testdata.Partition2.Description, *result.Description)
	require.Equal(t, mgmtService, *result.MgmtServiceAddress)
	require.Equal(t, downloadableFile, *result.PartitionBootConfiguration.ImageURL)
}

func TestPartitionCapacity(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)

	ecs := []metal.ProvisioningEventContainer{}
	for _, m := range testdata.TestMachines {
		m := m
		ecs = append(ecs, metal.ProvisioningEventContainer{
			Base: m.Base,
		})
	}
	mock.On(r.DB("mockdb").Table("event")).Return(ecs, nil)

	testdata.InitMockDBData(mock)
	log := zaptest.NewLogger(t).Sugar()

	service := NewPartition(log, ds, &nopTopicCreater{})
	container := restful.NewContainer().Add(service)

	pcRequest := &v1.PartitionCapacityRequest{}
	js, err := json.Marshal(pcRequest)
	require.NoError(t, err)
	body := bytes.NewBuffer(js)

	req := httptest.NewRequest("POST", "/v1/partition/capacity", body)
	req.Header.Add("Content-Type", "application/json")
	container = injectAdmin(log, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result []v1.PartitionCapacity
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, testdata.Partition1.ID, result[0].ID)
	require.NotNil(t, result[0].ServerCapacities)
	require.Len(t, result[0].ServerCapacities, 1)
	c := result[0].ServerCapacities[0]
	require.Equal(t, "1", c.Size)
	require.Equal(t, 5, c.Total)
	require.Equal(t, 0, c.Free)
}
