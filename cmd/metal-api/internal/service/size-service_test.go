package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	v1 "git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/service/v1"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/testdata"
	"git.f-i-ts.de/cloud-native/metallib/httperrors"
	"github.com/stretchr/testify/require"

	restful "github.com/emicklei/go-restful"
)

func TestGetSizes(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	sizeservice := NewSize(ds)
	container := restful.NewContainer().Add(sizeservice)
	req := httptest.NewRequest("GET", "/v1/size", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result []v1.SizeListResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
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
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	sizeservice := NewSize(ds)
	container := restful.NewContainer().Add(sizeservice)
	req := httptest.NewRequest("GET", "/v1/size/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.SizeDetailResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, testdata.Sz1.ID, result.ID)
	require.Equal(t, testdata.Sz1.Name, *result.Name)
	require.Equal(t, testdata.Sz1.Description, *result.Description)
	require.Equal(t, len(testdata.Sz1.Constraints), len(result.SizeConstraints))
}

func TestGetSizeNotFound(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	sizeservice := NewSize(ds)
	container := restful.NewContainer().Add(sizeservice)
	req := httptest.NewRequest("GET", "/v1/size/999", nil)
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

func TestDeleteSize(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	sizeservice := NewSize(ds)
	container := restful.NewContainer().Add(sizeservice)
	req := httptest.NewRequest("DELETE", "/v1/size/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.SizeDetailResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, testdata.Sz1.ID, result.ID)
	require.Equal(t, testdata.Sz1.Name, *result.Name)
	require.Equal(t, testdata.Sz1.Description, *result.Description)
}

func TestCreateSize(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	sizeservice := NewSize(ds)
	container := restful.NewContainer().Add(sizeservice)

	createRequest := v1.SizeCreateRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: testdata.Sz1.ID,
			},
			Describeable: v1.Describeable{
				Name:        &testdata.Sz1.Name,
				Description: &testdata.Sz1.Description,
			},
		},
	}
	js, _ := json.Marshal(createRequest)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("PUT", "/v1/size", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusCreated, resp.StatusCode, w.Body.String())
	var result v1.SizeDetailResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, testdata.Sz1.Name, *result.Name)
	require.Equal(t, testdata.Sz1.Description, *result.Description)
}

func TestUpdateSize(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	sizeservice := NewSize(ds)
	container := restful.NewContainer().Add(sizeservice)

	minCores := uint64(1)
	maxCores := uint64(4)
	updateRequest := v1.SizeUpdateRequest{
		Common: v1.Common{
			Describeable: v1.Describeable{
				Name:        &testdata.Sz2.Name,
				Description: &testdata.Sz2.Description,
			},
			Identifiable: v1.Identifiable{
				ID: testdata.Sz1.ID,
			},
		},
		SizeConstraints: &[]v1.SizeConstraint{
			v1.SizeConstraint{
				Type: metal.CoreConstraint,
				Min:  minCores,
				Max:  maxCores,
			},
		},
	}
	js, _ := json.Marshal(updateRequest)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/size", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.SizeDetailResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, testdata.Sz1.ID, result.ID)
	require.Equal(t, testdata.Sz2.Name, *result.Name)
	require.Equal(t, testdata.Sz2.Description, *result.Description)
	require.Equal(t, metal.CoreConstraint, result.SizeConstraints[0].Type)
	require.Equal(t, minCores, result.SizeConstraints[0].Min)
	require.Equal(t, maxCores, result.SizeConstraints[0].Max)
}
