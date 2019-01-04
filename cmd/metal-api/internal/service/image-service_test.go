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
)

func TestGetImages(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	metal.InitMockDBData(mock)

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
	require.Len(t, result, 3)
	require.Equal(t, metal.Img1.ID, result[0].ID)
	require.Equal(t, metal.Img1.Name, result[0].Name)
	require.Equal(t, metal.Img2.ID, result[1].ID)
	require.Equal(t, metal.Img2.Name, result[1].Name)
	require.Equal(t, metal.Img3.ID, result[2].ID)
	require.Equal(t, metal.Img3.Name, result[2].Name)
}

func TestGetImage(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	metal.InitMockDBData(mock)

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
	require.Equal(t, metal.Img1.ID, result.ID)
	require.Equal(t, metal.Img1.Name, result.Name)
}

func TestGetImageNotFound(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	metal.InitMockDBData(mock)

	imageservice := NewImage(testlogger, ds)
	container := restful.NewContainer().Add(imageservice)
	req := httptest.NewRequest("GET", "/v1/image/999", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusNotFound, resp.StatusCode, w.Body.String())
}

func TestDeleteImage(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	metal.InitMockDBData(mock)

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
	require.Equal(t, metal.Img1.ID, result.ID)
	require.Equal(t, metal.Img1.Name, result.Name)
}

func TestCreateImage(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	metal.InitMockDBData(mock)

	imageservice := NewImage(testlogger, ds)
	container := restful.NewContainer().Add(imageservice)

	js, _ := json.Marshal(metal.Img1)
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
	require.Equal(t, metal.Img1.ID, result.ID)
	require.Equal(t, metal.Img1.Name, result.Name)
}

func TestUpdateImage(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	metal.InitMockDBData(mock)

	imageservice := NewImage(testlogger, ds)
	container := restful.NewContainer().Add(imageservice)

	js, _ := json.Marshal(metal.Img1)
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
	require.Equal(t, metal.Img1.ID, result.ID)
	require.Equal(t, metal.Img1.Name, result.Name)
}
