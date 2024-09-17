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

func TestSwitchMigrateIntegration(t *testing.T) {
	ts := createTestService(t)
	defer ts.terminate()

	testPartitionID := "test-partition"
	testRackID := "test-rack"

	cumulus1 := metal.Switch{
		Base: metal.Base{
			ID:   "test-switch01",
			Name: "",
		},
		Nics: []metal.Nic{
			{
				MacAddress: "aa:aa:aa:aa:aa:aa",
				Name:       "swp1",
			},
		},
		PartitionID: testPartitionID,
		RackID:      testRackID,
		OS:          &metal.SwitchOS{Vendor: metal.SwitchOSVendorCumulus},
	}

	cumulus2 := metal.Switch{
		Base: metal.Base{
			ID:   "test-switch02",
			Name: "",
		},
		Nics: []metal.Nic{
			{
				MacAddress: "bb:bb:bb:bb:bb:bb",
				Name:       "swp1",
			},
		},
		PartitionID: testPartitionID,
		RackID:      testRackID,
		OS:          &metal.SwitchOS{Vendor: metal.SwitchOSVendorCumulus},
	}

	m := &metal.Machine{
		Base: metal.Base{
			ID: "test-machine",
		},
		PartitionID: testPartitionID,
		RackID:      testRackID,
		Hardware: metal.MachineHardware{
			Nics: []metal.Nic{
				{
					Name:       "eth0",
					MacAddress: "11:11:11:11:11:11",
					Neighbors: []metal.Nic{
						{
							Name:       cumulus1.Nics[0].Name,
							MacAddress: cumulus1.Nics[0].MacAddress,
						},
					},
				},
				{
					Name:       "eth1",
					MacAddress: "22:22:22:22:22:22",
					Neighbors: []metal.Nic{
						{
							Name:       cumulus2.Nics[0].Name,
							MacAddress: cumulus2.Nics[0].MacAddress,
						},
					},
				},
			},
		},
	}

	sonic1 := metal.Switch{
		Base: metal.Base{
			ID: "test-switch01-sonic",
		},
		Nics: []metal.Nic{
			{
				MacAddress: "cc:cc:cc:cc:cc:cc",
				Name:       "Ethernet0",
			},
		},
		PartitionID: testPartitionID,
		RackID:      testRackID,
		OS:          &metal.SwitchOS{Vendor: metal.SwitchOSVendorSonic},
	}

	wantConnections1 := metal.ConnectionMap{
		m.ID: metal.Connections{
			{
				Nic: metal.Nic{
					Name:       sonic1.Nics[0].Name,
					MacAddress: sonic1.Nics[0].MacAddress,
				},
				MachineID: m.ID,
			},
		},
	}

	wantMachineNics1 := metal.Nics{
		{
			Name:       m.Hardware.Nics[0].Name,
			MacAddress: m.Hardware.Nics[0].MacAddress,
			Neighbors: []metal.Nic{
				{
					Name:       sonic1.Nics[0].Name,
					MacAddress: sonic1.Nics[0].MacAddress,
				},
			},
		},
		{
			Name:       m.Hardware.Nics[1].Name,
			MacAddress: m.Hardware.Nics[1].MacAddress,
			Neighbors: []metal.Nic{
				{
					Name:       cumulus2.Nics[0].Name,
					MacAddress: cumulus2.Nics[0].MacAddress,
				},
			},
		},
	}

	sonic2 := metal.Switch{
		Base: metal.Base{
			ID: "test-switch02-sonic",
		},
		Nics: []metal.Nic{
			{
				MacAddress: "dd:dd:dd:dd:dd:dd",
				Name:       "Ethernet0",
			},
		},
		PartitionID: testPartitionID,
		RackID:      testRackID,
		OS:          &metal.SwitchOS{Vendor: metal.SwitchOSVendorSonic},
	}

	wantConnections2 := metal.ConnectionMap{
		m.ID: metal.Connections{
			{
				Nic: metal.Nic{
					Name:       sonic2.Nics[0].Name,
					MacAddress: sonic2.Nics[0].MacAddress,
				},
				MachineID: m.ID,
			},
		},
	}

	wantMachineNics2 := metal.Nics{
		{
			Name:       m.Hardware.Nics[0].Name,
			MacAddress: m.Hardware.Nics[0].MacAddress,
			Neighbors: []metal.Nic{
				{
					Name:       sonic1.Nics[0].Name,
					MacAddress: sonic1.Nics[0].MacAddress,
				},
			},
		},
		{
			Name:       m.Hardware.Nics[1].Name,
			MacAddress: m.Hardware.Nics[1].MacAddress,
			Neighbors: []metal.Nic{
				{
					Name:       sonic2.Nics[0].Name,
					MacAddress: sonic2.Nics[0].MacAddress,
				},
			},
		},
	}

	ts.createPartition(testPartitionID, testPartitionID, "Test Partition")

	ts.registerSwitch(cumulus1, true)
	ts.registerSwitch(cumulus2, true)
	ts.createMachine(m)

	ts.registerSwitch(sonic1, true)
	ts.migrateSwitch(cumulus1, sonic1, wantConnections1)
	ts.checkMachineNics(m.ID, wantMachineNics1)

	ts.registerSwitch(sonic2, true)
	ts.migrateSwitch(cumulus2, sonic2, wantConnections2)
	ts.checkMachineNics(m.ID, wantMachineNics2)
}

