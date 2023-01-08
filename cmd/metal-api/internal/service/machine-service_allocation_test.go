//go:build integration
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

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	grpcv1 "github.com/metal-stack/metal-api/pkg/api/v1"

	"github.com/avast/retry-go/v4"
	"github.com/emicklei/go-restful/v3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"golang.org/x/sync/errgroup"

	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	mdmv1mock "github.com/metal-stack/masterdata-api/api/v1/mocks"
	mdm "github.com/metal-stack/masterdata-api/pkg/client"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	metalgrpc "github.com/metal-stack/metal-api/cmd/metal-api/internal/grpc"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/ipam"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-api/test"
	"github.com/metal-stack/metal-lib/bus"
	"github.com/metal-stack/metal-lib/rest"
	"github.com/metal-stack/security"
)

var (
	hma = security.NewHMACAuth(testUserDirectory.edit.Name, []byte{1, 2, 3}, security.WithUser(testUserDirectory.edit))
)

func TestMachineAllocationIntegration(t *testing.T) {
	machineCount := 30

	// Setup
	rs, container := setupTestEnvironment(machineCount, t)
	defer rs.Close()

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	port := 50006
	conn, err := grpc.DialContext(ctx, fmt.Sprintf("localhost:%d", port), opts...)
	require.NoError(t, err)

	c := grpcv1.NewBootServiceClient(conn)

	// Register
	e, _ := errgroup.WithContext(context.Background())
	for i := 0; i < machineCount; i++ {
		i := i
		e.Go(func() error {
			var ma *grpcv1.BootServiceRegisterResponse
			mr := createMachineRegisterRequest(i)
			err := retry.Do(
				func() error {
					var err2 error
					ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					defer cancel()
					ma, err2 = c.Register(ctx, mr)
					if err2 != nil {
						t.Logf("machine registration failed, retrying:%v", err2.Error())
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

			t.Logf("machine:%s registered", ma.Uuid)
			return nil
		})
	}
	err = e.Wait()
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

			if ma.Allocation.Creator != editUserEmail {
				return fmt.Errorf("unexpected machine creator: %s, expected: %s", ma.Allocation.Creator, editUserEmail)
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
	js, err := json.Marshal(ar)
	if err != nil {
		return v1.MachineResponse{}, err
	}
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
	err = json.NewDecoder(resp.Body).Decode(&result)
	return result, err
}

func freeMachine(container *restful.Container, id string) (v1.MachineResponse, error) {
	js, err := json.Marshal("")
	if err != nil {
		return v1.MachineResponse{}, err
	}
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
	err = json.NewDecoder(resp.Body).Decode(&result)
	return result, err
}

// Helper -----------------------------------------------------------------------------------------------

const (
	swp1MacPrefix = "bb:ca"
	swp2MacPrefix = "bb:cb"
)

func createMachineRegisterRequest(i int) *grpcv1.BootServiceRegisterRequest {
	return &grpcv1.BootServiceRegisterRequest{
		Uuid: fmt.Sprintf("WaitingMachine%d", i),
		Bios: &grpcv1.MachineBIOS{
			Version: "a",
			Vendor:  "metal",
			Date:    "1970",
		},
		Hardware: &grpcv1.MachineHardware{
			Memory:   4,
			CpuCores: 4,
			Nics: []*grpcv1.MachineNic{
				{
					Name: "lan0",
					Mac:  fmt.Sprintf("aa:ba:%d", i),
					Neighbors: []*grpcv1.MachineNic{
						{
							Name: fmt.Sprintf("swp-%d", i),
							Mac:  fmt.Sprintf("%s:%d", swp1MacPrefix, i),
						},
					},
				},
				{
					Name: "lan1",
					Mac:  fmt.Sprintf("aa:bb:%d", i),
					Neighbors: []*grpcv1.MachineNic{
						{
							Name: fmt.Sprintf("swp-%d", i),
							Mac:  fmt.Sprintf("%s:%d", swp2MacPrefix, i),
						},
					},
				},
			},
		},
	}
}

func setupTestEnvironment(machineCount int, t *testing.T) (*datastore.RethinkStore, *restful.Container) {
	log := zaptest.NewLogger(t).Sugar()

	_, c, err := test.StartRethink(t)
	require.NoError(t, err)

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

	ipamer := ipam.InitTestIpam(t)
	createTestdata(machineCount, rs, ipamer, t)

	go func() {
		err := metalgrpc.Run(&metalgrpc.ServerConfig{
			Context:          context.Background(),
			Store:            rs,
			Publisher:        NopPublisher{},
			Logger:           log,
			GrpcPort:         50006,
			TlsEnabled:       false,
			ResponseInterval: 2 * time.Millisecond,
			CheckInterval:    1 * time.Hour,
		})
		require.NoError(t, err)
	}()

	usergetter := security.NewCreds(security.WithHMAC(hma))
	ms, err := NewMachine(log, rs, &emptyPublisher{}, bus.DirectEndpoints(), ipamer, mdc, nil, usergetter, 0, nil)
	require.NoError(t, err)
	container := restful.NewContainer().Add(ms)
	container.Filter(rest.UserAuth(usergetter, zaptest.NewLogger(t).Sugar()))
	return rs, container
}

func createTestdata(machineCount int, rs *datastore.RethinkStore, ipamer ipam.IPAMer, t *testing.T) {
	for i := 0; i < machineCount; i++ {
		id := fmt.Sprintf("WaitingMachine%d", i)
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

	super := metal.Prefix{IP: "10.0.0.0", Length: "20"}
	err = ipamer.CreatePrefix(super)
	require.NoError(t, err)
	private, err := ipamer.AllocateChildPrefix(super, 22)
	require.NoError(t, err)
	privateNetwork, err := metal.NewPrefixFromCIDR(private.String())
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
