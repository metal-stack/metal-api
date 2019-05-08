package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/ipam"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	v1 "git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/service/v1"
	"git.f-i-ts.de/cloud-native/metallib/httperrors"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/testdata"
	goipam "github.com/metal-pod/go-ipam"
	"github.com/stretchr/testify/require"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"

	"github.com/emicklei/go-restful"
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

	machineservice := NewMachine(ds, &emptyPublisher{}, ipam.New(goipam.New()))
	container := restful.NewContainer().Add(machineservice)
	req := httptest.NewRequest("GET", "/v1/machine", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result []v1.MachineListResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Len(t, result, len(testdata.TestMachines))
	require.Equal(t, testdata.M1.ID, result[0].ID)
	require.Equal(t, testdata.M1.Allocation.Name, result[0].Allocation.Name)
	require.Equal(t, testdata.Sz1.Name, *result[0].Size.Name)
	require.Equal(t, testdata.Partition1.Name, *result[0].Partition.Name)
	require.Equal(t, testdata.M2.ID, result[1].ID)
}

func TestRegisterMachine(t *testing.T) {
	data := []struct {
		name                 string
		uuid                 string
		partitionid          string
		numcores             int
		memory               int
		dbpartitions         []metal.Partition
		dbsizes              []metal.Size
		dbmachines           []metal.Machine
		expectedStatus       int
		expectedErrorMessage string
		expectedSizeName     string
	}{
		{
			name:             "insert new",
			uuid:             "0",
			partitionid:      "0",
			dbpartitions:     []metal.Partition{testdata.Partition1},
			dbsizes:          []metal.Size{testdata.Sz1},
			numcores:         1,
			memory:           100,
			expectedStatus:   http.StatusOK,
			expectedSizeName: testdata.Sz1.Name,
		},
		{
			name:             "insert existing",
			uuid:             "1",
			partitionid:      "1",
			dbpartitions:     []metal.Partition{testdata.Partition1},
			dbsizes:          []metal.Size{testdata.Sz1},
			dbmachines:       []metal.Machine{testdata.M1},
			numcores:         1,
			memory:           100,
			expectedStatus:   http.StatusOK,
			expectedSizeName: testdata.Sz1.Name,
		},
		{
			name:                 "empty uuid",
			uuid:                 "",
			partitionid:          "1",
			dbpartitions:         []metal.Partition{testdata.Partition1},
			dbsizes:              []metal.Size{testdata.Sz1},
			expectedStatus:       http.StatusUnprocessableEntity,
			expectedErrorMessage: "uuid cannot be empty",
		},
		{
			name:                 "empty partition",
			uuid:                 "1",
			partitionid:          "",
			dbpartitions:         nil,
			dbsizes:              []metal.Size{testdata.Sz1},
			expectedStatus:       http.StatusNotFound,
			expectedErrorMessage: "no partition with id \"\" found",
		},
		{
			name:             "unknown size because wrong cpu",
			uuid:             "0",
			partitionid:      "1",
			dbpartitions:     []metal.Partition{testdata.Partition1},
			dbsizes:          []metal.Size{testdata.Sz1},
			numcores:         2,
			memory:           100,
			expectedStatus:   http.StatusOK,
			expectedSizeName: metal.UnknownSize.Name,
		},
	}

	for _, test := range data {
		t.Run(test.name, func(t *testing.T) {
			ds, mock := datastore.InitMockDB()
			mock.On(r.DB("mockdb").Table("partition").Get(test.partitionid)).Return(test.dbpartitions, nil)

			if len(test.dbmachines) > 0 {
				mock.On(r.DB("mockdb").Table("size").Get(test.dbmachines[0].SizeID)).Return([]metal.Size{testdata.Sz1}, nil)
				mock.On(r.DB("mockdb").Table("machine").Get(test.dbmachines[0].ID).Replace(r.MockAnything())).Return(testdata.EmptyResult, nil)
			} else {
				mock.On(r.DB("mockdb").Table("machine").Insert(r.MockAnything(), r.InsertOpts{
					Conflict: "replace",
				})).Return(testdata.EmptyResult, nil)
			}
			mock.On(r.DB("mockdb").Table("size").Get(metal.UnknownSize.ID)).Return([]metal.Size{*metal.UnknownSize}, nil)
			mock.On(r.DB("mockdb").Table("switch").Filter(r.MockAnything(), r.FilterOpts{})).Return([]metal.Switch{}, nil)
			mock.On(r.DB("mockdb").Table("event").Filter(r.MockAnything(), r.FilterOpts{})).Return([]metal.ProvisioningEventContainer{}, nil)
			mock.On(r.DB("mockdb").Table("event").Insert(r.MockAnything(), r.InsertOpts{})).Return(testdata.EmptyResult, nil)
			testdata.InitMockDBData(mock)

			registerRequest := &v1.MachineRegisterRequest{
				UUID:        test.uuid,
				PartitionID: test.partitionid,
				RackID:      "1",
				IPMI: v1.MachineIPMI{
					Address:    testdata.IPMI1.Address,
					Interface:  testdata.IPMI1.Interface,
					MacAddress: testdata.IPMI1.MacAddress,
					Fru: v1.MachineFru{
						ChassisPartNumber:   &testdata.IPMI1.Fru.ChassisPartNumber,
						ChassisPartSerial:   &testdata.IPMI1.Fru.ChassisPartSerial,
						BoardMfg:            &testdata.IPMI1.Fru.BoardMfg,
						BoardMfgSerial:      &testdata.IPMI1.Fru.BoardMfgSerial,
						BoardPartNumber:     &testdata.IPMI1.Fru.BoardPartNumber,
						ProductManufacturer: &testdata.IPMI1.Fru.ProductManufacturer,
						ProductPartNumber:   &testdata.IPMI1.Fru.ProductPartNumber,
						ProductSerial:       &testdata.IPMI1.Fru.ProductSerial,
					},
				},
				Hardware: v1.MachineHardware{
					CPUCores: test.numcores,
					Memory:   uint64(test.memory),
				},
			}

			js, _ := json.Marshal(registerRequest)
			body := bytes.NewBuffer(js)
			machineservice := NewMachine(ds, &emptyPublisher{}, ipam.New(goipam.New()))
			container := restful.NewContainer().Add(machineservice)
			req := httptest.NewRequest("POST", "/v1/machine/register", body)
			req.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()
			container.ServeHTTP(w, req)

			resp := w.Result()
			require.Equal(t, test.expectedStatus, resp.StatusCode, w.Body.String())

			if test.expectedStatus > 300 {
				var result httperrors.HTTPErrorResponse
				err := json.NewDecoder(resp.Body).Decode(&result)

				require.Nil(t, err)
				require.Regexp(t, test.expectedErrorMessage, result.Message)
			} else {
				var result v1.MachineDetailResponse
				err := json.NewDecoder(resp.Body).Decode(&result)

				require.Nil(t, err)
				expectedid := "0"
				if len(test.dbmachines) > 0 {
					expectedid = test.dbmachines[0].ID
				}
				require.Equal(t, expectedid, result.ID)
				require.Equal(t, "1", result.RackID)
				require.Equal(t, test.expectedSizeName, *result.Size.Name)
				require.Equal(t, testdata.Partition1.Name, *result.Partition.Name)
			}
		})
	}
}