func TestSwitchReplaceIntegration(t *testing.T) {
	ts := createTestService(t)
	defer ts.terminate()

	testPartitionID := "test-partition"
	testRackID := "test-rack"

	cumulus1 := metal.Switch{
		Base: metal.Base{
			ID:   "test-switch01",
			Name: "",
		},
		Nics: []metal.Nic{
			{
				MacAddress: "aa:aa:aa:aa:aa:aa",
				Name:       "swp1",
			},
		},
		PartitionID: testPartitionID,
		RackID:      testRackID,
		OS:          &metal.SwitchOS{Vendor: metal.SwitchOSVendorCumulus},
	}

	cumulus2 := metal.Switch{
		Base: metal.Base{
			ID:   "test-switch02",
			Name: "",
		},
		Nics: []metal.Nic{
			{
				MacAddress: "bb:bb:bb:bb:bb:bb",
				Name:       "swp1",
			},
		},
		PartitionID: testPartitionID,
		RackID:      testRackID,
		OS:          &metal.SwitchOS{Vendor: metal.SwitchOSVendorCumulus},
	}

	m := &metal.Machine{
		Base: metal.Base{
			ID: "test-machine",
		},
		PartitionID: testPartitionID,
		RackID:      testRackID,
		Hardware: metal.MachineHardware{
			Nics: []metal.Nic{
				{
					Name:       "eth0",
					MacAddress: "11:11:11:11:11:11",
					Neighbors: []metal.Nic{
						{
							Name:       cumulus1.Nics[0].Name,
							MacAddress: cumulus1.Nics[0].MacAddress,
						},
					},
				},
				{
					Name:       "eth1",
					MacAddress: "22:22:22:22:22:22",
					Neighbors: []metal.Nic{
						{
							Name:       cumulus2.Nics[0].Name,
							MacAddress: cumulus2.Nics[0].MacAddress,
						},
					},
				},
			},
		},
	}

	cumulus3 := metal.Switch{
		Base: metal.Base{
			ID: "test-switch01",
		},
		Nics: []metal.Nic{
			{
				MacAddress: "cc:cc:cc:cc:cc:cc",
				Name:       "swp1",
			},
		},
		PartitionID: testPartitionID,
		RackID:      testRackID,
		OS:          &metal.SwitchOS{Vendor: metal.SwitchOSVendorCumulus},
	}

	wantConnections1 := metal.ConnectionMap{
		m.ID: metal.Connections{
			{
				Nic: metal.Nic{
					Name:       cumulus3.Nics[0].Name,
					MacAddress: cumulus3.Nics[0].MacAddress,
				},
				MachineID: m.ID,
			},
		},
	}

	wantMachineNics1 := metal.Nics{
		{
			Name:       m.Hardware.Nics[0].Name,
			MacAddress: m.Hardware.Nics[0].MacAddress,
			Neighbors: []metal.Nic{
				{
					Name:       cumulus3.Nics[0].Name,
					MacAddress: cumulus3.Nics[0].MacAddress,
				},
			},
		},
		{
			Name:       m.Hardware.Nics[1].Name,
			MacAddress: m.Hardware.Nics[1].MacAddress,
			Neighbors: []metal.Nic{
				{
					Name:       cumulus2.Nics[0].Name,
					MacAddress: cumulus2.Nics[0].MacAddress,
				},
			},
		},
	}

	sonic1 := metal.Switch{
		Base: metal.Base{
			ID: "test-switch02",
		},
		Nics: []metal.Nic{
			{
				MacAddress: "dd:dd:dd:dd:dd:dd",
				Name:       "Ethernet0",
			},
		},
		PartitionID: testPartitionID,
		RackID:      testRackID,
		OS:          &metal.SwitchOS{Vendor: metal.SwitchOSVendorSonic},
	}

	wantConnections2 := metal.ConnectionMap{
		m.ID: metal.Connections{
			{
				Nic: metal.Nic{
					Name:       sonic1.Nics[0].Name,
					MacAddress: sonic1.Nics[0].MacAddress,
				},
				MachineID: m.ID,
			},
		},
	}

	wantMachineNics2 := metal.Nics{
		{
			Name:       m.Hardware.Nics[0].Name,
			MacAddress: m.Hardware.Nics[0].MacAddress,
			Neighbors: []metal.Nic{
				{
					Name:       cumulus3.Nics[0].Name,
					MacAddress: cumulus3.Nics[0].MacAddress,
				},
			},
		},
		{
			Name:       m.Hardware.Nics[1].Name,
			MacAddress: m.Hardware.Nics[1].MacAddress,
			Neighbors: []metal.Nic{
				{
					Name:       sonic1.Nics[0].Name,
					MacAddress: sonic1.Nics[0].MacAddress,
				},
			},
		},
	}

	ts.createPartition(testPartitionID, testPartitionID, "Test Partition")

	ts.registerSwitch(cumulus1, true)
	ts.registerSwitch(cumulus2, true)
	ts.createMachine(m)

	ts.replaceSwitch(cumulus3, wantConnections1)
	ts.checkMachineNics(m.ID, wantMachineNics1)

	ts.replaceSwitch(sonic1, wantConnections2)
	ts.checkMachineNics(m.ID, wantMachineNics2)
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

func (ts *testService) switchMigrate(t *testing.T, smr v1.SwitchMigrateRequest, response interface{}) int {
	return webRequestPost(t, ts.switchService, &testUserDirectory.edit, smr, "/v1/switch/migrate", response)
}

func (ts *testService) switchDelete(t *testing.T, sid string, response interface{}) int {
	return webRequestDelete(t, ts.switchService, &testUserDirectory.admin, emptyBody{}, "/v1/switch/"+sid, response)
}

func (ts *testService) createPartition(id, name, description string) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "I am a downloadable content")
	}))
	defer s.Close()

	downloadableFile := s.URL
	partitionName := name
	partitionDesc := description
	partition := v1.PartitionCreateRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: id,
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
	status := ts.partitionCreate(ts.t, partition, &createdPartition)
	require.Equal(ts.t, http.StatusCreated, status)
	require.NotNil(ts.t, createdPartition)
	require.Equal(ts.t, partition.Name, createdPartition.Name)
	require.NotEmpty(ts.t, createdPartition.ID)

}

