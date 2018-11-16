package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/netbox"
	"git.f-i-ts.de/cloud-native/metal/metal-api/metal"
	"github.com/go-openapi/runtime"
	"github.com/stretchr/testify/require"

	"github.com/emicklei/go-restful"

	nbdevice "git.f-i-ts.de/cloud-native/metal/metal-api/netbox-api/client/devices"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

type emptyPublisher struct {
	doPublish func(topic string, data interface{}) error
}

func (p *emptyPublisher) Publish(topic string, data interface{}) error {
	if p.doPublish != nil {
		return p.doPublish(topic, data)
	}
	return nil
}

func (p *emptyPublisher) CreateTopic(topic string) error {
	return nil
}

func TestGetDevices(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("device")).Return([]interface{}{d1, d2}, nil)
	mock.On(r.DB("mockdb").Table("size")).Return([]interface{}{sz1, sz2}, nil)
	mock.On(r.DB("mockdb").Table("image")).Return([]interface{}{img1}, nil)
	mock.On(r.DB("mockdb").Table("site")).Return([]interface{}{site1}, nil)

	pub := &emptyPublisher{}
	nb := netbox.New()
	dservice := NewDevice(testlogger, ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	req := httptest.NewRequest("GET", "/device", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result []metal.Device
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Len(t, result, 2)
	require.Equal(t, d1.ID, result[0].ID)
	require.Equal(t, d1.Allocation.Name, result[0].Allocation.Name)
	require.Equal(t, sz1.Name, result[0].Size.Name)
	require.Equal(t, site1.Name, result[0].Site.Name)
	require.Equal(t, d2.ID, result[1].ID)
}

func TestGetDevice(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("device").Get("1")).Return([]interface{}{d1, d2}, nil)
	mock.On(r.DB("mockdb").Table("size").Get("1")).Return([]interface{}{sz1}, nil)
	mock.On(r.DB("mockdb").Table("image").Get("1")).Return([]interface{}{img1}, nil)
	mock.On(r.DB("mockdb").Table("site").Get("1")).Return([]interface{}{site1}, nil)

	pub := &emptyPublisher{}
	nb := netbox.New()
	dservice := NewDevice(testlogger, ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	req := httptest.NewRequest("GET", "/device/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result metal.Device
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, d1.ID, result.ID)
	require.Equal(t, d1.Allocation.Name, result.Allocation.Name)
	require.Equal(t, sz1.Name, result.Size.Name)
	require.Equal(t, img1.Name, result.Allocation.Image.Name)
	require.Equal(t, site1.Name, result.Site.Name)
}

func TestGetDeviceNotFound(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("device").Get("1")).Return(nil, nil)

	pub := &emptyPublisher{}
	nb := netbox.New()
	dservice := NewDevice(testlogger, ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	req := httptest.NewRequest("GET", "/device/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusNotFound, resp.StatusCode, w.Body.String())
}
func TestFreeDevice(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("device").Get("1")).Return(d1, nil)
	mock.On(r.DB("mockdb").Table("size").Get("1")).Return(sz1, nil)
	mock.On(r.DB("mockdb").Table("image").Get("1")).Return(img1, nil)
	mock.On(r.DB("mockdb").Table("device").Get("1").Replace(func(t r.Term) r.Term {
		return r.MockAnything()
	})).Return(emptyResult, nil)

	pub := &emptyPublisher{}
	pub.doPublish = func(topic string, data interface{}) error {
		require.Equal(t, "device", topic)
		dv := data.(metal.DeviceEvent)
		require.Equal(t, "1", dv.Old.ID)
		return nil
	}
	nb := netbox.New()
	called := false
	nb.DoRelease = func(params *nbdevice.NetboxAPIProxyAPIDeviceReleaseParams, authInfo runtime.ClientAuthInfoWriter) (*nbdevice.NetboxAPIProxyAPIDeviceReleaseOK, error) {
		called = true
		return &nbdevice.NetboxAPIProxyAPIDeviceReleaseOK{}, nil
	}
	dservice := NewDevice(testlogger, ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	req := httptest.NewRequest("DELETE", "/device/1/free", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	require.True(t, called, "netbox.DoRelease was not called")
}
