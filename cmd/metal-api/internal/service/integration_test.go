// +build integration

package service

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	metalgrpc "github.com/metal-stack/metal-api/cmd/metal-api/internal/grpc"
	"github.com/metal-stack/metal-api/test"
	"github.com/metal-stack/metal-lib/bus"
	"github.com/metal-stack/security"
	"github.com/testcontainers/testcontainers-go"
	"go.uber.org/zap/zaptest"

	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	mdmv1mock "github.com/metal-stack/masterdata-api/api/v1/mocks"
	mdm "github.com/metal-stack/masterdata-api/pkg/client"

	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/ipam"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	grpcv1 "github.com/metal-stack/metal-api/pkg/api/v1"

	"github.com/stretchr/testify/require"
)

type testEnv struct {
	imageService        *restful.WebService
	switchService       *restful.WebService
	sizeService         *restful.WebService
	networkService      *restful.WebService
	partitionService    *restful.WebService
	machineService      *restful.WebService
	ipService           *restful.WebService
	ws                  *metalgrpc.WaitService
	ds                  *datastore.RethinkStore
	privateSuperNetwork *v1.NetworkResponse
	privateNetwork      *v1.NetworkResponse
	rethinkContainer    testcontainers.Container
	ctx                 context.Context
}

func (te *testEnv) teardown() {
	_ = te.rethinkContainer.Terminate(te.ctx)
}