// TODO: Tests for reading IPMI
// 	// now read ipmi data

// 	ds, mock = datastore.InitMockDB()
// 	if len(test.dbmachines) > 0 {
// 		mock.On(r.DB("mockdb").Table("machine").Get(test.dbmachines[0].ID).Replace(r.MockAnything())).Return(testdata.EmptyResult, nil)
// 	} else {
// 		mock.On(r.DB("mockdb").Table("machine").Get("0").Replace(r.MockAnything())).Return(testdata.EmptyResult, nil)
// 	}

// 	testdata.InitMockDBData(mock)

// 	req = httptest.NewRequest("GET", fmt.Sprintf("/v1/machine/%s/ipmi", test.uuid), nil)
// 	w = httptest.NewRecorder()
// 	container.ServeHTTP(w, req)

// 	resp = w.Result()
// 	require.Equal(t, test.expectedIPMIStatus, resp.StatusCode, w.Body.String())
// 	// if resp.StatusCode >= 300 {
// 	// 	return
// 	// }
// 	var ipmiresult v1.MachineIPMIResponse
// 	err = json.NewDecoder(resp.Body).Decode(&ipmiresult)
// 	require.Nil(t, err)

// 	require.Equal(t, testdata.IPMI1.Address, ipmiresult.Address)
// 	require.Equal(t, testdata.IPMI1.Interface, ipmiresult.Interface)
// 	require.Equal(t, testdata.IPMI1.User, ipmiresult.User)
// 	require.Equal(t, testdata.IPMI1.Password, ipmiresult.Password)
// 	require.Equal(t, testdata.IPMI1.MacAddress, ipmiresult.MacAddress)

