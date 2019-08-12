package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	v1 "git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/service/v1"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/testdata"
	"git.f-i-ts.de/cloud-native/metallib/httperrors"
	restful "github.com/emicklei/go-restful"
	"github.com/stretchr/testify/require"
)

type nopTopicCreater struct {
}

func (n nopTopicCreater) CreateTopic(partitionID, topicFQN string) error {
	return nil
}

type expectingTopicCreater struct {
	t              *testing.T
	expectedTopics []string
}

func (n expectingTopicCreater) CreateTopic(partitionID, topicFQN string) error {
	assert := assert.New(n.t)
	assert.NotEmpty(topicFQN)
	assert.Contains(n.expectedTopics, topicFQN, "Expecatation %v contains %s failed.", n.expectedTopics, topicFQN)
	return nil
}

func TestGetPartitions(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	service := NewPartition(ds, &nopTopicCreater{})
	container := restful.NewContainer().Add(service)
	req := httptest.NewRequest("GET", "/v1/partition", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result []v1.PartitionResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
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
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	service := NewPartition(ds, &nopTopicCreater{})
	container := restful.NewContainer().Add(service)
	req := httptest.NewRequest("GET", "/v1/partition/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.PartitionResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, testdata.Partition1.ID, result.ID)
	require.Equal(t, testdata.Partition1.Name, *result.Name)
	require.Equal(t, testdata.Partition1.Description, *result.Description)
}

func TestGetPartitionNotFound(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	service := NewPartition(ds, &nopTopicCreater{})
	container := restful.NewContainer().Add(service)
	req := httptest.NewRequest("GET", "/v1/partition/999", nil)
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

func TestDeletePartition(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	service := NewPartition(ds, &nopTopicCreater{})
	container := restful.NewContainer().Add(service)
	req := httptest.NewRequest("DELETE", "/v1/partition/1", nil)
	container = injectAdmin(container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.PartitionResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, testdata.Partition1.ID, result.ID)
	require.Equal(t, testdata.Partition1.Name, *result.Name)
	require.Equal(t, testdata.Partition1.Description, *result.Description)
}

func TestCreatePartition(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	topicCreater := expectingTopicCreater{
		t:              t,
		expectedTopics: []string{"1-switch"},
	}
	service := NewPartition(ds, topicCreater)
	container := restful.NewContainer().Add(service)

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
	}
	js, _ := json.Marshal(createRequest)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("PUT", "/v1/partition", body)
	req.Header.Add("Content-Type", "application/json")
	container = injectAdmin(container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusCreated, resp.StatusCode, w.Body.String())
	var result v1.PartitionResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, testdata.Partition1.ID, result.ID)
	require.Equal(t, testdata.Partition1.Name, *result.Name)
	require.Equal(t, testdata.Partition1.Description, *result.Description)
}

func TestUpdatePartition(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	service := NewPartition(ds, &nopTopicCreater{})
	container := restful.NewContainer().Add(service)

	mgmtService := "mgmt"
	imageURL := "http://somewhere/image1.zip"
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
			ImageURL: &imageURL,
		},
	}
	js, _ := json.Marshal(updateRequest)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/partition", body)
	req.Header.Add("Content-Type", "application/json")
	container = injectAdmin(container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.PartitionResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, testdata.Partition1.ID, result.ID)
	require.Equal(t, testdata.Partition2.Name, *result.Name)
	require.Equal(t, testdata.Partition2.Description, *result.Description)
	require.Equal(t, mgmtService, *result.MgmtServiceAddress)
	require.Equal(t, imageURL, *result.PartitionBootConfiguration.ImageURL)
}

func TestPartitionCapacity(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	service := NewPartition(ds, &nopTopicCreater{})
	container := restful.NewContainer().Add(service)

	req := httptest.NewRequest("GET", "/v1/partition/capacity", nil)
	req.Header.Add("Content-Type", "application/json")
	container = injectAdmin(container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result []v1.PartitionCapacity
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, testdata.Partition1.ID, result[0].ID)
	require.NotNil(t, result[0].ServerCapacities)
	require.Equal(t, 1, len(result[0].ServerCapacities))
	cap := result[0].ServerCapacities[0]
	require.Equal(t, "1", cap.Size)
	require.Equal(t, 5, cap.Total)
	require.Equal(t, 0, cap.Free)
}