//nolint:deadcode
func createTestEnvironment(t *testing.T) testEnv {
	require := require.New(t)

	ipamer := ipam.InitTestIpam(t)
	rethinkContainer, c, err := test.StartRethink()
	require.NoError(err)

	ds := datastore.New(zaptest.NewLogger(t), c.IP+":"+c.Port, c.DB, c.User, c.Password)
	ds.VRFPoolRangeMax = 1000
	ds.ASNPoolRangeMax = 1000

	err = ds.Connect()
	require.NoError(err)
	err = ds.Initialize()
	require.NoError(err)

	psc := &mdmv1mock.ProjectServiceClient{}
	psc.On("Get", context.Background(), &mdmv1.ProjectGetRequest{Id: "test-project-1"}).Return(&mdmv1.ProjectResponse{Project: &mdmv1.Project{
		Meta: &mdmv1.Meta{
			Id: "test-project-1",
		},
	}}, nil)
	mdc := mdm.NewMock(psc, nil)

	log := zaptest.NewLogger(t)
	grpcServer, err := metalgrpc.NewServer(&metalgrpc.ServerConfig{
		Datasource:       ds,
		Logger:           log.Sugar(),
		GrpcPort:         50005,
		TlsEnabled:       false,
		ResponseInterval: 2 * time.Millisecond,
		CheckInterval:    1 * time.Hour,
	})
	require.NoError(err)
	go func() {
		err := grpcServer.Serve()
		require.Nil(t, err)
	}()
	grpcServer.Publisher = NopPublisher{} // has to be done after constructor because init would fail otherwise

	machineService, err := NewMachine(ds, &emptyPublisher{}, bus.DirectEndpoints(), ipamer, mdc, grpcServer, nil)
	require.NoError(err)
	imageService := NewImage(ds)
	switchService := NewSwitch(ds)
	sizeService := NewSize(ds)
	networkService := NewNetwork(ds, ipamer, mdc)
	partitionService := NewPartition(ds, &emptyPublisher{})
	ipService, err := NewIP(ds, bus.DirectEndpoints(), ipamer, mdc)
	require.NoError(err)

	te := testEnv{
		imageService:     imageService,
		switchService:    switchService,
		sizeService:      sizeService,
		networkService:   networkService,
		partitionService: partitionService,
		machineService:   machineService,
		ipService:        ipService,
		ds:               ds,
		ws:               grpcServer.WaitService,
		rethinkContainer: rethinkContainer,
		ctx:              context.TODO(),
	}

	imageID := "test-image-1.0.0"
	imageName := "testimage"
	imageDesc := "Test Image"
	image := v1.ImageCreateRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: imageID,
			},
			Describable: v1.Describable{
				Name:        &imageName,
				Description: &imageDesc,
			},
		},
		URL:      "https://www.google.com", // not good to rely on this page
		Features: []string{string(metal.ImageFeatureMachine)},
	}
	var createdImage v1.ImageResponse

	status := te.imageCreate(t, image, &createdImage)
	require.Equal(http.StatusCreated, status)
	require.NotNil(createdImage)
	require.Equal(image.ID, createdImage.ID)

	sizeName := "testsize"
	sizeDesc := "Test Size"
	size := v1.SizeCreateRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: "test-size",
			},
			Describable: v1.Describable{
				Name:        &sizeName,
				Description: &sizeDesc,
			},
		},
		SizeConstraints: []v1.SizeConstraint{
			{
				Type: metal.CoreConstraint,
				Min:  8,
				Max:  8,
			},
			{
				Type: metal.MemoryConstraint,
				Min:  1000,
				Max:  2000,
			},
			{
				Type: metal.StorageConstraint,
				Min:  2000,
				Max:  3000,
			},
		},
	}
	var createdSize v1.SizeResponse
	status = te.sizeCreate(t, size, &createdSize)
	require.Equal(http.StatusCreated, status)
	require.NotNil(createdSize)
	require.Equal(size.ID, createdSize.ID)

	err = ds.CreateFilesystemLayout(&metal.FilesystemLayout{Base: metal.Base{ID: "fsl1"}, Constraints: metal.FilesystemLayoutConstraints{Sizes: []string{"test-size"}, Images: map[string]string{"test-image": "*"}}})
	require.NoError(err)

	partitionName := "test-partition"
	partitionDesc := "Test Partition"
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
	}
	var createdPartition v1.PartitionResponse
	status = te.partitionCreate(t, partition, &createdPartition)
	require.Equal(http.StatusCreated, status)
	require.NotNil(createdPartition)
	require.Equal(partition.Name, createdPartition.Name)
	require.NotEmpty(createdPartition.ID)

	switchID := "test-switch01"
	sw := v1.SwitchRegisterRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: switchID,
			},
		},
		SwitchBase: v1.SwitchBase{
			RackID: "test-rack",
		},
		Nics: v1.SwitchNics{
			{
				MacAddress: "bb:aa:aa:aa:aa:aa",
				Name:       "swp1",
			},
		},
		PartitionID: "test-partition",
	}
	var createdSwitch v1.SwitchResponse

	status = te.switchRegister(t, sw, &createdSwitch)
	require.Equal(http.StatusCreated, status)
	require.NotNil(createdSwitch)
	require.Equal(sw.ID, createdSwitch.ID)
	require.Len(sw.Nics, 1)
	require.Equal(sw.Nics[0].Name, createdSwitch.Nics[0].Name)
	require.Equal(sw.Nics[0].MacAddress, createdSwitch.Nics[0].MacAddress)

	switchID = "test-switch02"
	sw = v1.SwitchRegisterRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: switchID,
			},
		},
		SwitchBase: v1.SwitchBase{
			RackID: "test-rack",
		},
		Nics: v1.SwitchNics{
			{
				MacAddress: "aa:bb:aa:aa:aa:aa",
				Name:       "swp1",
			},
		},
		PartitionID: "test-partition",
	}

	status = te.switchRegister(t, sw, &createdSwitch)
	require.Equal(http.StatusCreated, status)
	require.NotNil(createdSwitch)
	require.Equal(sw.ID, createdSwitch.ID)
	require.Len(sw.Nics, 1)
	require.Equal(sw.Nics[0].Name, createdSwitch.Nics[0].Name)
	require.Equal(sw.Nics[0].MacAddress, createdSwitch.Nics[0].MacAddress)

	var createdNetwork v1.NetworkResponse
	networkID := "test-private-super"
	networkName := "test-private-super-network"
	networkDesc := "Test Private Super Network"
	testPrivateSuperCidr := "10.0.0.0/16"
	ncr := v1.NetworkCreateRequest{
		ID: &networkID,
		Describable: v1.Describable{
			Name:        &networkName,
			Description: &networkDesc,
		},
		NetworkBase: v1.NetworkBase{
			PartitionID: &partition.ID,
		},
		NetworkImmutable: v1.NetworkImmutable{
			Prefixes:     []string{testPrivateSuperCidr},
			PrivateSuper: true,
		},
	}
	status = te.networkCreate(t, ncr, &createdNetwork)
	require.Equal(http.StatusCreated, status)
	require.NotNil(createdNetwork)
	require.Equal(*ncr.ID, createdNetwork.ID)

	te.privateSuperNetwork = &createdNetwork

	var acquiredPrivateNetwork v1.NetworkResponse
	privateNetworkName := "test-private-network"
	privateNetworkDesc := "Test Private Network"
	projectID := "test-project-1"
	nar := v1.NetworkAllocateRequest{
		Describable: v1.Describable{
			Name:        &privateNetworkName,
			Description: &privateNetworkDesc,
		},
		NetworkBase: v1.NetworkBase{
			ProjectID:   &projectID,
			PartitionID: &partition.ID,
		},
	}
	status = te.networkAcquire(t, nar, &acquiredPrivateNetwork)
	require.Equal(http.StatusCreated, status)
	require.NotNil(acquiredPrivateNetwork)
	require.Equal(ncr.ID, acquiredPrivateNetwork.ParentNetworkID)
	require.Len(acquiredPrivateNetwork.Prefixes, 1)
	_, ipnet, _ := net.ParseCIDR(testPrivateSuperCidr)
	_, privateNet, _ := net.ParseCIDR(acquiredPrivateNetwork.Prefixes[0])
	require.True(ipnet.Contains(privateNet.IP), "%s must be within %s", privateNet, ipnet)
	te.privateNetwork = &acquiredPrivateNetwork

	return te
}