// 	require.Equal(t, testdata.IPMI1.Fru.ChassisPartNumber, *ipmiresult.Fru.ChassisPartNumber)
// 	require.Equal(t, testdata.IPMI1.Fru.ChassisPartSerial, *ipmiresult.Fru.ChassisPartSerial)
// 	require.Equal(t, testdata.IPMI1.Fru.BoardMfg, *ipmiresult.Fru.BoardMfg)
// 	require.Equal(t, testdata.IPMI1.Fru.BoardMfgSerial, *ipmiresult.Fru.BoardMfgSerial)
// 	require.Equal(t, testdata.IPMI1.Fru.BoardPartNumber, *ipmiresult.Fru.BoardPartNumber)
// 	require.Equal(t, testdata.IPMI1.Fru.ProductManufacturer, *ipmiresult.Fru.ProductManufacturer)
// 	require.Equal(t, testdata.IPMI1.Fru.ProductPartNumber, *ipmiresult.Fru.ProductPartNumber)
// 	require.Equal(t, testdata.IPMI1.Fru.ProductSerial, *ipmiresult.Fru.ProductSerial)
// })

// func TestReportMachine(t *testing.T) {
// 	ds, mock := datastore.InitMockDB()
// 	testdata.InitMockDBData(mock)

// 	pub := &emptyPublisher{}
// 	ip := goipam.New()
// 	ipamer := ipam.New(ip)
// 	dservice := NewMachine(ds, pub, ipamer)
// 	container := restful.NewContainer().Add(dservice)
// 	rep := metal.ReportAllocation{
// 		Success:         true,
// 		ConsolePassword: "blubber",
// 	}
// 	js, _ := json.Marshal(rep)
// 	body := bytes.NewBuffer(js)
// 	req := httptest.NewRequest("POST", "/v1/machine/1/report", body)
// 	req.Header.Add("Content-Type", "application/json")
// 	w := httptest.NewRecorder()
// 	container.ServeHTTP(w, req)

// 	resp := w.Result()
// 	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
// 	var result metal.MachineAllocation
// 	err := json.NewDecoder(resp.Body).Decode(&result)
// 	require.Nil(t, err)
// 	require.Equal(t, result.ConsolePassword, rep.ConsolePassword)
// }

// func TestReportFailureMachine(t *testing.T) {
// 	ds, mock := datastore.InitMockDB()
// 	testdata.InitMockDBData(mock)

// 	pub := &emptyPublisher{}
// 	ip := goipam.New()
// 	ipamer := ipam.New(ip)
// 	dservice := NewMachine(ds, pub, ipamer)
// 	container := restful.NewContainer().Add(dservice)
// 	rep := metal.ReportAllocation{
// 		Success:         false,
// 		ErrorMessage:    "my error message",
// 		ConsolePassword: "blubber",
// 	}
// 	js, _ := json.Marshal(rep)
// 	body := bytes.NewBuffer(js)
// 	req := httptest.NewRequest("POST", "/v1/machine/1/report", body)
// 	req.Header.Add("Content-Type", "application/json")
// 	w := httptest.NewRecorder()
// 	container.ServeHTTP(w, req)

