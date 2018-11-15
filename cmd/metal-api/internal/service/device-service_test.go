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
	mock.On(r.DB("mockdb").Table("device")).Return([]interface{}{
		map[string]interface{}{"id": 1, "name": "d1", "sizeid": 1, "imageid": 1, "siteid": 1},
		map[string]interface{}{"id": 2, "name": "d2", "sizeid": 2},
	}, nil)
	mock.On(r.DB("mockdb").Table("size")).Return([]interface{}{
		map[string]interface{}{"id": 1, "name": "sz1"},
		map[string]interface{}{"id": 2, "name": "sz2"},
	}, nil)
	mock.On(r.DB("mockdb").Table("image")).Return([]interface{}{
		map[string]interface{}{"id": 1, "name": "i1"},
	}, nil)
	mock.On(r.DB("mockdb").Table("site")).Return([]interface{}{
		map[string]interface{}{"id": 1, "name": "s1"},
	}, nil)

	pub := &emptyPublisher{}
	nb := netbox.New()
	dservice := NewDevice(testlogger, ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	req := httptest.NewRequest("GET", "/device", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var result []metal.Device
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Len(t, result, 2)
	require.Equal(t, "1", result[0].ID)
	require.Equal(t, "d1", result[0].Name)
	require.Equal(t, "sz1", result[0].Size.Name)
	require.Equal(t, "i1", result[0].Image.Name)
	require.Equal(t, "s1", result[0].Site.Name)
	require.Equal(t, "2", result[1].ID)
	require.Equal(t, "d2", result[1].Name)
}

func TestGetDevice(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("device").Get("1")).Return([]interface{}{
		map[string]interface{}{"id": 1, "name": "d1", "sizeid": 1, "imageid": 1, "siteid": 1},
		map[string]interface{}{"id": 2, "name": "d2", "sizeid": 2},
	}, nil)
	mock.On(r.DB("mockdb").Table("size").Get("1")).Return([]interface{}{
		map[string]interface{}{"id": 1, "name": "sz1"},
	}, nil)
	mock.On(r.DB("mockdb").Table("image").Get("1")).Return([]interface{}{
		map[string]interface{}{"id": 1, "name": "i1"},
	}, nil)
	mock.On(r.DB("mockdb").Table("site").Get("1")).Return([]interface{}{
		map[string]interface{}{"id": 1, "name": "s1"},
	}, nil)

	pub := &emptyPublisher{}
	nb := netbox.New()
	dservice := NewDevice(testlogger, ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	req := httptest.NewRequest("GET", "/device/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var result metal.Device
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, "1", result.ID)
	require.Equal(t, "d1", result.Name)
	require.Equal(t, "sz1", result.Size.Name)
	require.Equal(t, "i1", result.Image.Name)
	require.Equal(t, "s1", result.Site.Name)
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
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}
func TestFreeDevice(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("device").Get("1")).Return([]interface{}{
		map[string]interface{}{"id": 1, "name": "d1", "project": 1},
	}, nil)
	mock.On(r.DB("mockdb").Table("device").Get("1").Replace(func(t r.Term) r.Term {
		return r.MockAnything()
	})).Return([]interface{}{
		map[string]interface{}{"id": 1, "name": "d1", "project": 1},
	}, nil)

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
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.True(t, called, "netbox.DoRelease was not called")
}
