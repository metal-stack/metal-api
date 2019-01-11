package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/netbox"
	"github.com/go-openapi/runtime"
	"github.com/stretchr/testify/require"

	restful "github.com/emicklei/go-restful"

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
	ds, mock := datastore.InitMockDB()
	metal.InitMockDBData(mock)

	pub := &emptyPublisher{}
	nb := netbox.New()
	dservice := NewDevice(testlogger, ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	req := httptest.NewRequest("GET", "/v1/device", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result []metal.Device
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Len(t, result, len(metal.TestDevices))
	require.Equal(t, metal.D1.ID, result[0].ID)
	require.Equal(t, metal.D1.Allocation.Name, result[0].Allocation.Name)
	require.Equal(t, metal.Sz1.Name, result[0].Size.Name)
	require.Equal(t, metal.Site1.Name, result[0].Site.Name)
	require.Equal(t, metal.D2.ID, result[1].ID)
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
			dbsites:            []metal.Site{metal.Site1},
			dbsizes:            []metal.Size{metal.Sz1},
			numcores:           1,
			memory:             100,
			expectedStatus:     http.StatusOK,
			expectedIPMIStatus: http.StatusOK,
			expectedSizeName:   metal.Sz1.Name,
			ipmiresult:         []metal.IPMI{ipmi},
			ipmiresulterror:    nil,
		},
		{
			name:               "no ipmi data",
			uuid:               "1",
			siteid:             "1",
			dbsites:            []metal.Site{metal.Site1},
			dbsizes:            []metal.Size{metal.Sz1},
			numcores:           1,
			memory:             100,
			expectedStatus:     http.StatusOK,
			expectedIPMIStatus: http.StatusNotFound,
			expectedSizeName:   metal.Sz1.Name,
			ipmiresult:         []metal.IPMI{},
			ipmiresulterror:    nil,
		},
		{
			name:               "ipmi fetch error",
			uuid:               "1",
			siteid:             "1",
			dbsites:            []metal.Site{metal.Site1},
			dbsizes:            []metal.Size{metal.Sz1},
			numcores:           1,
			memory:             100,
			expectedStatus:     http.StatusOK,
			expectedIPMIStatus: http.StatusInternalServerError,
			expectedSizeName:   metal.Sz1.Name,
			ipmiresult:         []metal.IPMI{},
			ipmiresulterror:    fmt.Errorf("Test Error"),
		},
		{
			name:    "insert existing",
			uuid:    "1",
			siteid:  "1",
			dbsites: []metal.Site{metal.Site1},
			dbsizes: []metal.Size{metal.Sz1},
			// If here D3 is set instead of D1, it fails. ==> The device is never set in the Register device function endpoint below, but is compared to fix value 1 in line 362: require.Equal(t, expectedid, result.ID)
			//mock.On(r.DB("mockdb").Table("device").Filter(r.MockAnything())).Return([]interface{}{metal.D1}, nil) Deswegen, trotz
			// The DB returns D3(Id=3) where the request would be: give ID=1??
			// ==> mock.On(r.DB("mockdb").Table("device").Get("1")).Return(test.dbdevices, nil)
			dbdevices:          []metal.Device{metal.D1},
			numcores:           1,
			memory:             100,
			expectedStatus:     http.StatusOK,
			expectedIPMIStatus: http.StatusOK,
			expectedSizeName:   metal.Sz1.Name,
			ipmiresult:         []metal.IPMI{ipmi},
			ipmiresulterror:    nil,
		},
		{
			name:           "empty uuid",
			uuid:           "",
			siteid:         "1",
			dbsites:        []metal.Site{metal.Site1},
			dbsizes:        []metal.Size{metal.Sz1},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "error when impi update fails",
			uuid:           "1",
			siteid:         "1",
			dbsites:        []metal.Site{metal.Site1},
			dbsizes:        []metal.Size{metal.Sz1},
			ipmidberror:    fmt.Errorf("ipmi insert fails"),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "empty site",
			uuid:           "1",
			siteid:         "",
			dbsites:        nil,
			dbsizes:        []metal.Size{metal.Sz1},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:               "unknown size because wrong cpu",
			uuid:               "1",
			siteid:             "1",
			dbsites:            []metal.Site{metal.Site1},
			dbsizes:            []metal.Size{metal.Sz1},
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
			dbsites:        []metal.Site{metal.Site1},
			dbsizes:        []metal.Size{metal.Sz1},
			numcores:       2,
			memory:         100,
			netboxerror:    fmt.Errorf("this should happen"),
			expectedStatus: http.StatusInternalServerError,
		},
	}
	for _, test := range testdata {
		t.Run(test.name, func(t *testing.T) {
			ds, mock := datastore.InitMockDB()
			mock.On(r.DB("mockdb").Table("ipmi").Insert(r.MockAnything(), r.InsertOpts{
				Conflict: "replace",
			})).Return(emptyResult, test.ipmidberror)

			rr := metal.RegisterDevice{
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

			if len(test.dbdevices) > 0 {
				mock.On(r.DB("mockdb").Table("size").Get(test.dbdevices[0].SizeID)).Return([]metal.Size{metal.Sz1}, nil)
				mock.On(r.DB("mockdb").Table("device").Get(test.dbdevices[0].ID).Replace(r.MockAnything())).Return(emptyResult, nil)
			} else {
				mock.On(r.DB("mockdb").Table("device").Insert(r.MockAnything(), r.InsertOpts{
					Conflict: "replace",
				})).Return(emptyResult, nil)
			}
			mock.On(r.DB("mockdb").Table("ipmi").Get(test.uuid)).Return(test.ipmiresult, test.ipmiresulterror)
			metal.InitMockDBData(mock)
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
			req := httptest.NewRequest("POST", "/v1/device/register", body)
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
			expectedid := metal.D1.ID
			if len(test.dbdevices) > 0 {
				expectedid = test.dbdevices[0].ID
			}
			require.Equal(t, expectedid, result.ID)
			require.Equal(t, test.expectedSizeName, result.Size.Name)
			require.Equal(t, metal.Site1.Name, result.Site.Name)
			// no read ipmi data
			req = httptest.NewRequest("GET", fmt.Sprintf("/v1/device/%s/ipmi", test.uuid), nil)
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
	ds, mock := datastore.InitMockDB()
	metal.InitMockDBData(mock)

	pub := &emptyPublisher{}
	nb := netbox.New()
	dservice := NewDevice(testlogger, ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	rep := metal.ReportAllocation{
		Success:         true,
		ConsolePassword: "blubber",
	}
	js, _ := json.Marshal(rep)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/device/1/report", body)
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
	ds, mock := datastore.InitMockDB()
	metal.InitMockDBData(mock)

	pub := &emptyPublisher{}
	nb := netbox.New()
	dservice := NewDevice(testlogger, ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	rep := metal.ReportAllocation{
		Success:         false,
		ErrorMessage:    "my error message",
		ConsolePassword: "blubber",
	}
	js, _ := json.Marshal(rep)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/device/1/report", body)
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
	ds, mock := datastore.InitMockDB()
	metal.InitMockDBData(mock)

	pub := &emptyPublisher{}
	nb := netbox.New()
	dservice := NewDevice(testlogger, ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	rep := metal.ReportAllocation{
		Success:         false,
		ErrorMessage:    "my error message",
		ConsolePassword: "blubber",
	}
	js, _ := json.Marshal(rep)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/device/999/report", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusNotFound, resp.StatusCode, w.Body.String())
}

func TestReportUnknownFailure(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	metal.InitMockDBData(mock)

	pub := &emptyPublisher{}
	nb := netbox.New()
	dservice := NewDevice(testlogger, ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	rep := metal.ReportAllocation{
		Success:         false,
		ErrorMessage:    "my error message",
		ConsolePassword: "blubber",
	}
	js, _ := json.Marshal(rep)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/device/404/report", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode, w.Body.String())
}

func TestReportUnallocatedDevice(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	metal.InitMockDBData(mock)

	pub := &emptyPublisher{}
	nb := netbox.New()
	dservice := NewDevice(testlogger, ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	rep := metal.ReportAllocation{
		Success:         true,
		ErrorMessage:    "",
		ConsolePassword: "blubber",
	}
	js, _ := json.Marshal(rep)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/device/3/report", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode, w.Body.String())
}

func TestGetDevice(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	metal.InitMockDBData(mock)
	pub := &emptyPublisher{}
	nb := netbox.New()
	dservice := NewDevice(testlogger, ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	req := httptest.NewRequest("GET", "/v1/device/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result metal.Device
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, metal.D1.ID, result.ID)
	require.Equal(t, metal.D1.Allocation.Name, result.Allocation.Name)
	require.Equal(t, metal.Sz1.Name, result.Size.Name)
	require.Equal(t, metal.Img1.Name, result.Allocation.Image.Name)
	require.Equal(t, metal.Site1.Name, result.Site.Name)
}

func TestGetDeviceNotFound(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	metal.InitMockDBData(mock)

	pub := &emptyPublisher{}
	nb := netbox.New()
	dservice := NewDevice(testlogger, ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	req := httptest.NewRequest("GET", "/v1/device/999", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusNotFound, resp.StatusCode, w.Body.String())
}
func TestFreeDevice(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	metal.InitMockDBData(mock)

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
	req := httptest.NewRequest("DELETE", "/v1/device/1/free", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	require.True(t, called, "netbox.DoRelease was not called")
}

func TestSearchDevice(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	mock.On(r.DB("mockdb").Table("device").Filter(r.MockAnything())).Return([]interface{}{metal.D1}, nil)
	metal.InitMockDBData(mock)

	pub := &emptyPublisher{}
	nb := netbox.New()
	dservice := NewDevice(testlogger, ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	req := httptest.NewRequest("GET", "/v1/device/find?mac=1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var results []metal.Device
	err := json.NewDecoder(resp.Body).Decode(&results)
	require.Nil(t, err)
	require.Len(t, results, 1)
	result := results[0]
	require.Equal(t, metal.D1.ID, result.ID)
	require.Equal(t, metal.D1.Allocation.Name, result.Allocation.Name)
	require.Equal(t, metal.Sz1.Name, result.Size.Name)
	require.Equal(t, metal.Img1.Name, result.Allocation.Image.Name)
	require.Equal(t, metal.Site1.Name, result.Site.Name)
}