// 	resp := w.Result()
// 	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
// 	var result metal.MachineAllocation
// 	err := json.NewDecoder(resp.Body).Decode(&result)
// 	require.Nil(t, err)
// }

// func TestReportUnknownMachine(t *testing.T) {
// 	ds, mock := datastore.InitMockDB()
// 	testdata.InitMockDBData(mock)

// 	pub := &emptyPublisher{}
// 	ip := goipam.New()
// 	ipamer := ipam.New(ip)
// 	dservice := NewMachine(ds, pub, ipamer)
// 	container := restful.NewContainer().Add(dservice)
// 	rep := metal.ReportAllocation{
// 		Success:         false,
// 		ErrorMessage:    "my error message",
// 		ConsolePassword: "blubber",
// 	}
// 	js, _ := json.Marshal(rep)
// 	body := bytes.NewBuffer(js)
// 	req := httptest.NewRequest("POST", "/v1/machine/999/report", body)
// 	req.Header.Add("Content-Type", "application/json")
// 	w := httptest.NewRecorder()
// 	container.ServeHTTP(w, req)

// 	resp := w.Result()
// 	require.Equal(t, http.StatusNotFound, resp.StatusCode, w.Body.String())
// }

func TestSetMachineState(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	machineservice := NewMachine(ds, &emptyPublisher{}, ipam.New(goipam.New()))
	container := restful.NewContainer().Add(machineservice)

	stateRequest := v1.MachineState{
		Value:       string(metal.ReservedState),
		Description: "blubber",
	}
	js, _ := json.Marshal(stateRequest)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/machine/1/state", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.MachineDetailResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, "1", result.ID)
	require.Equal(t, string(metal.ReservedState), result.State.Value)
	require.Equal(t, "blubber", result.State.Description)

}

// func TestReportUnknownFailure(t *testing.T) {
// 	ds, mock := datastore.InitMockDB()
// 	testdata.InitMockDBData(mock)

// 	pub := &emptyPublisher{}
// 	ip := goipam.New()
// 	ipamer := ipam.New(ip)
// 	dservice := NewMachine(ds, pub, ipamer)
// 	container := restful.NewContainer().Add(dservice)
// 	rep := metal.ReportAllocation{
// 		Success:         false,
// 		ErrorMessage:    "my error message",
// 		ConsolePassword: "blubber",
// 	}
// 	js, _ := json.Marshal(rep)
// 	body := bytes.NewBuffer(js)
// 	req := httptest.NewRequest("POST", "/v1/machine/404/report", body)
// 	req.Header.Add("Content-Type", "application/json")
// 	w := httptest.NewRecorder()
// 	container.ServeHTTP(w, req)

// 	resp := w.Result()
// 	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode, w.Body.String())
// }

// func TestReportUnallocatedMachine(t *testing.T) {
// 	ds, mock := datastore.InitMockDB()
// 	testdata.InitMockDBData(mock)

// 	pub := &emptyPublisher{}
// 	ip := goipam.New()
// 	ipamer := ipam.New(ip)
// 	dservice := NewMachine(ds, pub, ipamer)
// 	container := restful.NewContainer().Add(dservice)
// 	rep := metal.ReportAllocation{
// 		Success:         true,
// 		ErrorMessage:    "",
// 		ConsolePassword: "blubber",
// 	}
// 	js, _ := json.Marshal(rep)
// 	body := bytes.NewBuffer(js)
// 	req := httptest.NewRequest("POST", "/v1/machine/3/report", body)
// 	req.Header.Add("Content-Type", "application/json")
// 	w := httptest.NewRecorder()
// 	container.ServeHTTP(w, req)

// 	resp := w.Result()
// 	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode, w.Body.String())
// }

