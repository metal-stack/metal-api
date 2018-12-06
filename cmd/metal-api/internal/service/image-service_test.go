package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"github.com/stretchr/testify/require"

	"github.com/emicklei/go-restful"

	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

func TestGetImages(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("image")).Return([]interface{}{img1, img2}, nil)

	imageservice := NewImage(testlogger, ds)
	container := restful.NewContainer().Add(imageservice)
	req := httptest.NewRequest("GET", "/v1/image", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result []metal.Site
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Len(t, result, 2)
	require.Equal(t, img1.ID, result[0].ID)
	require.Equal(t, img1.Name, result[0].Name)
	require.Equal(t, img2.ID, result[1].ID)
	require.Equal(t, img2.Name, result[1].Name)
}

func TestGetImage(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("image").Get("1")).Return(img1, nil)

	imageservice := NewImage(testlogger, ds)
	container := restful.NewContainer().Add(imageservice)
	req := httptest.NewRequest("GET", "/v1/image/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result metal.Site
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, img1.ID, result.ID)
	require.Equal(t, img1.Name, result.Name)
}

func TestGetImageNotFound(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("image").Get("1")).Return(nil, nil)

	imageservice := NewImage(testlogger, ds)
	container := restful.NewContainer().Add(imageservice)
	req := httptest.NewRequest("GET", "/v1/image/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusNotFound, resp.StatusCode, w.Body.String())
}

func TestDeleteImage(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("image").Get("1")).Return([]interface{}{img1}, nil)
	mock.On(r.DB("mockdb").Table("image").Get("1").Delete()).Return(emptyResult, nil)

	imageservice := NewImage(testlogger, ds)
	container := restful.NewContainer().Add(imageservice)
	req := httptest.NewRequest("DELETE", "/v1/image/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result metal.Site
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, img1.ID, result.ID)
	require.Equal(t, img1.Name, result.Name)
}

func TestCreateImage(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("image").Get("1")).Return(img1, nil)
	mock.On(r.DB("mockdb").Table("image").Insert(r.MockAnything())).Return(emptyResult, nil)

	imageservice := NewImage(testlogger, ds)
	container := restful.NewContainer().Add(imageservice)

	js, _ := json.Marshal(img1)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("PUT", "/v1/image", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusCreated, resp.StatusCode, w.Body.String())
	var result metal.Site
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, img1.ID, result.ID)
	require.Equal(t, img1.Name, result.Name)
}

func TestUpdateImage(t *testing.T) {
	ds, mock := initMockDB()

	mock.On(r.DB("mockdb").Table("image").Get("1")).Return(img1, nil)
	mock.On(r.DB("mockdb").Table("image").Get("1").Replace(func(t r.Term) r.Term {
		return r.MockAnything()
	})).Return([]interface{}{
		map[string]interface{}{},
	}, nil)

	imageservice := NewImage(testlogger, ds)
	container := restful.NewContainer().Add(imageservice)

	js, _ := json.Marshal(img1)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/image", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result metal.Site
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, img1.ID, result.ID)
	require.Equal(t, img1.Name, result.Name)
}
