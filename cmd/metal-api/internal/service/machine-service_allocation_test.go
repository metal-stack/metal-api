// +build integration

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/avast/retry-go"
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
	"github.com/metal-stack/metal-lib/rest"
	"github.com/metal-stack/security"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"golang.org/x/sync/errgroup"
)

var (
	hma = security.NewHMACAuth(testUserDirectory.edit.Name, []byte{1, 2, 3}, security.WithUser(testUserDirectory.edit))
)

func TestMachineAllocationIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	machineCount := 50

	rs, container := setupTestEnvironment(machineCount, t)
	defer rs.Close()

	ar := v1.MachineAllocateRequest{
		SizeID:      "s1",
		PartitionID: "p1",
		ProjectID:   "pr1",
		ImageID:     "i-1.0.0",
		Networks:    v1.MachineAllocationNetworks{{NetworkID: "private"}},
	}

	g, _ := errgroup.WithContext(context.Background())

	mu := sync.Mutex{}
	ips := make(map[string]string)

	start := time.Now()
	for i := 0; i < machineCount; i++ {
		g.Go(func() error {
			var ma v1.MachineResponse
			err := retry.Do(
				func() error {
					var err2 error
					ma, err2 = allocMachine(container, ar)
					if err2 != nil {
						t.Logf("machine allocation failed, retrying:%v", err2)
						return err2
					}
					return nil
				},
				retry.Attempts(10),
				// to have even more stress, comment the next line
				retry.DelayType(retry.CombineDelay(retry.BackOffDelay, retry.RandomDelay)),
				retry.LastErrorOnly(true),
			)
			if err != nil {
				return err
			}

			if len(ma.Allocation.MachineNetworks) < 1 {
				return fmt.Errorf("did not get a machine network")
			}
			if len(ma.Allocation.MachineNetworks[0].IPs) < 1 {
				return fmt.Errorf("did not get a private IP for machine")
			}
			ip := ma.Allocation.MachineNetworks[0].IPs[0]

			mu.Lock()
			existingmachine, ok := ips[ip]
			if ok {
				return fmt.Errorf("%s got a ip of a already allocated machine:%s", ma.ID, existingmachine)
			}
			ips[ip] = ma.ID
			mu.Unlock()
			return nil
		})
	}

	err := g.Wait()
	require.NoError(t, err)

	require.Equal(t, len(ips), machineCount)
	t.Logf("allocated:%d machines in %s", machineCount, time.Since(start))

	// Now free them all
	f, _ := errgroup.WithContext(context.Background())
	for _, id := range ips {
		id := id
		f.Go(func() error {
			err := retry.Do(
				func() error {
					// TODO add switch config in testdata to have switch updates covered
					var err2 error
					_, err2 = freeMachine(container, id)
					if err2 != nil {
						t.Logf("machine free failed, retrying:%v", err2)
						return err2
					}
					return nil
				},
				retry.Attempts(10),
				// to have even more stress, comment the next line
				retry.DelayType(retry.CombineDelay(retry.BackOffDelay, retry.RandomDelay)),
				retry.LastErrorOnly(true),
			)
			if err != nil {
				return err
			}

			return nil
		})
	}
	err = f.Wait()
	require.NoError(t, err)

}

// Methods under Test ---------------------------------------------------------------------------------------

func allocMachine(container *restful.Container, ar v1.MachineAllocateRequest) (v1.MachineResponse, error) {
	js, _ := json.Marshal(ar)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/machine/allocate", body)
	req.Header.Add("Content-Type", "application/json")
	hma.AddAuth(req, time.Now(), js)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return v1.MachineResponse{}, fmt.Errorf(w.Body.String())
	}
	var result v1.MachineResponse
	err := json.NewDecoder(resp.Body).Decode(&result)
	return result, err
}

func freeMachine(container *restful.Container, id string) (v1.MachineResponse, error) {
	js, _ := json.Marshal("")
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/v1/machine/%s/free", id), body)
	req.Header.Add("Content-Type", "application/json")
	hma.AddAuth(req, time.Now(), js)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return v1.MachineResponse{}, fmt.Errorf(w.Body.String())
	}
	var result v1.MachineResponse
	err := json.NewDecoder(resp.Body).Decode(&result)
	return result, err
}

// Helper -----------------------------------------------------------------------------------------------

func setupTestEnvironment(machineCount int, t *testing.T) (*datastore.RethinkStore, *restful.Container) {
	log := zaptest.NewLogger(t)

	_, c, err := test.StartRethink()
	require.NoError(t, err)

	ws := &grpc.Server{
		WaitService: &grpc.WaitService{
			Publisher: NopPublisher{},
			Logger:    log.Sugar(),
		},
	}

	rs := datastore.New(log, c.IP+":"+c.Port, c.DB, c.User, c.Password)
	rs.VRFPoolRangeMax = 1000
	rs.ASNPoolRangeMax = 1000

	err = rs.Connect()
	require.NoError(t, err)
	err = rs.Initialize()
	require.NoError(t, err)

	psc := &mdmv1mock.ProjectServiceClient{}
	psc.On("Get", context.Background(), &mdmv1.ProjectGetRequest{Id: "pr1"}).Return(&mdmv1.ProjectResponse{Project: &mdmv1.Project{}}, nil)
	mdc := mdm.NewMock(psc, nil)

	_, pg, err := test.StartPostgres()
	require.NoError(t, err)
	pgStorage, err := goipam.NewPostgresStorage(pg.IP, pg.Port, pg.User, pg.Password, pg.DB, goipam.SSLModeDisable)
	require.NoError(t, err)

	ipamer := goipam.NewWithStorage(pgStorage)

	createTestdata(machineCount, rs, ipamer, t)

	ms, err := NewMachine(rs, &emptyPublisher{}, bus.DirectEndpoints(), ipam.New(ipamer), mdc, ws, nil)
	require.NoError(t, err)
	container := restful.NewContainer().Add(ms)
	usergetter := security.NewCreds(security.WithHMAC(hma))
	container.Filter(rest.UserAuth(usergetter))
	return rs, container
}

func createTestdata(machineCount int, rs *datastore.RethinkStore, ipamer goipam.Ipamer, t *testing.T) {
	for i := 0; i < machineCount; i++ {
		id := fmt.Sprintf("WaitingMaschine%d", i)
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

	super, err := ipamer.NewPrefix("10.0.0.0/20")
	require.NoError(t, err)
	private, err := ipamer.AcquireChildPrefix(super.Cidr, 22)
	require.NoError(t, err)
	privateNetwork, err := metal.NewPrefixFromCIDR(private.Cidr)
	require.NoError(t, err)
	require.NotNil(t, privateNetwork)

	err = rs.CreateNetwork(&metal.Network{Base: metal.Base{ID: "super"}, PrivateSuper: true, PartitionID: "p1"})
	require.NoError(t, err)
	err = rs.CreateNetwork(&metal.Network{Base: metal.Base{ID: "private"}, ParentNetworkID: "super", ProjectID: "pr1", PartitionID: "p1", Prefixes: metal.Prefixes{*privateNetwork}})
	require.NoError(t, err)
	err = rs.CreatePartition(&metal.Partition{Base: metal.Base{ID: "p1"}})
	require.NoError(t, err)
	err = rs.CreateSize(&metal.Size{Base: metal.Base{ID: "s1"}})
	require.NoError(t, err)
}