func TestGetMachine(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	machineservice := NewMachine(ds, &emptyPublisher{}, ipam.New(goipam.New()))
	container := restful.NewContainer().Add(machineservice)
	req := httptest.NewRequest("GET", "/v1/machine/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.MachineDetailResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, testdata.M1.ID, result.ID)
	require.Equal(t, testdata.M1.Allocation.Name, result.Allocation.Name)
	require.Equal(t, testdata.Sz1.Name, *result.Size.Name)
	require.Equal(t, testdata.Img1.Name, *result.Allocation.Image.Name)
	require.Equal(t, testdata.Partition1.Name, *result.Partition.Name)
}

func TestGetMachineNotFound(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	machineservice := NewMachine(ds, &emptyPublisher{}, ipam.New(goipam.New()))
	container := restful.NewContainer().Add(machineservice)
	req := httptest.NewRequest("GET", "/v1/machine/999", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusNotFound, resp.StatusCode, w.Body.String())
}

// func TestFreeMachine(t *testing.T) {
// 	ds, mock := datastore.InitMockDB()
// 	testdata.InitMockDBData(mock)

// 	pub := &emptyPublisher{}
// 	events := []string{"machine", "switch"}
// 	eventidx := 0
// 	pub.doPublish = func(topic string, data interface{}) error {
// 		require.Equal(t, events[eventidx], topic)
// 		eventidx++
// 		if eventidx == 0 {
// 			dv := data.(metal.MachineEvent)
// 			require.Equal(t, "1", dv.Old.ID)
// 		}
// 		return nil
// 	}
// 	ip := goipam.New()
// 	ipamer := ipam.New(ip)
// 	dservice := NewMachine(ds, pub, ipamer)
// 	container := restful.NewContainer().Add(dservice)
// 	req := httptest.NewRequest("DELETE", "/v1/machine/1/free", nil)
// 	w := httptest.NewRecorder()
// 	container.ServeHTTP(w, req)

// 	resp := w.Result()
// 	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
// }

func TestSearchMachine(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	mock.On(r.DB("mockdb").Table("machine").Filter(r.MockAnything())).Return([]interface{}{testdata.M1}, nil)
	testdata.InitMockDBData(mock)

	machineservice := NewMachine(ds, &emptyPublisher{}, ipam.New(goipam.New()))
	container := restful.NewContainer().Add(machineservice)
	req := httptest.NewRequest("GET", "/v1/machine/find?mac=1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var results []v1.MachineListResponse
	err := json.NewDecoder(resp.Body).Decode(&results)

	require.Nil(t, err)
	require.Len(t, results, 1)
	result := results[0]
	require.Equal(t, testdata.M1.ID, result.ID)
	require.Equal(t, testdata.M1.Allocation.Name, result.Allocation.Name)
	require.Equal(t, testdata.Sz1.Name, *result.Size.Name)
	require.Equal(t, testdata.Img1.Name, *result.Allocation.Image.Name)
	require.Equal(t, testdata.Partition1.Name, *result.Partition.Name)
}

func TestAddProvisioningEvent(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	machineservice := NewMachine(ds, &emptyPublisher{}, ipam.New(goipam.New()))
	container := restful.NewContainer().Add(machineservice)
	event := &metal.ProvisioningEvent{
		Event:   metal.ProvisioningEventPreparing,
		Message: "starting metal-hammer",
	}
	js, _ := json.Marshal(event)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/machine/1/event", body)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.MachineRecentProvisioningEvents
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, "0", *result.IncompleteProvisioningCycles)
	require.Len(t, result.Events, 1)
	if len(result.Events) > 0 {
		require.Equal(t, "starting metal-hammer", result.Events[0].Message)
		require.Equal(t, string(metal.ProvisioningEventPreparing), result.Events[0].Event)
	}
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
		{
			cmd:      metal.MachineBiosCmd,
			endpoint: "bios",
			param:    "blubabla",
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

			machineservice := NewMachine(ds, pub, ipam.New(goipam.New()))

			js, _ := json.Marshal([]string{d.param})
			body := bytes.NewBuffer(js)
			container := restful.NewContainer().Add(machineservice)
			req := httptest.NewRequest("POST", "/v1/machine/1/power/"+d.endpoint, body)
			req.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()
			container.ServeHTTP(w, req)

			resp := w.Result()
			require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
		})
	}
}