var adminUser = &security.User{
	Tenant: "provider",
	Groups: []security.ResourceAccess{
		"maas-all-all-admin",
	},
}

func (te *testEnv) sizeCreate(t *testing.T, icr v1.SizeCreateRequest, response interface{}) int {
	return webRequestPut(t, te.sizeService, adminUser, icr, "/v1/size/", response)
}

func (te *testEnv) partitionCreate(t *testing.T, icr v1.PartitionCreateRequest, response interface{}) int {
	return webRequestPut(t, te.partitionService, adminUser, icr, "/v1/partition/", response)
}

func (te *testEnv) switchRegister(t *testing.T, srr v1.SwitchRegisterRequest, response interface{}) int {
	return webRequestPost(t, te.switchService, adminUser, srr, "/v1/switch/register", response)
}

func (te *testEnv) switchGet(t *testing.T, swid string, response interface{}) int {
	return webRequestGet(t, te.switchService, adminUser, emptyBody{}, "/v1/switch/"+swid, response)
}

func (te *testEnv) imageCreate(t *testing.T, icr v1.ImageCreateRequest, response interface{}) int {
	return webRequestPut(t, te.imageService, adminUser, icr, "/v1/image/", response)
}

func (te *testEnv) networkCreate(t *testing.T, icr v1.NetworkCreateRequest, response interface{}) int {
	return webRequestPut(t, te.networkService, adminUser, icr, "/v1/network/", response)
}

func (te *testEnv) networkAcquire(t *testing.T, nar v1.NetworkAllocateRequest, response interface{}) int {
	return webRequestPost(t, te.networkService, adminUser, nar, "/v1/network/allocate", response)
}

func (te *testEnv) machineAllocate(t *testing.T, mar v1.MachineAllocateRequest, response interface{}) int {
	return webRequestPost(t, te.machineService, adminUser, mar, "/v1/machine/allocate", response)
}

func (te *testEnv) machineFree(t *testing.T, uuid string, response interface{}) int {
	return webRequestDelete(t, te.machineService, adminUser, &emptyBody{}, "/v1/machine/"+uuid+"/free", response)
}

func (te *testEnv) machineRegister(t *testing.T, mrr v1.MachineRegisterRequest, response interface{}) int {
	return webRequestPost(t, te.machineService, adminUser, mrr, "/v1/machine/register", response)
}

func (te *testEnv) machineWait(uuid string) error {
	kacp := keepalive.ClientParameters{
		Time:                5 * time.Millisecond,
		Timeout:             20 * time.Millisecond,
		PermitWithoutStream: true,
	}
	opts := []grpc.DialOption{
		grpc.WithKeepaliveParams(kacp),
		grpc.WithInsecure(),
		grpc.WithBlock(),
	}

	port := 50005
	conn, err := grpc.DialContext(context.Background(), fmt.Sprintf("localhost:%d", port), opts...)
	if err != nil {
		return err
	}

	isWaiting := make(chan bool)

	go func() {
		waitClient := grpcv1.NewWaitClient(conn)
		waitForAllocation(uuid, waitClient, context.Background())
		isWaiting <- true
	}()

	<-isWaiting

	return nil
}

func waitForAllocation(machineID string, c grpcv1.WaitClient, ctx context.Context) {
	req := &grpcv1.WaitRequest{
		MachineID: machineID,
	}

	for {
		_, err := c.Wait(ctx, req)
		time.Sleep(5 * time.Millisecond)
		if err != nil {
			continue
		}
		return
	}
}
