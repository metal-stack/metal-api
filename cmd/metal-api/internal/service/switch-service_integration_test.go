//go:build integration
// +build integration

package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emicklei/go-restful/v3"
	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	mdmv1mock "github.com/metal-stack/masterdata-api/api/v1/mocks"
	mdm "github.com/metal-stack/masterdata-api/pkg/client"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/ipam"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-api/test"
	"github.com/metal-stack/metal-lib/bus"
	"github.com/metal-stack/security"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

func TestSwitchReplacementIntegration(t *testing.T) {
	ts := createTestService(t)
	defer ts.terminate()

	ts.createPartition("test-partition", "Test Partition")

	// register switches
	var res v1.SwitchResponse
	srr := v1.SwitchRegisterRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: "test-switch01",
			},
		},
		Nics: []v1.SwitchNic{
			{
				MacAddress: "aa:aa:aa:aa:aa:aa",
				Name:       "swp1",
			},
		},
		PartitionID: "test-partition",
		SwitchBase: v1.SwitchBase{
			RackID: "test-rack",
			OS: &v1.SwitchOS{
				Vendor: metal.SwitchOSVendorCumulus,
			},
		},
	}

	status := ts.switchRegister(t, srr, &res)
	require.Equal(t, http.StatusCreated, status)
	require.NotNil(t, res)
	require.Equal(t, srr.ID, res.ID)
	require.Len(t, res.Nics, 1)
	require.Equal(t, srr.Nics[0].Name, res.Nics[0].Name)
	require.Equal(t, srr.Nics[0].MacAddress, res.Nics[0].MacAddress)

	srr = v1.SwitchRegisterRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: "test-switch02",
			},
		},
		Nics: []v1.SwitchNic{
			{
				MacAddress: "bb:bb:bb:bb:bb:bb",
				Name:       "swp1",
			},
		},
		PartitionID: "test-partition",
		SwitchBase: v1.SwitchBase{
			RackID: "test-rack",
			OS: &v1.SwitchOS{
				Vendor: metal.SwitchOSVendorCumulus,
			},
		},
	}

	status = ts.switchRegister(t, srr, &res)
	require.Equal(t, http.StatusCreated, status)
	require.NotNil(t, res)
	require.Equal(t, srr.ID, res.ID)
	require.Len(t, res.Nics, 1)
	require.Equal(t, srr.Nics[0].Name, res.Nics[0].Name)
	require.Equal(t, srr.Nics[0].MacAddress, res.Nics[0].MacAddress)

	// create machine
	m := metal.Machine{
		Base: metal.Base{
			ID: "test-machine",
		},
		PartitionID: "test-partition",
		RackID:      "test-rack",
		Hardware: metal.MachineHardware{
			Nics: []metal.Nic{
				{
					Name:       "eth0",
					MacAddress: "11:11:11:11:11:11",
					Neighbors: []metal.Nic{
						{
							Name:       "swp1",
							MacAddress: "aa:aa:aa:aa:aa:aa",
						},
					},
				},
				{
					Name:       "eth1",
					MacAddress: "22:22:22:22:22:22",
					Neighbors: []metal.Nic{
						{
							Name:       "swp1",
							MacAddress: "bb:bb:bb:bb:bb:bb",
						},
					},
				},
			},
		},
	}

	err := ts.ds.CreateMachine(&m)
	require.NoError(t, err)
	err = ts.ds.ConnectMachineWithSwitches(&m)
	require.NoError(t, err)

	err = ts.ds.CreateProvisioningEventContainer(&metal.ProvisioningEventContainer{
		Base:       metal.Base{ID: m.ID},
		Liveliness: metal.MachineLivelinessAlive,
	})
	require.NoError(t, err)

	// replace first switch
	sur := v1.SwitchUpdateRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: "test-switch01",
			},
		},
		SwitchBase: v1.SwitchBase{
			Mode: string(metal.SwitchReplace),
		},
	}

	status = ts.switchUpdate(t, sur, &res)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, string(metal.SwitchReplace), res.SwitchBase.Mode)

	srr = v1.SwitchRegisterRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: "test-switch01",
			},
		},
		Nics: []v1.SwitchNic{
			{
				MacAddress: "dd:dd:dd:dd:dd:dd",
				Name:       "Ethernet4",
			},
		},
		PartitionID: "test-partition",
		SwitchBase: v1.SwitchBase{
			RackID: "test-rack",
			OS:     &v1.SwitchOS{Vendor: metal.SwitchOSVendorSonic},
		},
	}

	status = ts.switchRegister(t, srr, &res)
	require.Equal(t, http.StatusOK, status)

	status = ts.switchGet(t, "test-switch01", &res)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, res.Nics, 1)
	require.Equal(t, srr.Nics[0].Name, res.Nics[0].Name)
	require.Equal(t, srr.Nics[0].MacAddress, res.Nics[0].MacAddress)
	require.Equal(t, string(metal.SwitchOperational), res.Mode)
	require.Len(t, res.Connections, 1)
	require.Equal(t, "test-machine", res.Connections[0].MachineID)
	require.Equal(t, "Ethernet4", res.Connections[0].Nic.Name)
	require.Equal(t, "dd:dd:dd:dd:dd:dd", res.Connections[0].Nic.MacAddress)

	var mres v1.MachineResponse
	status = ts.machineGet(t, m.ID, &mres)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, mres.Hardware.Nics, 2)

	var nic v1.MachineNic
	for _, n := range mres.Hardware.Nics {
		if n.Name == "eth0" {
			nic = n
		}
	}
	require.Equal(t, "eth0", nic.Name)
	require.Len(t, nic.Neighbors, 1)
	require.Equal(t, "dd:dd:dd:dd:dd:dd", nic.Neighbors[0].MacAddress)
	require.Equal(t, "Ethernet4", nic.Neighbors[0].Name)

	// replace second switch
	sur = v1.SwitchUpdateRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: "test-switch02",
			},
		},
		SwitchBase: v1.SwitchBase{
			Mode: string(metal.SwitchReplace),
		},
	}

	status = ts.switchUpdate(t, sur, &res)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, string(metal.SwitchReplace), res.SwitchBase.Mode)

	srr = v1.SwitchRegisterRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: "test-switch02",
			},
		},
		Nics: []v1.SwitchNic{
			{
				MacAddress: "cc:cc:cc:cc:cc:cc",
				Name:       "Ethernet4",
			},
		},
		PartitionID: "test-partition",
		SwitchBase: v1.SwitchBase{
			RackID: "test-rack",
			OS:     &v1.SwitchOS{Vendor: metal.SwitchOSVendorSonic},
		},
	}

	status = ts.switchRegister(t, srr, &res)
	require.Equal(t, http.StatusOK, status)

	status = ts.switchGet(t, "test-switch02", &res)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, res.Nics, 1)
	require.Equal(t, srr.Nics[0].Name, res.Nics[0].Name)
	require.Equal(t, srr.Nics[0].MacAddress, res.Nics[0].MacAddress)
	require.Equal(t, string(metal.SwitchOperational), res.Mode)
	require.Len(t, res.Connections, 1)
	require.Equal(t, m.ID, res.Connections[0].MachineID)
	require.Equal(t, "Ethernet4", res.Connections[0].Nic.Name)
	require.Equal(t, "cc:cc:cc:cc:cc:cc", res.Connections[0].Nic.MacAddress)

	status = ts.machineGet(t, m.ID, &mres)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, mres.Hardware.Nics, 2)

	for _, n := range mres.Hardware.Nics {
		if n.Name == "eth1" {
			nic = n
		}
	}
	require.Equal(t, "eth1", nic.Name)
	require.Len(t, nic.Neighbors, 1)
	require.Equal(t, "cc:cc:cc:cc:cc:cc", nic.Neighbors[0].MacAddress)
	require.Equal(t, "Ethernet4", nic.Neighbors[0].Name)
}