func (ts *testService) registerSwitch(sw metal.Switch, isNewId bool) {
	nics := make([]v1.SwitchNic, 0)
	for _, nic := range sw.Nics {
		nic := v1.SwitchNic{
			MacAddress: string(nic.MacAddress),
			Name:       nic.Name,
		}
		nics = append(nics, nic)
	}

	srr := v1.SwitchRegisterRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: sw.ID,
			},
		},
		Nics:        nics,
		PartitionID: sw.PartitionID,
		SwitchBase: v1.SwitchBase{
			RackID: sw.RackID,
			OS: &v1.SwitchOS{
				Vendor: sw.OS.Vendor,
			},
		},
	}

	wantStatus := http.StatusOK
	if isNewId {
		wantStatus = http.StatusCreated
	}
	var res v1.SwitchResponse
	status := ts.switchRegister(ts.t, srr, &res)
	require.Equal(ts.t, wantStatus, status)
	ts.checkSwitchResponse(sw, &res)
}

func (ts *testService) createMachine(m *metal.Machine) {
	err := ts.ds.CreateMachine(m)
	require.NoError(ts.t, err)
	err = ts.ds.ConnectMachineWithSwitches(m)
	require.NoError(ts.t, err)

	err = ts.ds.CreateProvisioningEventContainer(&metal.ProvisioningEventContainer{
		Base:       metal.Base{ID: m.ID},
		Liveliness: metal.MachineLivelinessAlive,
	})
	require.NoError(ts.t, err)
}

func (ts *testService) migrateSwitch(oldSwitch, newSwitch metal.Switch, wantConnections metal.ConnectionMap) {
	wantSwitch := newSwitch
	wantSwitch.Mode = metal.SwitchOperational
	wantSwitch.MachineConnections = wantConnections

	smr := v1.SwitchMigrateRequest{
		OldSwitchID: oldSwitch.ID,
		NewSwitchID: newSwitch.ID,
	}

	var res v1.SwitchResponse
	ts.switchMigrate(ts.t, smr, &res)
	status := ts.switchGet(ts.t, wantSwitch.ID, &res)
	require.Equal(ts.t, http.StatusOK, status)
	ts.checkSwitchResponse(wantSwitch, &res)

	status = ts.switchDelete(ts.t, oldSwitch.ID, &res)
	require.Equal(ts.t, http.StatusOK, status)
}

