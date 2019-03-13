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
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/testdata"
	"github.com/go-openapi/runtime"
	"github.com/stretchr/testify/require"

	restful "github.com/emicklei/go-restful"

	nbmachine "git.f-i-ts.de/cloud-native/metal/metal-api/netbox-api/client/machines"
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

func TestGetMachines(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	pub := &emptyPublisher{}
	nb := netbox.New()
	dservice := NewMachine(ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	req := httptest.NewRequest("GET", "/v1/machine", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result []metal.Machine
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Len(t, result, len(testdata.TestMachines))
	require.Equal(t, testdata.M1.ID, result[0].ID)
	require.Equal(t, testdata.M1.Allocation.Name, result[0].Allocation.Name)
	require.Equal(t, testdata.Sz1.Name, result[0].Size.Name)
	require.Equal(t, testdata.Partition1.Name, result[0].Partition.Name)
	require.Equal(t, testdata.M2.ID, result[1].ID)
}

func TestRegisterMachine(t *testing.T) {
	ipmi := metal.IPMI{
		Address:    "address",
		Interface:  "interface",
		MacAddress: "mac",
	}
	data := []struct {
		name               string
		uuid               string
		partitionid        string
		numcores           int
		memory             int
		dbpartitions       []metal.Partition
		dbsizes            []metal.Size
		dbmachines         []metal.Machine
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
			partitionid:        "1",
			dbpartitions:       []metal.Partition{testdata.Partition1},
			dbsizes:            []metal.Size{testdata.Sz1},
			numcores:           1,
			memory:             100,
			expectedStatus:     http.StatusOK,
			expectedIPMIStatus: http.StatusOK,
			expectedSizeName:   testdata.Sz1.Name,
			ipmiresult:         []metal.IPMI{ipmi},
			ipmiresulterror:    nil,
		},
		{
			name:               "no ipmi data",
			uuid:               "1",
			partitionid:        "1",
			dbpartitions:       []metal.Partition{testdata.Partition1},
			dbsizes:            []metal.Size{testdata.Sz1},
			numcores:           1,
			memory:             100,
			expectedStatus:     http.StatusOK,
			expectedIPMIStatus: http.StatusNotFound,
			expectedSizeName:   testdata.Sz1.Name,
			ipmiresult:         []metal.IPMI{},
			ipmiresulterror:    nil,
		},
		{
			name:               "ipmi fetch error",
			uuid:               "1",
			partitionid:        "1",
			dbpartitions:       []metal.Partition{testdata.Partition1},
			dbsizes:            []metal.Size{testdata.Sz1},
			numcores:           1,
			memory:             100,
			expectedStatus:     http.StatusOK,
			expectedIPMIStatus: http.StatusUnprocessableEntity,
			expectedSizeName:   testdata.Sz1.Name,

			ipmiresult:      []metal.IPMI{},
			ipmiresulterror: fmt.Errorf("Test Error"),
		},
		{
			name:               "insert existing",
			uuid:               "1",
			partitionid:        "1",
			dbpartitions:       []metal.Partition{testdata.Partition1},
			dbsizes:            []metal.Size{testdata.Sz1},
			dbmachines:         []metal.Machine{testdata.M1},
			numcores:           1,
			memory:             100,
			expectedStatus:     http.StatusOK,
			expectedIPMIStatus: http.StatusOK,
			expectedSizeName:   testdata.Sz1.Name,
			ipmiresult:         []metal.IPMI{ipmi},
			ipmiresulterror:    nil,
		},
		{
			name:           "empty uuid",
			uuid:           "",
			partitionid:    "1",
			dbpartitions:   []metal.Partition{testdata.Partition1},
			dbsizes:        []metal.Size{testdata.Sz1},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:           "error when impi update fails",
			uuid:           "1",
			partitionid:    "1",
			dbpartitions:   []metal.Partition{testdata.Partition1},
			dbsizes:        []metal.Size{testdata.Sz1},
			ipmidberror:    fmt.Errorf("ipmi insert fails"),
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:           "empty partition",
			uuid:           "1",
			partitionid:    "",
			dbpartitions:   nil,
			dbsizes:        []metal.Size{testdata.Sz1},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:               "unknown size because wrong cpu",
			uuid:               "1",
			partitionid:        "1",
			dbpartitions:       []metal.Partition{testdata.Partition1},
			dbsizes:            []metal.Size{testdata.Sz1},
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
			partitionid:    "1",
			dbpartitions:   []metal.Partition{testdata.Partition1},
			dbsizes:        []metal.Size{testdata.Sz1},
			numcores:       2,
			memory:         100,
			netboxerror:    fmt.Errorf("this should happen"),
			expectedStatus: http.StatusUnprocessableEntity,
		},
	}
	for _, test := range data {
		t.Run(test.name, func(t *testing.T) {
			ds, mock := datastore.InitMockDB()
			mock.On(r.DB("mockdb").Table("ipmi").Insert(r.MockAnything(), r.InsertOpts{
				Conflict: "replace",
			})).Return(testdata.EmptyResult, test.ipmidberror)

			rr := metal.RegisterMachine{
				UUID:        test.uuid,
				PartitionID: test.partitionid,
				RackID:      "1",
				IPMI:        ipmi,
				Hardware: metal.MachineHardware{
					CPUCores: test.numcores,
					Memory:   uint64(test.memory),
				},
			}
			mock.On(r.DB("mockdb").Table("partition").Get(test.partitionid)).Return(test.dbpartitions, nil)

			if len(test.dbmachines) > 0 {
				mock.On(r.DB("mockdb").Table("size").Get(test.dbmachines[0].SizeID)).Return([]metal.Size{testdata.Sz1}, nil)
				mock.On(r.DB("mockdb").Table("machine").Get(test.dbmachines[0].ID).Replace(r.MockAnything())).Return(testdata.EmptyResult, nil)
			} else {
				mock.On(r.DB("mockdb").Table("machine").Insert(r.MockAnything(), r.InsertOpts{
					Conflict: "replace",
				})).Return(testdata.EmptyResult, nil)
			}
			mock.On(r.DB("mockdb").Table("ipmi").Get(test.uuid)).Return(test.ipmiresult, test.ipmiresulterror)
			testdata.InitMockDBData(mock)
			mock.On(r.DB("mockdb").Table("switch").Filter(r.MockAnything(), r.FilterOpts{})).Return([]metal.Switch{}, nil)

			pub := &emptyPublisher{}
			nb := netbox.New()
			called := false
			nb.DoRegister = func(params *nbmachine.NetboxAPIProxyAPIMachineRegisterParams, authInfo runtime.ClientAuthInfoWriter) (*nbmachine.NetboxAPIProxyAPIMachineRegisterOK, error) {
				called = true
				return &nbmachine.NetboxAPIProxyAPIMachineRegisterOK{}, test.netboxerror
			}
			js, _ := json.Marshal(rr)
			body := bytes.NewBuffer(js)

			dservice := NewMachine(ds, pub, nb)
			container := restful.NewContainer().Add(dservice)
			req := httptest.NewRequest("POST", "/v1/machine/register", body)
			req.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()
			container.ServeHTTP(w, req)

			resp := w.Result()
			require.Equal(t, test.expectedStatus, resp.StatusCode, w.Body.String())
			if resp.StatusCode >= 300 {
				return
			}
			var result metal.Machine

			err := json.NewDecoder(resp.Body).Decode(&result)
			require.Nil(t, err)
			require.True(t, called, "netbox register was not called")
			expectedid := testdata.M1.ID
			if len(test.dbmachines) > 0 {
				expectedid = test.dbmachines[0].ID
			}
			require.Equal(t, expectedid, result.ID)
			require.Equal(t, test.expectedSizeName, result.Size.Name)
			require.Equal(t, testdata.Partition1.Name, result.Partition.Name)
			// no read ipmi data
			req = httptest.NewRequest("GET", fmt.Sprintf("/v1/machine/%s/ipmi", test.uuid), nil)
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

func TestReportMachine(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	pub := &emptyPublisher{}
	nb := netbox.New()
	dservice := NewMachine(ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	rep := metal.ReportAllocation{
		Success:         true,
		ConsolePassword: "blubber",
	}
	js, _ := json.Marshal(rep)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/machine/1/report", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result metal.MachineAllocation
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, result.ConsolePassword, rep.ConsolePassword)
}

func TestReportFailureMachine(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	pub := &emptyPublisher{}
	nb := netbox.New()
	dservice := NewMachine(ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	rep := metal.ReportAllocation{
		Success:         false,
		ErrorMessage:    "my error message",
		ConsolePassword: "blubber",
	}
	js, _ := json.Marshal(rep)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/machine/1/report", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result metal.MachineAllocation
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
}

func TestReportUnknownMachine(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	pub := &emptyPublisher{}
	nb := netbox.New()
	dservice := NewMachine(ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	rep := metal.ReportAllocation{
		Success:         false,
		ErrorMessage:    "my error message",
		ConsolePassword: "blubber",
	}
	js, _ := json.Marshal(rep)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/machine/999/report", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusNotFound, resp.StatusCode, w.Body.String())
}

func TestReportUnknownFailure(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	pub := &emptyPublisher{}
	nb := netbox.New()
	dservice := NewMachine(ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	rep := metal.ReportAllocation{
		Success:         false,
		ErrorMessage:    "my error message",
		ConsolePassword: "blubber",
	}
	js, _ := json.Marshal(rep)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/machine/404/report", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode, w.Body.String())
}

func TestReportUnallocatedMachine(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	pub := &emptyPublisher{}
	nb := netbox.New()
	dservice := NewMachine(ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	rep := metal.ReportAllocation{
		Success:         true,
		ErrorMessage:    "",
		ConsolePassword: "blubber",
	}
	js, _ := json.Marshal(rep)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/machine/3/report", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode, w.Body.String())
}

func TestGetMachine(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)
	pub := &emptyPublisher{}
	nb := netbox.New()
	dservice := NewMachine(ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	req := httptest.NewRequest("GET", "/v1/machine/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result metal.Machine
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.Nil(t, err)
	require.Equal(t, testdata.M1.ID, result.ID)
	require.Equal(t, testdata.M1.Allocation.Name, result.Allocation.Name)
	require.Equal(t, testdata.Sz1.Name, result.Size.Name)
	require.Equal(t, testdata.Img1.Name, result.Allocation.Image.Name)
	require.Equal(t, testdata.Partition1.Name, result.Partition.Name)
}

func TestGetMachineNotFound(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	pub := &emptyPublisher{}
	nb := netbox.New()
	dservice := NewMachine(ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	req := httptest.NewRequest("GET", "/v1/machine/999", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusNotFound, resp.StatusCode, w.Body.String())
}
func TestFreeMachine(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	pub := &emptyPublisher{}
	pub.doPublish = func(topic string, data interface{}) error {
		require.Equal(t, "machine", topic)
		dv := data.(metal.MachineEvent)
		require.Equal(t, "1", dv.Old.ID)
		return nil
	}
	nb := netbox.New()
	called := false
	nb.DoRelease = func(params *nbmachine.NetboxAPIProxyAPIMachineReleaseParams, authInfo runtime.ClientAuthInfoWriter) (*nbmachine.NetboxAPIProxyAPIMachineReleaseOK, error) {
		called = true
		return &nbmachine.NetboxAPIProxyAPIMachineReleaseOK{}, nil
	}
	dservice := NewMachine(ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	req := httptest.NewRequest("DELETE", "/v1/machine/1/free", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	require.True(t, called, "netbox.DoRelease was not called")
}

func TestSearchMachine(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	mock.On(r.DB("mockdb").Table("machine").Filter(r.MockAnything())).Return([]interface{}{testdata.M1}, nil)
	testdata.InitMockDBData(mock)

	pub := &emptyPublisher{}
	nb := netbox.New()
	dservice := NewMachine(ds, pub, nb)
	container := restful.NewContainer().Add(dservice)
	req := httptest.NewRequest("GET", "/v1/machine/find?mac=1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var results []metal.Machine
	err := json.NewDecoder(resp.Body).Decode(&results)
	require.Nil(t, err)
	require.Len(t, results, 1)
	result := results[0]
	require.Equal(t, testdata.M1.ID, result.ID)
	require.Equal(t, testdata.M1.Allocation.Name, result.Allocation.Name)
	require.Equal(t, testdata.Sz1.Name, result.Size.Name)
	require.Equal(t, testdata.Img1.Name, result.Allocation.Image.Name)
	require.Equal(t, testdata.Partition1.Name, result.Partition.Name)
}

func TestOnMachine(t *testing.T) {

	data := []struct {
		cmd      metal.MachineCommand
		endpoint string
		param    string
	}{
		{
			cmd:      metal.MachineOnCmd,
			endpoint: "on",
			param:    "blub",
		},
		{
			cmd:      metal.MachineOffCmd,
			endpoint: "off",
			param:    "blubber",
		},
		{
			cmd:      metal.MachineResetCmd,
			endpoint: "reset",
			param:    "bluba",
		},
	}

	for _, d := range data {
		t.Run("cmd_"+d.endpoint, func(t *testing.T) {
			ds, mock := datastore.InitMockDB()
			testdata.InitMockDBData(mock)
			pub := &emptyPublisher{}
			pub.doPublish = func(topic string, data interface{}) error {
				require.Equal(t, "machine", topic)
				dv := data.(metal.MachineEvent)
				require.Equal(t, d.cmd, dv.Cmd.Command)
				require.Equal(t, d.param, dv.Cmd.Params[0])
				require.Equal(t, "1", dv.Cmd.Target.ID)
				return nil
			}
			js, _ := json.Marshal([]string{d.param})
			body := bytes.NewBuffer(js)
			nb := netbox.New()
			dservice := NewMachine(ds, pub, nb)
			container := restful.NewContainer().Add(dservice)
			req := httptest.NewRequest("POST", "/v1/machine/1/"+d.endpoint, body)
			req.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()
			container.ServeHTTP(w, req)

			resp := w.Result()
			require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
		})
	}
}
