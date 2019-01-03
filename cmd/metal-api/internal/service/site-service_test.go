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

func TestGetSites(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	mock.On(r.DB("mockdb").Table("site")).Return([]interface{}{metal.Site1, metal.Site2}, nil)

	siteservice := NewSite(testlogger, ds)
	container := restful.NewContainer().Add(siteservice)
	req := httptest.NewRequest("GET", "/v1/site", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result []metal.Site
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Len(t, result, 2)
	require.Equal(t, metal.Site1.ID, result[0].ID)
	require.Equal(t, metal.Site1.Name, result[0].Name)
	require.Equal(t, metal.Site2.ID, result[1].ID)
	require.Equal(t, metal.Site2.Name, result[1].Name)
}

func TestGetSite(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	mock.On(r.DB("mockdb").Table("site").Get("1")).Return(metal.Site1, nil)

	siteservice := NewSite(testlogger, ds)
	container := restful.NewContainer().Add(siteservice)
	req := httptest.NewRequest("GET", "/v1/site/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result metal.Site
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, metal.Site1.ID, result.ID)
	require.Equal(t, metal.Site1.Name, result.Name)
}

func TestGetSiteNotFound(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	mock.On(r.DB("mockdb").Table("site").Get("1")).Return(nil, nil)

	siteservice := NewSite(testlogger, ds)
	container := restful.NewContainer().Add(siteservice)
	req := httptest.NewRequest("GET", "/v1/site/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusNotFound, resp.StatusCode, w.Body.String())
}

func TestDeleteSite(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	mock.On(r.DB("mockdb").Table("site").Get("1")).Return(metal.Site1, nil)
	mock.On(r.DB("mockdb").Table("site").Get("1").Delete()).Return(emptyResult, nil)

	siteservice := NewSite(testlogger, ds)
	container := restful.NewContainer().Add(siteservice)
	req := httptest.NewRequest("DELETE", "/v1/site/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result metal.Site
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, metal.Site1.ID, result.ID)
	require.Equal(t, metal.Site1.Name, result.Name)
}

func TestCreateSite(t *testing.T) {
	ds, mock := datastore.InitMockDB()

	mock.On(r.DB("mockdb").Table("site").Get("1")).Return(metal.Site1, nil)
	mock.On(r.DB("mockdb").Table("site").Insert(r.MockAnything())).Return(emptyResult, nil)

	siteservice := NewSite(testlogger, ds)
	container := restful.NewContainer().Add(siteservice)

	js, _ := json.Marshal(metal.Site1)
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
	require.Equal(t, metal.Site1.ID, result.ID)
	require.Equal(t, metal.Site1.Name, result.Name)
	require.Equal(t, metal.Site1.Description, result.Description)
}

func TestUpdateSite(t *testing.T) {
	ds, mock := datastore.InitMockDB()

	mock.On(r.DB("mockdb").Table("site").Get("1")).Return(metal.Site1, nil)
	mock.On(r.DB("mockdb").Table("site").Get("1").Replace(func(t r.Term) r.Term {
		return r.MockAnything()
	})).Return(emptyResult, nil)

	siteservice := NewSite(testlogger, ds)
	container := restful.NewContainer().Add(siteservice)

	js, _ := json.Marshal(metal.Site1)
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
	require.Equal(t, metal.Site1.ID, result.ID)
	require.Equal(t, metal.Site1.Name, result.Name)
	require.Equal(t, metal.Site1.Description, result.Description)
}
