package service

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func TestSearchDevice(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("device").Filter(r.MockAnything())).Return([]interface{}{d1}, nil)
	mock.On(r.DB("mockdb").Table("size")).Return([]interface{}{sz1}, nil)
	mock.On(r.DB("mockdb").Table("image")).Return([]interface{}{img1}, nil)
	mock.On(r.DB("mockdb").Table("site")).Return([]interface{}{site1}, nil)

	pub := &emptyPublisher{}
	nb := netbox.New()
	dservice := NewDevice(testlogger, ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	req := httptest.NewRequest("GET", "/device/find?mac=1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var results []metal.Device
	err := json.NewDecoder(resp.Body).Decode(&results)
	require.Nil(t, err)
	require.Len(t, results, 1)
	result := results[0]
	require.Equal(t, d1.ID, result.ID)
	require.Equal(t, d1.Allocation.Name, result.Allocation.Name)
	require.Equal(t, sz1.Name, result.Size.Name)
	require.Equal(t, img1.Name, result.Allocation.Image.Name)
	require.Equal(t, site1.Name, result.Site.Name)
}

func TestRegisterDevice(t *testing.T) {
	ipmi := metal.IPMI{
		Address:    "address",
		Interface:  "interface",
		MacAddress: "mac",
	}
	testdata := []struct {
		name               string
		uuid               string
		siteid             string
		numcores           int
		memory             int
		dbsites            []metal.Site
		dbsizes            []metal.Size
		dbdevices          []metal.Device
		netboxerror        error
		ipmidberror        error
		ipmiresult         []metal.IPMI
		ipmiresulterror    error
		expectedIPMIStatus int
		expectedStatus     int
		expectedSizeName   string
	}{
		{
			name:               "insert new",
			uuid:               "1",
			siteid:             "1",
			dbsites:            []metal.Site{site1},
			dbsizes:            []metal.Size{sz1},
			numcores:           1,
			memory:             100,
			expectedStatus:     http.StatusOK,
			expectedIPMIStatus: http.StatusOK,
			expectedSizeName:   sz1.Name,
			ipmiresult:         []metal.IPMI{ipmi},
			ipmiresulterror:    nil,
		},
		{
			name:               "no ipmi data",
			uuid:               "1",
			siteid:             "1",
			dbsites:            []metal.Site{site1},
			dbsizes:            []metal.Size{sz1},
			numcores:           1,
			memory:             100,
			expectedStatus:     http.StatusOK,
			expectedIPMIStatus: http.StatusNotFound,
			expectedSizeName:   sz1.Name,
			ipmiresult:         []metal.IPMI{},
			ipmiresulterror:    nil,
		},
		{
			name:               "ipmi fetch error",
			uuid:               "1",
			siteid:             "1",
			dbsites:            []metal.Site{site1},
			dbsizes:            []metal.Size{sz1},
			numcores:           1,
			memory:             100,
			expectedStatus:     http.StatusOK,
			expectedIPMIStatus: http.StatusInternalServerError,
			expectedSizeName:   sz1.Name,
			ipmiresult:         []metal.IPMI{},
			ipmiresulterror:    fmt.Errorf("nope"),
		},
		{
			name:               "insert existing",
			uuid:               "1",
			siteid:             "1",
			dbsites:            []metal.Site{site1},
			dbsizes:            []metal.Size{sz1},
			dbdevices:          []metal.Device{d3},
			numcores:           1,
			memory:             100,
			expectedStatus:     http.StatusOK,
			expectedIPMIStatus: http.StatusOK,
			expectedSizeName:   sz1.Name,
			ipmiresult:         []metal.IPMI{ipmi},
			ipmiresulterror:    nil,
		},
		{
			name:           "empty uuid",
			uuid:           "",
			siteid:         "1",
			dbsites:        []metal.Site{site1},
			dbsizes:        []metal.Size{sz1},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "error when impi update fails",
			uuid:           "1",
			siteid:         "1",
			dbsites:        []metal.Site{site1},
			dbsizes:        []metal.Size{sz1},
			ipmidberror:    fmt.Errorf("ipmi insert fails"),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "empty site",
			uuid:           "1",
			siteid:         "",
			dbsites:        nil,
			dbsizes:        []metal.Size{sz1},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:               "unknown size because wrong cpu",
			uuid:               "1",
			siteid:             "1",
			dbsites:            []metal.Site{site1},
			dbsizes:            []metal.Size{sz1},
			numcores:           2,
			memory:             100,
			expectedStatus:     http.StatusOK,
			expectedSizeName:   metal.UnknownSize.Name,
			ipmiresult:         []metal.IPMI{ipmi},
			expectedIPMIStatus: http.StatusOK,
			ipmiresulterror:    nil,
		},
		{
			name:           "fail on netbox error",
			uuid:           "1",
			siteid:         "1",
			dbsites:        []metal.Site{site1},
			dbsizes:        []metal.Size{sz1},
			numcores:       2,
			memory:         100,
			netboxerror:    fmt.Errorf("this should happen"),
			expectedStatus: http.StatusInternalServerError,
		},
	}
	for _, test := range testdata {
		t.Run(test.name, func(t *testing.T) {
			ds, mock := initMockDB()

			rr := registerRequest{
				UUID:   test.uuid,
				SiteID: test.siteid,
				RackID: "1",
				IPMI:   ipmi,
				Hardware: metal.DeviceHardware{
					CPUCores: test.numcores,
					Memory:   uint64(test.memory),
				},
			}

			mock.On(r.DB("mockdb").Table("site").Get(test.siteid)).Return(test.dbsites, nil)
			mock.On(r.DB("mockdb").Table("size")).Return([]metal.Size{sz1}, nil)
			mock.On(r.DB("mockdb").Table("device").Get("1")).Return(test.dbdevices, nil)
			// TODO Filter mock is currently noop, makes switch result in device service empty.
			mock.On(r.DB("mockdb").Table("switch").Filter(r.MockAnything(), r.FilterOpts{})).Return([]metal.Switch{}, nil)

			if len(test.dbdevices) > 0 {
				mock.On(r.DB("mockdb").Table("size").Get(test.dbdevices[0].SizeID)).Return([]metal.Size{sz1}, nil)
				mock.On(r.DB("mockdb").Table("device").Get(test.dbdevices[0].ID).Replace(r.MockAnything())).Return(emptyResult, nil)
			} else {
				mock.On(r.DB("mockdb").Table("device").Insert(r.MockAnything(), r.InsertOpts{
					Conflict: "replace",
				})).Return(emptyResult, nil)
			}
			mock.On(r.DB("mockdb").Table("ipmi").Insert(r.MockAnything(), r.InsertOpts{
				Conflict: "replace",
			})).Return(emptyResult, test.ipmidberror)
			mock.On(r.DB("mockdb").Table("ipmi").Get(test.uuid)).Return(test.ipmiresult, test.ipmiresulterror)
			pub := &emptyPublisher{}
			nb := netbox.New()
			called := false
			nb.DoRegister = func(params *nbdevice.NetboxAPIProxyAPIDeviceRegisterParams, authInfo runtime.ClientAuthInfoWriter) (*nbdevice.NetboxAPIProxyAPIDeviceRegisterOK, error) {
				called = true
				return &nbdevice.NetboxAPIProxyAPIDeviceRegisterOK{}, test.netboxerror
			}
			js, _ := json.Marshal(rr)
			body := bytes.NewBuffer(js)

			dservice := NewDevice(testlogger, ds, pub, nb)
			container := restful.NewContainer().Add(dservice)
			req := httptest.NewRequest("POST", "/device/register", body)
			req.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()
			container.ServeHTTP(w, req)

			resp := w.Result()
			require.Equal(t, test.expectedStatus, resp.StatusCode, w.Body.String())
			if resp.StatusCode >= 300 {
				return
			}
			var result metal.Device
			err := json.NewDecoder(resp.Body).Decode(&result)
			require.Nil(t, err)
			require.True(t, called, "netbox register was not called")
			expectedid := d1.ID
			if len(test.dbdevices) > 0 {
				expectedid = test.dbdevices[0].ID
			}
			require.Equal(t, expectedid, result.ID)
			require.Equal(t, test.expectedSizeName, result.Size.Name)
			require.Equal(t, site1.Name, result.Site.Name)
			// no read ipmi data
			req = httptest.NewRequest("POST", fmt.Sprintf("/device/%s/ipmi", test.uuid), nil)
			req.Header.Add("Content-Type", "application/json")
			w = httptest.NewRecorder()
			container.ServeHTTP(w, req)

			resp = w.Result()
			require.Equal(t, test.expectedIPMIStatus, resp.StatusCode, w.Body.String())
			if resp.StatusCode >= 300 {
				return
			}
			var ipmiresult metal.IPMI
			err = json.NewDecoder(resp.Body).Decode(&ipmiresult)
			require.Nil(t, err)
			require.Equal(t, ipmi.Address, ipmiresult.Address)
			require.Equal(t, ipmi.Interface, ipmiresult.Interface)
			require.Equal(t, ipmi.MacAddress, ipmiresult.MacAddress)
		})
	}
}

func TestReportDevice(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("device").Get("1")).Return(d1, nil)
	mock.On(r.DB("mockdb").Table("size").Get("1")).Return(sz1, nil)
	mock.On(r.DB("mockdb").Table("image").Get("1")).Return(img1, nil)
	mock.On(r.DB("mockdb").Table("device").Get("1").Replace(func(t r.Term) r.Term {
		return r.MockAnything()
	})).Return(emptyResult, nil)

	mock.On(r.DB("mockdb").Table("site")).Return([]interface{}{site1}, nil)

	pub := &emptyPublisher{}
	nb := netbox.New()
	dservice := NewDevice(testlogger, ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	rep := allocationReport{
		Success:         true,
		ConsolePassword: "blubber",
	}
	js, _ := json.Marshal(rep)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/device/1/report", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result metal.DeviceAllocation
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, result.ConsolePassword, rep.ConsolePassword)
}