type testService struct {
	partitionService *restful.WebService
	switchService    *restful.WebService
	machineService   *restful.WebService
	ds               *datastore.RethinkStore
	rethinkContainer testcontainers.Container
	ctx              context.Context
	t                *testing.T
}

func (ts *testService) terminate() {
	_ = ts.rethinkContainer.Terminate(ts.ctx)
}

func createTestService(t *testing.T) testService {
	ipamer := ipam.InitTestIpam(t)
	rethinkContainer, c, err := test.StartRethink(t)
	require.NoError(t, err)

	log := slog.Default()
	ds := datastore.New(log, c.IP+":"+c.Port, c.DB, c.User, c.Password)
	ds.VRFPoolRangeMax = 1000
	ds.ASNPoolRangeMax = 1000

	err = ds.Connect()
	require.NoError(t, err)
	err = ds.Initialize()
	require.NoError(t, err)

	psc := &mdmv1mock.ProjectServiceClient{}
	psc.On("Get", testifymock.Anything, &mdmv1.ProjectGetRequest{Id: "test-project-1"}).Return(&mdmv1.ProjectResponse{Project: &mdmv1.Project{
		Meta: &mdmv1.Meta{
			Id: "test-project-1",
		},
	}}, nil)
	psc.On("Find", testifymock.Anything, &mdmv1.ProjectFindRequest{}).Return(&mdmv1.ProjectListResponse{Projects: []*mdmv1.Project{
		{Meta: &mdmv1.Meta{Id: "test-project-1"}},
	}}, nil)
	mdc := mdm.NewMock(psc, nil, nil, nil)

	hma := security.NewHMACAuth(testUserDirectory.admin.Name, []byte{1, 2, 3}, security.WithUser(testUserDirectory.admin))
	usergetter := security.NewCreds(security.WithHMAC(hma))
	machineService, err := NewMachine(log, ds, &emptyPublisher{}, bus.DirectEndpoints(), ipamer, mdc, nil, usergetter, 0, nil, metal.DisabledIPMISuperUser())
	require.NoError(t, err)
	switchService := NewSwitch(log, ds)
	require.NoError(t, err)
	partitionService := NewPartition(log, ds, &emptyPublisher{})
	require.NoError(t, err)

	ts := testService{
		partitionService: partitionService,
		switchService:    switchService,
		machineService:   machineService,
		ds:               ds,
		rethinkContainer: rethinkContainer,
		ctx:              context.TODO(),
		t:                t,
	}
	return ts
}

