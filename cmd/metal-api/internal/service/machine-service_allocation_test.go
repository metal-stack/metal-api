package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emicklei/go-restful/v3"
	goipam "github.com/metal-stack/go-ipam"
	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	mdmv1mock "github.com/metal-stack/masterdata-api/api/v1/mocks"
	mdm "github.com/metal-stack/masterdata-api/pkg/client"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/grpc"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/ipam"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-api/test"
	"github.com/metal-stack/metal-lib/bus"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"golang.org/x/sync/errgroup"
)

func TestMachineAllocationIntegration(t *testing.T) {

	datastore.VRFPoolRangeMax = 100
	datastore.ASNPoolRangeMax = 100

	_, c, err := test.StartRethink()
	require.NoError(t, err)
	log := zaptest.NewLogger(t)

	ws := &grpc.Server{
		WaitService: &grpc.WaitService{
			Publisher: NopPublisher{},
			Logger:    log.Sugar(),
		},
	}

	rs1 := datastore.New(log, c.IP+":"+c.Port, c.DB, c.User, c.Password)
	rs2 := datastore.New(log, c.IP+":"+c.Port, c.DB, c.User, c.Password)
	rs3 := datastore.New(log, c.IP+":"+c.Port, c.DB, c.User, c.Password)
	rss := []*datastore.RethinkStore{rs1, rs2, rs3}

	for i := range rss {
		rs := rss[i]
		err = rs.Connect()
		require.NoError(t, err)
		defer rs.Close()
	}
	err = rs1.Initialize()
	require.NoError(t, err)

	psc := &mdmv1mock.ProjectServiceClient{}
	psc.On("Get", context.Background(), &mdmv1.ProjectGetRequest{Id: "pr1"}).Return(&mdmv1.ProjectResponse{Project: &mdmv1.Project{}}, nil)
	mdc := mdm.NewMock(psc, nil)

	ipamer := goipam.New()

	createTestdata(rs1, ipamer, t)

	ms1, err := NewMachine(rs1, &emptyPublisher{}, bus.DirectEndpoints(), ipam.New(ipamer), mdc, ws, nil)
	require.NoError(t, err)
	ms2, err := NewMachine(rs2, &emptyPublisher{}, bus.DirectEndpoints(), ipam.New(ipamer), mdc, ws, nil)
	require.NoError(t, err)
	ms3, err := NewMachine(rs3, &emptyPublisher{}, bus.DirectEndpoints(), ipam.New(ipamer), mdc, ws, nil)
	require.NoError(t, err)
	mss := []*restful.Container{
		restful.NewContainer().Add(ms1),
		restful.NewContainer().Add(ms2),
		restful.NewContainer().Add(ms3),
	}

	for i := range mss {
		getMachine(mss[i], t)
	}

	ar := v1.MachineAllocateRequest{
		SizeID:      "s1",
		PartitionID: "p1",
		ProjectID:   "pr1",
		ImageID:     "i-1.0.0",
		Networks:    v1.MachineAllocationNetworks{{NetworkID: "private"}},
	}

	g, _ := errgroup.WithContext(context.Background())

	for i := 0; i < 5; i++ {
		g.Go(func() error {
			return allocMachine(mss[1], ar, t)
		})
	}

	err = g.Wait()
	require.NoError(t, err)

}

func createTestdata(rs *datastore.RethinkStore, ipamer goipam.Ipamer, t *testing.T) {
	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("M%d", i)
		m := &metal.Machine{
			Base:        metal.Base{ID: id},
			SizeID:      "s1",
			PartitionID: "p1",
			Waiting:     true,
			State:       metal.MachineState{Value: metal.AvailableState},
		}
		err := rs.CreateMachine(m)
		require.NoError(t, err)
		err = rs.CreateProvisioningEventContainer(&metal.ProvisioningEventContainer{Base: metal.Base{ID: id}, Liveliness: metal.MachineLivelinessAlive})
		require.NoError(t, err)
	}
	err := rs.CreateImage(&metal.Image{Base: metal.Base{ID: "i-1.0.0"}, OS: "i", Version: "1.0.0", Features: map[metal.ImageFeatureType]bool{metal.ImageFeatureMachine: true}})
	require.NoError(t, err)

	super, err := ipamer.NewPrefix("10.0.0.0/14")
	require.NoError(t, err)
	private, err := ipamer.AcquireChildPrefix(super.Cidr, 22)
	require.NoError(t, err)
	privateNetwork, err := metal.NewPrefixFromCIDR(private.Cidr)
	require.NoError(t, err)
	require.NotNil(t, privateNetwork)

	err = rs.CreateNetwork(&metal.Network{Base: metal.Base{ID: "super"}, PrivateSuper: true, PartitionID: "p1", Prefixes: metal.Prefixes{{IP: "10.0.0.0", Length: "8"}}})
	require.NoError(t, err)
	err = rs.CreateNetwork(&metal.Network{Base: metal.Base{ID: "private"}, ParentNetworkID: "super", ProjectID: "pr1", PartitionID: "p1", Prefixes: metal.Prefixes{*privateNetwork}})
	require.NoError(t, err)
	err = rs.CreatePartition(&metal.Partition{Base: metal.Base{ID: "p1"}})
	require.NoError(t, err)
	err = rs.CreateSize(&metal.Size{Base: metal.Base{ID: "s1"}})
	require.NoError(t, err)

}

func getMachine(container *restful.Container, t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/machine", nil)
	container = injectViewer(container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result []v1.MachineResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
}

func allocMachine(container *restful.Container, ar v1.MachineAllocateRequest, t *testing.T) error {
	js, _ := json.Marshal(ar)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/machine/allocate", body)
	req.Header.Add("Content-Type", "application/json")
	container = injectEditor(container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.MachineResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	return nil
}

type NopPublisher struct {
}

func (p NopPublisher) Publish(topic string, data interface{}) error {
	return nil
}

func (p NopPublisher) CreateTopic(topic string) error {
	return nil
}

func (p NopPublisher) Stop() {}