func (ts *testService) replaceSwitch(newSwitch metal.Switch, wantConnections metal.ConnectionMap) {
	ts.setReplaceMode(newSwitch.ID)
	ts.registerSwitch(newSwitch, false)
	wantSwitch := newSwitch
	wantSwitch.Mode = metal.SwitchOperational
	wantSwitch.MachineConnections = wantConnections

	var res v1.SwitchResponse
	status := ts.switchGet(ts.t, wantSwitch.ID, &res)
	require.Equal(ts.t, http.StatusOK, status)
	ts.checkSwitchResponse(wantSwitch, &res)
}

func (ts *testService) setReplaceMode(id string) {
	var res v1.SwitchResponse

	sur := v1.SwitchUpdateRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: id,
			},
		},
		SwitchBase: v1.SwitchBase{
			Mode: string(metal.SwitchReplace),
		},
	}

	status := ts.switchUpdate(ts.t, sur, &res)
	require.Equal(ts.t, http.StatusOK, status)
	require.Equal(ts.t, string(metal.SwitchReplace), res.SwitchBase.Mode)
}

func (ts *testService) checkSwitchResponse(sw metal.Switch, res *v1.SwitchResponse) {
	require.NotNil(ts.t, res)
	require.Equal(ts.t, sw.Mode, metal.SwitchMode(res.Mode))

	require.Len(ts.t, res.Nics, len(sw.Nics))
	for _, nic := range sw.Nics {
		n := findNicByNameInSwitchNics(nic.Name, res.Nics)
		ts.checkCorrectNic(n, nic)
	}

	require.Len(ts.t, res.Connections, len(sw.MachineConnections))
	connectionsByNicName, err := sw.MachineConnections.ByNicName()
	require.NoError(ts.t, err)
	for nicName, con := range connectionsByNicName {
		c, found := findMachineConnection(nicName, res.Connections)
		require.True(ts.t, found)
		require.Equal(ts.t, con.MachineID, c.MachineID)
		require.Equal(ts.t, con.Nic.Name, c.Nic.Name)
		require.Equal(ts.t, con.Nic.MacAddress, c.Nic.MacAddress)
	}
}

func (ts *testService) checkMachineNics(mid string, wantNics metal.Nics) {
	var m v1.MachineResponse
	status := ts.machineGet(ts.t, mid, &m)
	require.Equal(ts.t, http.StatusOK, status)
	require.NotNil(ts.t, m)

	for _, wantNic := range wantNics {
		nic := findNicByNameInMachineNics(wantNic.Name, m.Hardware.Nics)
		ts.checkCorrectNic(nic, wantNic)
		for _, wantNeigh := range wantNic.Neighbors {
			neigh := findNicByName(wantNeigh.Name, nic.Neighbors)
			ts.checkCorrectNic(neigh, wantNeigh)
		}
	}
}

func (ts *testService) checkCorrectNic(nic *metal.Nic, wantNic metal.Nic) {
	require.NotNil(ts.t, wantNic)
	require.Equal(ts.t, wantNic.Name, nic.Name)
	require.Equal(ts.t, wantNic.MacAddress, nic.MacAddress)
}

func findNicByName(name string, nics metal.Nics) *metal.Nic {
	for _, nic := range nics {
		n := nic
		if nic.Name == name {
			return &n
		}
	}
	return nil
}

func findNicByNameInSwitchNics(name string, nics v1.SwitchNics) *metal.Nic {
	for _, nic := range nics {
		if nic.Name == name {
			n := &metal.Nic{
				Name:       nic.Name,
				MacAddress: metal.MacAddress(nic.MacAddress),
			}
			return n
		}
	}
	return nil
}

func findNicByNameInMachineNics(name string, nics v1.MachineNics) *metal.Nic {
	for _, nic := range nics {
		if nic.Name == name {
			neighbors := make(metal.Nics, 0)
			for _, neigh := range nic.Neighbors {
				n := metal.Nic{
					Name:       neigh.Name,
					MacAddress: metal.MacAddress(neigh.MacAddress),
				}
				neighbors = append(neighbors, n)
			}
			n := &metal.Nic{
				Name:       nic.Name,
				MacAddress: metal.MacAddress(nic.MacAddress),
				Neighbors:  neighbors,
			}
			return n
		}
	}
	return nil
}

func findMachineConnection(nicName string, connections []v1.SwitchConnection) (*metal.Connection, bool) {
	for _, con := range connections {
		if con.Nic.Name == nicName {
			c := &metal.Connection{
				Nic: metal.Nic{
					Name:       con.Nic.Name,
					MacAddress: metal.MacAddress(con.Nic.MacAddress),
				},
				MachineID: con.MachineID,
			}
			return c, true
		}
	}
	return nil, false
}
