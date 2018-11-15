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

func TestGetImages(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("image")).Return([]interface{}{
		map[string]interface{}{"id": 1, "name": "image1", "description": "description 1"},
		map[string]interface{}{"id": 2, "name": "image2", "description": "description 2"},
	}, nil)

	imageservice := NewImage(testlogger, ds)
	container := restful.NewContainer().Add(imageservice)
	req := httptest.NewRequest("GET", "/image", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var result []metal.Site
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Len(t, result, 2)
	require.Equal(t, "1", result[0].ID)
	require.Equal(t, "image1", result[0].Name)
	require.Equal(t, "description 1", result[0].Description)
	require.Equal(t, "2", result[1].ID)
	require.Equal(t, "image2", result[1].Name)
	require.Equal(t, "description 2", result[1].Description)
}

func TestGetImage(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("image").Get("1")).Return([]interface{}{
		map[string]interface{}{"id": 1, "name": "image1", "description": "description 1"},
	}, nil)

	imageservice := NewImage(testlogger, ds)
	container := restful.NewContainer().Add(imageservice)
	req := httptest.NewRequest("GET", "/image/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var result metal.Site
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, "1", result.ID)
	require.Equal(t, "image1", result.Name)
	require.Equal(t, "description 1", result.Description)
}

func TestGetImageNotFound(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("image").Get("1")).Return(nil, nil)

	imageservice := NewImage(testlogger, ds)
	container := restful.NewContainer().Add(imageservice)
	req := httptest.NewRequest("GET", "/image/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestDeleteImage(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("image").Get("1")).Return([]interface{}{
		map[string]interface{}{"id": 1, "name": "image1", "description": "description 1"},
	}, nil)
	mock.On(r.DB("mockdb").Table("image").Get("1").Delete()).Return([]interface{}{
		map[string]interface{}{"id": 1, "name": "image1", "description": "description 1"},
	}, nil)

	imageservice := NewImage(testlogger, ds)
	container := restful.NewContainer().Add(imageservice)
	req := httptest.NewRequest("DELETE", "/image/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var result metal.Site
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, "1", result.ID)
	require.Equal(t, "image1", result.Name)
	require.Equal(t, "description 1", result.Description)
}

func TestCreateImage(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("image").Get("1")).Return([]interface{}{
		map[string]interface{}{"id": 1, "name": "image1", "description": "description 1"},
	}, nil)
	mock.On(r.DB("mockdb").Table("image").Insert(r.MockAnything())).Return([]interface{}{
		map[string]interface{}{"id": 1, "name": "image1", "description": "description 1"},
	}, nil)

	imageservice := NewImage(testlogger, ds)
	container := restful.NewContainer().Add(imageservice)

	sz := metal.Site{
		ID:          "1",
		Name:        "image1",
		Description: "description 1",
	}
	js, _ := json.Marshal(sz)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("PUT", "/image", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var result metal.Site
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, "1", result.ID)
	require.Equal(t, "image1", result.Name)
	require.Equal(t, "description 1", result.Description)
}

func TestUpdateImage(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("image").Get("1")).Return([]interface{}{
		map[string]interface{}{"id": 1, "name": "image1", "description": "description 1"},
	}, nil)
	mock.On(r.DB("mockdb").Table("image").Get("1").Replace(func(t r.Term) r.Term {
		return r.MockAnything()
	})).Return([]interface{}{
		map[string]interface{}{"id": 1, "name": "image1", "description": "description 1"},
	}, nil)

	imageservice := NewImage(testlogger, ds)
	container := restful.NewContainer().Add(imageservice)

	sz := metal.Site{
		ID:          "1",
		Name:        "image1",
		Description: "description 1",
	}
	js, _ := json.Marshal(sz)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/image", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var result metal.Site
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, "1", result.ID)
	require.Equal(t, "image1", result.Name)
	require.Equal(t, "description 1", result.Description)
}
