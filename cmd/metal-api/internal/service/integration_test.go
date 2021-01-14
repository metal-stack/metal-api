package service

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/grpc"
	"github.com/metal-stack/metal-lib/zapup"
	"github.com/metal-stack/security"

	mdm "github.com/metal-stack/masterdata-api/pkg/client"
	"go.uber.org/zap"

	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/eventbus"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/ipam"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

//nolint:golint,unused
type testEnv struct {
	imageService        *restful.WebService
	switchService       *restful.WebService
	sizeService         *restful.WebService
	networkService      *restful.WebService
	partitionService    *restful.WebService
	machineService      *restful.WebService
	ipService           *restful.WebService
	privateSuperNetwork *v1.NetworkResponse
	privateNetwork      *v1.NetworkResponse
	rethinkContainer    testcontainers.Container
	ctx                 context.Context
}

func (te *testEnv) teardown() {
	_ = te.rethinkContainer.Terminate(te.ctx)
}

//nolint:golint,unused,deadcode
func createTestEnvironment(t *testing.T) testEnv {
	require := require.New(t)
	log, err := zap.NewDevelopment()
	require.NoError(err)

	ipamer := ipam.InitTestIpam(t)
	nsq := eventbus.InitTestPublisher(t)
	ds, rc, ctx := datastore.InitTestDB(t)
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	mdc, err := mdm.NewClient(timeoutCtx, "localhost", 50051, "certs/client.pem", "certs/client-key.pem", "certs/ca.pem", "hmac", log)
	require.NoError(err)
	grpcServer, err := grpc.NewServer(&grpc.ServerConfig{
		Datasource: ds,
		Publisher:  nsq.Publisher,
	})
	require.NoError(err)

	machineService, err := NewMachine(ds, nsq.Publisher, nsq.Endpoints, ipamer, mdc, grpcServer)
	require.NoError(err)
	imageService := NewImage(ds)
	switchService := NewSwitch(ds)
	sizeService := NewSize(ds)
	networkService := NewNetwork(ds, ipamer, mdc)
	partitionService := NewPartition(ds, nsq)
	ipService, err := NewIP(ds, nsq.Endpoints, ipamer, mdc)
	require.NoError(err)

	te := testEnv{
		imageService:     imageService,
		switchService:    switchService,
		sizeService:      sizeService,
		networkService:   networkService,
		partitionService: partitionService,
		machineService:   machineService,
		ipService:        ipService,
		rethinkContainer: rc,
		ctx:              ctx,
	}

	imageID := "test-image"
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
		URL:      "https://blobstore/image",
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

//nolint:golint,unused
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

func (te *testEnv) machineWait(uuid string) {
	container := restful.NewContainer().Add(te.machineService)
	createReq := httptest.NewRequest(http.MethodGet, "/v1/machine/"+uuid+"/wait", nil)
	container = injectAdmin(container, createReq)
	w := httptest.NewRecorder()
	for {
		container.ServeHTTP(w, createReq)
		resp := w.Result()
		var response map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			panic(err)
		}
		if resp.StatusCode == http.StatusOK {
			break
		}
		if resp.StatusCode == http.StatusInternalServerError {
			break
		}
	}
}

//nolint:golint,unused
type emptyBody struct{}

func webRequestPut(t *testing.T, service *restful.WebService, user *security.User, request interface{}, path string, response interface{}) int {
	return webRequest(t, http.MethodPut, service, user, request, path, response)
}

func webRequestPost(t *testing.T, service *restful.WebService, user *security.User, request interface{}, path string, response interface{}) int {
	return webRequest(t, http.MethodPost, service, user, request, path, response)
}

func webRequestDelete(t *testing.T, service *restful.WebService, user *security.User, request interface{}, path string, response interface{}) int {
	return webRequest(t, http.MethodDelete, service, user, request, path, response)
}

func webRequestGet(t *testing.T, service *restful.WebService, user *security.User, request interface{}, path string, response interface{}) int {
	return webRequest(t, http.MethodGet, service, user, request, path, response)
}

//nolint:golint,unused
func webRequest(t *testing.T, method string, service *restful.WebService, user *security.User, request interface{}, path string, response interface{}) int {
	container := restful.NewContainer().Add(service)

	jsonBody, err := json.Marshal(request)
	require.NoError(t, err)
	body := ioutil.NopCloser(strings.NewReader(string(jsonBody)))
	createReq := httptest.NewRequest(method, path, body)
	createReq.Header.Set("Content-Type", "application/json")

	container.Filter(MockAuth(user))

	w := httptest.NewRecorder()
	container.ServeHTTP(w, createReq)

	resp := w.Result()

	err = json.NewDecoder(resp.Body).Decode(response)
	require.NoError(t, err)
	return resp.StatusCode
}

func MockAuth(user *security.User) restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		log := zapup.RequestLogger(req.Request)
		rq := req.Request
		ctx := security.PutUserInContext(zapup.PutLogger(rq.Context(), log), user)
		req.Request = rq.WithContext(ctx)
		chain.ProcessFilter(req, resp)
	}
}