func TestReportFailureDevice(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("device").Get("1")).Return(d1, nil)
	mock.On(r.DB("mockdb").Table("size").Get("1")).Return(sz1, nil)
	mock.On(r.DB("mockdb").Table("image").Get("1")).Return(img1, nil)

	pub := &emptyPublisher{}
	nb := netbox.New()
	dservice := NewDevice(testlogger, ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	rep := allocationReport{
		Success:         false,
		ErrorMessage:    "my error message",
		ConsolePassword: "blubber",
	}
	js, _ := json.Marshal(rep)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/device/1/report", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result metal.DeviceAllocation
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
}

func TestReportUnknownDevice(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("device").Get("10")).Return(nil, nil)

	pub := &emptyPublisher{}
	nb := netbox.New()
	dservice := NewDevice(testlogger, ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	rep := allocationReport{
		Success:         false,
		ErrorMessage:    "my error message",
		ConsolePassword: "blubber",
	}
	js, _ := json.Marshal(rep)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/device/10/report", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusNotFound, resp.StatusCode, w.Body.String())
}

func TestReportUnknownFailure(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("device").Get("10")).Return(nil, fmt.Errorf("nope"))

	pub := &emptyPublisher{}
	nb := netbox.New()
	dservice := NewDevice(testlogger, ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	rep := allocationReport{
		Success:         false,
		ErrorMessage:    "my error message",
		ConsolePassword: "blubber",
	}
	js, _ := json.Marshal(rep)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/device/10/report", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode, w.Body.String())
}

func TestReportUnallocatedDevice(t *testing.T) {
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("device").Get("1")).Return(d3, nil)
	mock.On(r.DB("mockdb").Table("size").Get("1")).Return(sz1, nil)
	mock.On(r.DB("mockdb").Table("image").Get("1")).Return(img1, nil)

	pub := &emptyPublisher{}
	nb := netbox.New()
	dservice := NewDevice(testlogger, ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	rep := allocationReport{
		Success:         true,
		ErrorMessage:    "",
		ConsolePassword: "blubber",
	}
	js, _ := json.Marshal(rep)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/device/1/report", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode, w.Body.String())
}
