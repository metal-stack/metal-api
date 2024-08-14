//go:build integration
// +build integration

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	grpcv1 "github.com/metal-stack/metal-api/pkg/api/v1"

	"github.com/avast/retry-go/v4"
	"github.com/emicklei/go-restful/v3"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	goipam "github.com/metal-stack/go-ipam"
	"github.com/metal-stack/go-ipam/api/v1/apiv1connect"
	"github.com/metal-stack/go-ipam/pkg/service"

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
	log := slog.Default()

	nsqContainer, publisher, consumer := test.StartNsqd(t, log)
	rethinkContainer, cd, err := test.StartRethink(t)
	require.NoError(t, err)

	defer func() {
		_ = rethinkContainer.Terminate(context.Background())
		_ = nsqContainer.Terminate(context.Background())
	}()

	rs := datastore.New(log, cd.IP+":"+cd.Port, cd.DB, cd.User, cd.Password)
	rs.VRFPoolRangeMax = 1000
	rs.ASNPoolRangeMax = 1000

	err = rs.Connect()
	require.NoError(t, err)

	err = rs.Initialize()
	require.NoError(t, err)

	webContainer, listener := setupTestEnvironment(machineCount, t, rs, publisher, consumer)

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.NewClient(listener.Addr().String(), opts...)
	require.NoError(t, err)

	c := grpcv1.NewBootServiceClient(conn)

	// Register
	e, _ := errgroup.WithContext(context.Background())
	for i := range machineCount {
		e.Go(func() error {
			mr := createMachineRegisterRequest(i)
			err := retry.Do(
				func() error {
					var err2 error
					ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					defer cancel()
					_, err2 = c.Register(ctx, mr)
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
	for range machineCount {
		g.Go(func() error {
			var ma v1.MachineResponse
			err := retry.Do(
				func() error {
					var err2 error
					ma, err2 = allocMachine(webContainer, ar)
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
			return nil
		})
	}
	err = g.Wait()
	require.NoError(t, err)
	require.Len(t, ips, machineCount)
	t.Logf("allocated:%d machines in %s", machineCount, time.Since(start))

	// Free
	f, _ := errgroup.WithContext(context.Background())
	for _, id := range ips {
		id := id
		f.Go(func() error {
			err := retry.Do(
				func() error {
					// TODO add switch config in testdata to have switch updates covered
					var err2 error
					_, err2 = freeMachine(webContainer, id)
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
		return v1.MachineResponse{}, errors.New(w.Body.String())
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
		return v1.MachineResponse{}, errors.New(w.Body.String())
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
			Memory: 4,
			Cpus: []*grpcv1.MachineCPU{
				{
					Model:   "Intel Xeon Silver",
					Cores:   4,
					Threads: 4,
				},
			},
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

func setupTestEnvironment(machineCount int, t *testing.T, ds *datastore.RethinkStore, publisher bus.Publisher, consumer *bus.Consumer) (*restful.Container, net.Listener) {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	psc := &mdmv1mock.ProjectServiceClient{}
	psc.On("Get", testifymock.Anything, &mdmv1.ProjectGetRequest{Id: "pr1"}).Return(&mdmv1.ProjectResponse{Project: &mdmv1.Project{}}, nil)
	mdc := mdm.NewMock(psc, nil, nil, nil)

	_, pg, err := test.StartPostgres()
	require.NoError(t, err)

	pgStorage, err := goipam.NewPostgresStorage(pg.IP, pg.Port, pg.User, pg.Password, pg.DB, goipam.SSLModeDisable)
	require.NoError(t, err)

	ipamer := goipam.NewWithStorage(pgStorage)

	mux := http.NewServeMux()
	mux.Handle(apiv1connect.NewIpamServiceHandler(
		service.New(log.WithGroup("ipamservice"), ipamer),
	))
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()

	ipamclient := apiv1connect.NewIpamServiceClient(
		server.Client(),
		server.URL,
	)

	metalIPAMer := ipam.New(ipamclient)

	createTestdata(machineCount, ds, metalIPAMer, t)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	go func() {
		err := metalgrpc.Run(&metalgrpc.ServerConfig{
			Context:          context.Background(),
			Store:            ds,
			Publisher:        publisher,
			Consumer:         consumer,
			Listener:         listener,
			Logger:           log,
			TlsEnabled:       false,
			ResponseInterval: 2 * time.Millisecond,
			CheckInterval:    1 * time.Hour,
		})
		if err != nil {
			t.Errorf("grpc server shutdown unexpectedly: %s", err)
		}
	}()

	usergetter := security.NewCreds(security.WithHMAC(hma))
	ms, err := NewMachine(log, ds, publisher, bus.NewEndpoints(consumer, publisher), metalIPAMer, mdc, nil, usergetter, 0, nil, metal.DisabledIPMISuperUser())
	require.NoError(t, err)
	container := restful.NewContainer().Add(ms)
	container.Filter(rest.UserAuth(usergetter, log))

	return container, listener
}

func createTestdata(machineCount int, rs *datastore.RethinkStore, ipamer ipam.IPAMer, t *testing.T) {
	for i := range machineCount {
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

	ctx := context.Background()
	tenantSuper := metal.Prefix{IP: "10.0.0.0", Length: "20"}
	err = ipamer.CreatePrefix(ctx, tenantSuper)
	require.NoError(t, err)
	private, err := ipamer.AllocateChildPrefix(ctx, tenantSuper, 22)
	require.NoError(t, err)
	require.NotNil(t, private)
	privateNetwork, err := metal.NewPrefixFromCIDR(private.String())
	require.NoError(t, err)
	require.NotNil(t, privateNetwork)

	t.Logf("tenant super network:%s and private network created:%s created", tenantSuper, *private)

	err = rs.CreateNetwork(&metal.Network{Base: metal.Base{ID: "super"}, PrivateSuper: true, PartitionID: "p1", Prefixes: metal.Prefixes{tenantSuper}})
	require.NoError(t, err)
	err = rs.CreateNetwork(&metal.Network{Base: metal.Base{ID: "private"}, ParentNetworkID: "super", ProjectID: "pr1", PartitionID: "p1", Prefixes: metal.Prefixes{*privateNetwork}})
	require.NoError(t, err)
	err = rs.CreatePartition(&metal.Partition{Base: metal.Base{ID: "p1"}})
	require.NoError(t, err)
	err = rs.CreateSize(&metal.Size{Base: metal.Base{ID: "s1"}, Constraints: []metal.Constraint{{Type: metal.MemoryConstraint, Min: 4, Max: 4}, {Type: metal.CoreConstraint, Min: 4, Max: 4}}})
	require.NoError(t, err)

	sw1nics := metal.Nics{}
	sw2nics := metal.Nics{}
	for j := range machineCount {
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