func (ts *testService) partitionCreate(t *testing.T, icr v1.PartitionCreateRequest, response interface{}) int {
	return webRequestPut(t, ts.partitionService, &testUserDirectory.admin, icr, "/v1/partition/", response)
}

func (ts *testService) switchRegister(t *testing.T, srr v1.SwitchRegisterRequest, response interface{}) int {
	return webRequestPost(t, ts.switchService, &testUserDirectory.admin, srr, "/v1/switch/register", response)
}

func (ts *testService) switchGet(t *testing.T, swid string, response interface{}) int {
	return webRequestGet(t, ts.switchService, &testUserDirectory.admin, emptyBody{}, "/v1/switch/"+swid, response)
}

func (ts *testService) switchUpdate(t *testing.T, sur v1.SwitchUpdateRequest, response interface{}) int {
	return webRequestPost(t, ts.switchService, &testUserDirectory.admin, sur, "/v1/switch/", response)
}

func (ts *testService) machineGet(t *testing.T, mid string, response interface{}) int {
	return webRequestGet(t, ts.machineService, &testUserDirectory.admin, emptyBody{}, "/v1/machine/"+mid, response)
}

func (ts *testService) createPartition(name, description string) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "I am a downloadable content")
	}))
	defer s.Close()

	downloadableFile := s.URL
	partition := v1.PartitionCreateRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: "test-partition",
			},
			Describable: v1.Describable{
				Name:        &partitionName,
				Description: &partitionDesc,
			},
		},
		PartitionBootConfiguration: v1.PartitionBootConfiguration{
			ImageURL:  &downloadableFile,
			KernelURL: &downloadableFile,
		},
	}
	var createdPartition v1.PartitionResponse
	status := ts.partitionCreate(t, partition, &createdPartition)
	require.Equal(t, http.StatusCreated, status)
	require.NotNil(t, createdPartition)
	require.Equal(t, partition.Name, createdPartition.Name)
	require.NotEmpty(t, createdPartition.ID)

}
