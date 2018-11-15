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
	mock.On(r.DB("mockdb").Table("size")).Return([]interface{}{
		map[string]interface{}{"id": 1, "name": "size1", "description": "description 1"},
		map[string]interface{}{"id": 2, "name": "size2", "description": "description 2"},
	}, nil)

	sizeservice := NewSize(testlogger, ds)
	container := restful.NewContainer().Add(sizeservice)
	req := httptest.NewRequest("GET", "/size", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var result []metal.Size
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Len(t, result, 2)
	require.Equal(t, result[0].ID, "1")
	require.Equal(t, result[0].Name, "size1")
	require.Equal(t, result[0].Description, "description 1")
	require.Equal(t, result[1].ID, "2")
	require.Equal(t, result[1].Name, "size2")
	require.Equal(t, result[1].Description, "description 2")
}

func TestGetSize(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("size").Get("1")).Return([]interface{}{
		map[string]interface{}{"id": 1, "name": "size1", "description": "description 1"},
	}, nil)

	sizeservice := NewSize(testlogger, ds)
	container := restful.NewContainer().Add(sizeservice)
	req := httptest.NewRequest("GET", "/size/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var result metal.Size
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, result.ID, "1")
	require.Equal(t, result.Name, "size1")
	require.Equal(t, result.Description, "description 1")
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
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestDeleteSize(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("size").Get("1")).Return([]interface{}{
		map[string]interface{}{"id": 1, "name": "size1", "description": "description 1"},
	}, nil)
	mock.On(r.DB("mockdb").Table("size").Get("1").Delete()).Return([]interface{}{
		map[string]interface{}{"id": 1, "name": "size1", "description": "description 1"},
	}, nil)

	sizeservice := NewSize(testlogger, ds)
	container := restful.NewContainer().Add(sizeservice)
	req := httptest.NewRequest("DELETE", "/size/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var result metal.Size
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, "1", result.ID)
	require.Equal(t, "size1", result.Name)
	require.Equal(t, "description 1", result.Description)
}

func TestCreateSize(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("size").Get("1")).Return([]interface{}{
		map[string]interface{}{"id": 1, "name": "size1", "description": "description 1"},
	}, nil)
	mock.On(r.DB("mockdb").Table("size").Insert(r.MockAnything())).Return([]interface{}{
		map[string]interface{}{"id": 1, "name": "size1", "description": "description 1"},
	}, nil)

	sizeservice := NewSize(testlogger, ds)
	container := restful.NewContainer().Add(sizeservice)

	sz := metal.Size{
		ID:          "1",
		Name:        "size1",
		Description: "description 1",
	}
	js, _ := json.Marshal(sz)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("PUT", "/size", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var result metal.Size
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, "1", result.ID)
	require.Equal(t, "size1", result.Name)
	require.Equal(t, "description 1", result.Description)
}

func TestUpdateSize(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("size").Get("1")).Return([]interface{}{
		map[string]interface{}{"id": 1, "name": "size1", "description": "description 1"},
	}, nil)
	mock.On(r.DB("mockdb").Table("size").Get("1").Replace(func(t r.Term) r.Term {
		return r.MockAnything()
	})).Return([]interface{}{
		map[string]interface{}{"id": 1, "name": "size1", "description": "description 1"},
	}, nil)

	sizeservice := NewSize(testlogger, ds)
	container := restful.NewContainer().Add(sizeservice)

	sz := metal.Size{
		ID:          "1",
		Name:        "size1",
		Description: "description 1",
	}
	js, _ := json.Marshal(sz)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/size", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var result metal.Size
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, "1", result.ID)
	require.Equal(t, "size1", result.Name)
	require.Equal(t, "description 1", result.Description)
}
