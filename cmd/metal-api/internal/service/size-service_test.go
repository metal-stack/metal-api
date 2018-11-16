package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-api/metal"
	"github.com/stretchr/testify/require"

	"github.com/emicklei/go-restful"

	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

func TestGetSizes(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("size")).Return([]interface{}{sz1, sz2}, nil)

	sizeservice := NewSize(testlogger, ds)
	container := restful.NewContainer().Add(sizeservice)
	req := httptest.NewRequest("GET", "/size", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result []metal.Size
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Len(t, result, 2)
	require.Equal(t, sz1.ID, result[0].ID)
	require.Equal(t, sz1.Name, result[0].Name)
	require.Equal(t, sz1.Description, result[0].Description)
	require.Equal(t, sz2.ID, result[1].ID)
	require.Equal(t, sz2.Name, result[1].Name)
	require.Equal(t, sz2.Description, result[1].Description)
}

func TestGetSize(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("size").Get("1")).Return(sz1, nil)

	sizeservice := NewSize(testlogger, ds)
	container := restful.NewContainer().Add(sizeservice)
	req := httptest.NewRequest("GET", "/size/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result metal.Size
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, sz1.ID, result.ID)
	require.Equal(t, sz1.Name, result.Name)
	require.Equal(t, sz1.Description, result.Description)
}

func TestGetSizeNotFound(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("size").Get("1")).Return(nil, nil)

	sizeservice := NewSize(testlogger, ds)
	container := restful.NewContainer().Add(sizeservice)
	req := httptest.NewRequest("GET", "/size/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusNotFound, resp.StatusCode, w.Body.String())
}

func TestDeleteSize(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("size").Get("1")).Return(sz1, nil)
	mock.On(r.DB("mockdb").Table("size").Get("1").Delete()).Return(emptyResult, nil)

	sizeservice := NewSize(testlogger, ds)
	container := restful.NewContainer().Add(sizeservice)
	req := httptest.NewRequest("DELETE", "/size/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result metal.Size
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, sz1.ID, result.ID)
	require.Equal(t, sz1.Name, result.Name)
	require.Equal(t, sz1.Description, result.Description)
}

func TestCreateSize(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("size").Get("1")).Return(sz1, nil)
	mock.On(r.DB("mockdb").Table("size").Insert(r.MockAnything())).Return(emptyResult, nil)

	sizeservice := NewSize(testlogger, ds)
	container := restful.NewContainer().Add(sizeservice)

	js, _ := json.Marshal(sz1)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("PUT", "/size", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusCreated, resp.StatusCode, w.Body.String())
	var result metal.Size
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, sz1.ID, result.ID)
	require.Equal(t, sz1.Name, result.Name)
	require.Equal(t, sz1.Description, result.Description)
}

func TestUpdateSize(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("size").Get("1")).Return(sz1, nil)
	mock.On(r.DB("mockdb").Table("size").Get("1").Replace(func(t r.Term) r.Term {
		return r.MockAnything()
	})).Return(emptyResult, nil)

	sizeservice := NewSize(testlogger, ds)
	container := restful.NewContainer().Add(sizeservice)

	js, _ := json.Marshal(sz1)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/size", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result metal.Size
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, sz1.ID, result.ID)
	require.Equal(t, sz1.Name, result.Name)
	require.Equal(t, sz1.Description, result.Description)
}
