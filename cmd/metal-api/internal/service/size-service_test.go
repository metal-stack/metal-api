package service

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	mdmv1mock "github.com/metal-stack/masterdata-api/api/v1/mocks"
	mdm "github.com/metal-stack/masterdata-api/pkg/client"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	restful "github.com/emicklei/go-restful/v3"
)

func TestGetSizes(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)

	sizeservice := NewSize(zaptest.NewLogger(t).Sugar(), ds, nil)
	container := restful.NewContainer().Add(sizeservice)
	req := httptest.NewRequest("GET", "/v1/size", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result []v1.SizeResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Len(t, result, 3)
	require.Equal(t, testdata.Sz1.ID, result[0].ID)
	require.Equal(t, testdata.Sz1.Name, *result[0].Name)
	require.Equal(t, testdata.Sz1.Description, *result[0].Description)
	require.Equal(t, testdata.Sz2.ID, result[1].ID)
	require.Equal(t, testdata.Sz2.Name, *result[1].Name)
	require.Equal(t, testdata.Sz2.Description, *result[1].Description)
	require.Equal(t, testdata.Sz3.ID, result[2].ID)
	require.Equal(t, testdata.Sz3.Name, *result[2].Name)
	require.Equal(t, testdata.Sz3.Description, *result[2].Description)
}

func TestGetSize(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)

	sizeservice := NewSize(zaptest.NewLogger(t).Sugar(), ds, nil)
	container := restful.NewContainer().Add(sizeservice)
	req := httptest.NewRequest("GET", "/v1/size/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.SizeResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.Sz1.ID, result.ID)
	require.Equal(t, testdata.Sz1.Name, *result.Name)
	require.Equal(t, testdata.Sz1.Description, *result.Description)
	require.Equal(t, len(testdata.Sz1.Constraints), len(result.SizeConstraints))
}

func TestSuggest(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)

	createRequest := v1.SizeSuggestRequest{
		MachineID: "1",
	}
	js, err := json.Marshal(createRequest)
	require.NoError(t, err)
	body := bytes.NewBuffer(js)

	sizeservice := NewSize(zaptest.NewLogger(t).Sugar(), ds, nil)
	container := restful.NewContainer().Add(sizeservice)
	req := httptest.NewRequest("POST", "/v1/size/suggest", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result []v1.SizeConstraint
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	require.Len(t, result, 3)

	assert.Contains(t, result, v1.SizeConstraint{
		Type: metal.MemoryConstraint,
		Min:  1 << 30,
		Max:  1 << 30,
	})

	assert.Contains(t, result, v1.SizeConstraint{
		Type: metal.CoreConstraint,
		Min:  8,
		Max:  8,
	})

	assert.Contains(t, result, v1.SizeConstraint{
		Type: metal.StorageConstraint,
		Min:  3000,
		Max:  3000,
	})

}

func TestGetSizeNotFound(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)

	sizeservice := NewSize(zaptest.NewLogger(t).Sugar(), ds, nil)
	container := restful.NewContainer().Add(sizeservice)
	req := httptest.NewRequest("GET", "/v1/size/999", nil)
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

func TestDeleteSize(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := zaptest.NewLogger(t).Sugar()

	sizeservice := NewSize(log, ds, nil)
	container := restful.NewContainer().Add(sizeservice)
	req := httptest.NewRequest("DELETE", "/v1/size/1", nil)
	container = injectAdmin(log, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.SizeResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.Sz1.ID, result.ID)
	require.Equal(t, testdata.Sz1.Name, *result.Name)
	require.Equal(t, testdata.Sz1.Description, *result.Description)
}

func TestCreateSize(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := zaptest.NewLogger(t).Sugar()

	psc := &mdmv1mock.ProjectServiceClient{}
	psc.On("Find", context.Background(), &mdmv1.ProjectFindRequest{}).Return(&mdmv1.ProjectListResponse{Projects: []*mdmv1.Project{
		{Meta: &mdmv1.Meta{Id: "a"}},
	}}, nil)
	mdc := mdm.NewMock(psc, &mdmv1mock.TenantServiceClient{})

	sizeservice := NewSize(log, ds, mdc)
	container := restful.NewContainer().Add(sizeservice)

	createRequest := v1.SizeCreateRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: testdata.Sz1.ID,
			},
			Describable: v1.Describable{
				Name:        &testdata.Sz1.Name,
				Description: &testdata.Sz1.Description,
			},
		},
		SizeConstraints: []v1.SizeConstraint{
			{
				Type: metal.CoreConstraint,
				Min:  15,
				Max:  27,
			},
			{
				Type: metal.MemoryConstraint,
				Min:  100,
				Max:  100,
			},
		},
		SizeReservations: []v1.SizeReservation{
			{
				Amount:       3,
				ProjectID:    "a",
				PartitionIDs: []string{testdata.Partition1.ID},
				Description:  "test",
			},
		},
	}
	js, err := json.Marshal(createRequest)
	require.NoError(t, err)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("PUT", "/v1/size", body)
	req.Header.Add("Content-Type", "application/json")
	container = injectAdmin(log, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode, w.Body.String())
	var result v1.SizeResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.Sz1.ID, result.ID)
	require.Equal(t, testdata.Sz1.Name, *result.Name)
	require.Equal(t, testdata.Sz1.Description, *result.Description)
}

func TestUpdateSize(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := zaptest.NewLogger(t).Sugar()

	psc := &mdmv1mock.ProjectServiceClient{}
	psc.On("Find", context.Background(), &mdmv1.ProjectFindRequest{}).Return(&mdmv1.ProjectListResponse{Projects: []*mdmv1.Project{
		{Meta: &mdmv1.Meta{Id: "a"}},
	}}, nil)
	mdc := mdm.NewMock(psc, &mdmv1mock.TenantServiceClient{})

	sizeservice := NewSize(log, ds, mdc)
	container := restful.NewContainer().Add(sizeservice)

	minCores := uint64(8)
	maxCores := uint64(16)
	updateRequest := v1.SizeUpdateRequest{
		Common: v1.Common{
			Describable: v1.Describable{
				Name:        &testdata.Sz2.Name,
				Description: &testdata.Sz2.Description,
			},
			Identifiable: v1.Identifiable{
				ID: testdata.Sz1.ID,
			},
		},
		SizeConstraints: &[]v1.SizeConstraint{
			{
				Type: metal.CoreConstraint,
				Min:  minCores,
				Max:  maxCores,
			},
		},
	}
	js, err := json.Marshal(updateRequest)
	require.NoError(t, err)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/size", body)
	req.Header.Add("Content-Type", "application/json")
	container = injectAdmin(log, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.SizeResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.Sz1.ID, result.ID)
	require.Equal(t, testdata.Sz2.Name, *result.Name)
	require.Equal(t, testdata.Sz2.Description, *result.Description)
	require.Equal(t, metal.CoreConstraint, result.SizeConstraints[0].Type)
	require.Equal(t, minCores, result.SizeConstraints[0].Min)
	require.Equal(t, maxCores, result.SizeConstraints[0].Max)
}
