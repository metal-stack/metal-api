package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"github.com/stretchr/testify/require"

	restful "github.com/emicklei/go-restful"

	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

func TestGetSizes(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	mock.On(r.DB("mockdb").Table("size")).Return([]interface{}{metal.Sz1, metal.Sz2}, nil)

	sizeservice := NewSize(testlogger, ds)
	container := restful.NewContainer().Add(sizeservice)
	req := httptest.NewRequest("GET", "/v1/size", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result []metal.Size
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Len(t, result, 2)
	require.Equal(t, metal.Sz1.ID, result[0].ID)
	require.Equal(t, metal.Sz1.Name, result[0].Name)
	require.Equal(t, metal.Sz1.Description, result[0].Description)
	require.Equal(t, metal.Sz2.ID, result[1].ID)
	require.Equal(t, metal.Sz2.Name, result[1].Name)
	require.Equal(t, metal.Sz2.Description, result[1].Description)
}

func TestGetSize(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	mock.On(r.DB("mockdb").Table("size").Get("1")).Return(metal.Sz1, nil)

	sizeservice := NewSize(testlogger, ds)
	container := restful.NewContainer().Add(sizeservice)
	req := httptest.NewRequest("GET", "/v1/size/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result metal.Size
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, metal.Sz1.ID, result.ID)
	require.Equal(t, metal.Sz1.Name, result.Name)
	require.Equal(t, metal.Sz1.Description, result.Description)
}

func TestGetSizeNotFound(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	mock.On(r.DB("mockdb").Table("size").Get("1")).Return(nil, nil)

	sizeservice := NewSize(testlogger, ds)
	container := restful.NewContainer().Add(sizeservice)
	req := httptest.NewRequest("GET", "/v1/size/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusNotFound, resp.StatusCode, w.Body.String())
}

func TestDeleteSize(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	mock.On(r.DB("mockdb").Table("size").Get("1")).Return(metal.Sz1, nil)
	mock.On(r.DB("mockdb").Table("size").Get("1").Delete()).Return(emptyResult, nil)

	sizeservice := NewSize(testlogger, ds)
	container := restful.NewContainer().Add(sizeservice)
	req := httptest.NewRequest("DELETE", "/v1/size/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result metal.Size
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, metal.Sz1.ID, result.ID)
	require.Equal(t, metal.Sz1.Name, result.Name)
	require.Equal(t, metal.Sz1.Description, result.Description)
}

func TestCreateSize(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	mock.On(r.DB("mockdb").Table("size").Get("1")).Return(metal.Sz1, nil)
	mock.On(r.DB("mockdb").Table("size").Insert(r.MockAnything())).Return(emptyResult, nil)

	sizeservice := NewSize(testlogger, ds)
	container := restful.NewContainer().Add(sizeservice)

	js, _ := json.Marshal(metal.Sz1)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("PUT", "/v1/size", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusCreated, resp.StatusCode, w.Body.String())
	var result metal.Size
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, metal.Sz1.ID, result.ID)
	require.Equal(t, metal.Sz1.Name, result.Name)
	require.Equal(t, metal.Sz1.Description, result.Description)
}

func TestUpdateSize(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	mock.On(r.DB("mockdb").Table("size").Get("1")).Return(metal.Sz1, nil)
	mock.On(r.DB("mockdb").Table("size").Get("1").Replace(func(t r.Term) r.Term {
		return r.MockAnything()
	})).Return(emptyResult, nil)

	sizeservice := NewSize(testlogger, ds)
	container := restful.NewContainer().Add(sizeservice)

	js, _ := json.Marshal(metal.Sz1)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/size", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result metal.Size
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, metal.Sz1.ID, result.ID)
	require.Equal(t, metal.Sz1.Name, result.Name)
	require.Equal(t, metal.Sz1.Description, result.Description)
}
