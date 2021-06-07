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

	// Setup
	rs, container := setupTestEnvironment(machineCount, t)
	defer rs.Close()

	// Register
	e, _ := errgroup.WithContext(context.Background())
	for i := 0; i < machineCount; i++ {
		i := i
		e.Go(func() error {
			var ma v1.MachineResponse
			mr := createMachineRegisterRequest(i)
			err := retry.Do(
				func() error {
					var err2 error
					ma, err2 = registerMachine(container, mr)
					if err2 != nil {
						// FIXME err is a machineResponse ?
						// t.Logf("machine registration failed, retrying:%v", err2.Error())
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
			nics := ma.Hardware.Nics
			if len(nics) < 2 {
				return fmt.Errorf("did not get 2 nics")
			}
			t.Logf("machine:%s registered", ma.ID)
			return nil
		})
	}
	err := e.Wait()
	require.NoError(t, err)

	// Allocate
	g, _ := errgroup.WithContext(context.Background())
	ar := v1.MachineAllocateRequest{
		SizeID:      "s1",
		PartitionID: "p1",
		ProjectID:   "pr1",
		ImageID:     "i-1.0.0",
		Networks:    v1.MachineAllocationNetworks{{NetworkID: "private"}},
	}
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
			t.Logf("machine:%s allocated", ma.ID)
			return nil
		})
	}
	err = g.Wait()
	require.NoError(t, err)
	require.Equal(t, len(ips), machineCount)
	t.Logf("allocated:%d machines in %s", machineCount, time.Since(start))

	// Free
	f, _ := errgroup.WithContext(context.Background())
	for _, id := range ips {
		id := id
		f.Go(func() error {
			var ma v1.MachineResponse
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
			t.Logf("machine:%s freed", ma.ID)

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

func registerMachine(container *restful.Container, rm v1.MachineRegisterRequest) (v1.MachineResponse, error) {
	js, _ := json.Marshal(rm)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/machine/register", body)
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

const (
	swp1MacPrefix = "bb:ca"
	swp2MacPrefix = "bb:cb"
)

func createMachineRegisterRequest(i int) v1.MachineRegisterRequest {
	return v1.MachineRegisterRequest{
		UUID:        fmt.Sprintf("WaitingMachine%d", i),
		PartitionID: "p1",
		Hardware: v1.MachineHardwareExtended{
			MachineHardwareBase: v1.MachineHardwareBase{Memory: 4, CPUCores: 4},
			Nics: v1.MachineNicsExtended{
				{
					MachineNic: v1.MachineNic{
						Name:       "lan0",
						MacAddress: fmt.Sprintf("aa:ba:%d", i),
					},
					Neighbors: v1.MachineNicsExtended{
						{
							MachineNic: v1.MachineNic{
								Name:       fmt.Sprintf("swp-%d", i),
								MacAddress: fmt.Sprintf("%s:%d", swp1MacPrefix, i),
							},
						},
					},
				},
				{
					MachineNic: v1.MachineNic{
						Name:       "lan1",
						MacAddress: fmt.Sprintf("aa:bb:%d", i),
					},
					Neighbors: v1.MachineNicsExtended{
						{
							MachineNic: v1.MachineNic{
								Name:       fmt.Sprintf("swp-%d", i),
								MacAddress: fmt.Sprintf("%s:%d", swp2MacPrefix, i),
							},
						},
					},
				},
			},
		},
	}
}

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

	ms, err := NewMachine(rs, &emptyPublisher{}, bus.DirectEndpoints(), ipam.New(ipamer), mdc, ws, nil, nil, 0)
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
	err = rs.CreateSize(&metal.Size{Base: metal.Base{ID: "s1"}, Constraints: []metal.Constraint{{Type: metal.MemoryConstraint, Min: 4, Max: 4}, {Type: metal.CoreConstraint, Min: 4, Max: 4}}})
	require.NoError(t, err)

	sw1nics := metal.Nics{}
	sw2nics := metal.Nics{}
	for j := 0; j < machineCount; j++ {
		sw1nic := metal.Nic{
			Name:       fmt.Sprintf("swp-%d", j),
			MacAddress: metal.MacAddress(fmt.Sprintf("%s:%d", swp1MacPrefix, j)),
		}
		sw2nic := metal.Nic{
			Name:       fmt.Sprintf("swp-%d", j),
			MacAddress: metal.MacAddress(fmt.Sprintf("%s:%d", swp2MacPrefix, j)),
		}
		sw1nics = append(sw1nics, sw1nic)
		sw2nics = append(sw2nics, sw2nic)
	}
	err = rs.CreateSwitch(&metal.Switch{Base: metal.Base{ID: "sw1"}, PartitionID: "p1", Nics: sw1nics, MachineConnections: metal.ConnectionMap{}})
	require.NoError(t, err)
	err = rs.CreateSwitch(&metal.Switch{Base: metal.Base{ID: "sw2"}, PartitionID: "p1", Nics: sw2nics, MachineConnections: metal.ConnectionMap{}})
	require.NoError(t, err)
	err = rs.CreateFilesystemLayout(&metal.FilesystemLayout{Base: metal.Base{ID: "fsl1"}, Constraints: metal.FilesystemLayoutConstraints{Sizes: []string{"s1"}, Images: map[string]string{"i": "*"}}})
	require.NoError(t, err)
}
