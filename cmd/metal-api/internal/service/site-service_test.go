package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/testdata"
	"github.com/stretchr/testify/require"

	restful "github.com/emicklei/go-restful"
)

func TestGetSites(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	siteservice := NewSite(testdata.Testlogger, ds)
	container := restful.NewContainer().Add(siteservice)
	req := httptest.NewRequest("GET", "/v1/site", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result []metal.Site
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Len(t, result, 3)
	require.Equal(t, testdata.Site1.ID, result[0].ID)
	require.Equal(t, testdata.Site1.Name, result[0].Name)
	require.Equal(t, testdata.Site2.ID, result[1].ID)
	require.Equal(t, testdata.Site2.Name, result[1].Name)
	require.Equal(t, testdata.Site3.ID, result[2].ID)
	require.Equal(t, testdata.Site3.Name, result[2].Name)
}

func TestGetSite(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	siteservice := NewSite(testdata.Testlogger, ds)
	container := restful.NewContainer().Add(siteservice)
	req := httptest.NewRequest("GET", "/v1/site/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result metal.Site
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, testdata.Site1.ID, result.ID)
	require.Equal(t, testdata.Site1.Name, result.Name)
}

func TestGetSiteNotFound(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	siteservice := NewSite(testdata.Testlogger, ds)
	container := restful.NewContainer().Add(siteservice)
	req := httptest.NewRequest("GET", "/v1/site/999", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusNotFound, resp.StatusCode, w.Body.String())
}

func TestDeleteSite(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	siteservice := NewSite(testdata.Testlogger, ds)
	container := restful.NewContainer().Add(siteservice)
	req := httptest.NewRequest("DELETE", "/v1/site/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result metal.Site
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, testdata.Site1.ID, result.ID)
	require.Equal(t, testdata.Site1.Name, result.Name)
}

func TestCreateSite(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	siteservice := NewSite(testdata.Testlogger, ds)
	container := restful.NewContainer().Add(siteservice)

	js, _ := json.Marshal(testdata.Site1)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("PUT", "/v1/site", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusCreated, resp.StatusCode, w.Body.String())
	var result metal.Site
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, testdata.Site1.ID, result.ID)
	require.Equal(t, testdata.Site1.Name, result.Name)
	require.Equal(t, testdata.Site1.Description, result.Description)
}

func TestUpdateSite(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	siteservice := NewSite(testdata.Testlogger, ds)
	container := restful.NewContainer().Add(siteservice)

	js, _ := json.Marshal(testdata.Site1)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/site", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result metal.Site
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, testdata.Site1.ID, result.ID)
	require.Equal(t, testdata.Site1.Name, result.Name)
	require.Equal(t, testdata.Site1.Description, result.Description)
}
