package grpc

import (
	"context"
	"log/slog"
	"reflect"
	"sync"
	"testing"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"github.com/metal-stack/metal-lib/bus"
	"github.com/stretchr/testify/require"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
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

func (p *emptyPublisher) Stop() {}
func TestBootService_Register(t *testing.T) {
	tests := []struct {
		name                 string
		uuid                 string
		numcores             int
		memory               int
		dbsizes              []metal.Size
		dbmachines           metal.Machines
		neighbormac1         metal.MacAddress
		neighbormac2         metal.MacAddress
		expectedErrorMessage string
		expectedSizeId       string
	}{
		{
			name:           "insert new",
			uuid:           "0",
			dbsizes:        []metal.Size{testdata.Sz1},
			neighbormac1:   testdata.Switch1.Nics[0].MacAddress,
			neighbormac2:   testdata.Switch2.Nics[0].MacAddress,
			numcores:       1,
			memory:         100,
			expectedSizeId: testdata.Sz1.ID,
		},
		{
			name:           "insert existing",
			uuid:           "1",
			dbsizes:        []metal.Size{testdata.Sz1},
			neighbormac1:   testdata.Switch1.Nics[0].MacAddress,
			neighbormac2:   testdata.Switch2.Nics[0].MacAddress,
			dbmachines:     metal.Machines{testdata.M1},
			numcores:       1,
			memory:         100,
			expectedSizeId: testdata.Sz1.ID,
		},
		{
			name:                 "insert existing without second neighbor",
			uuid:                 "1",
			dbsizes:              []metal.Size{testdata.Sz1},
			neighbormac1:         testdata.Switch1.Nics[0].MacAddress,
			dbmachines:           metal.Machines{testdata.M1},
			numcores:             1,
			memory:               100,
			expectedErrorMessage: "machine 1 is not connected to exactly two switches, found connections to 1 switches",
		},
		{
			name:                 "empty uuid",
			uuid:                 "",
			dbsizes:              []metal.Size{testdata.Sz1},
			expectedErrorMessage: "uuid is empty",
		},
		{
			name:           "new with unknown size",
			uuid:           "0",
			dbsizes:        []metal.Size{testdata.Sz1},
			neighbormac1:   testdata.Switch1.Nics[0].MacAddress,
			neighbormac2:   testdata.Switch2.Nics[0].MacAddress,
			numcores:       2,
			memory:         100,
			expectedSizeId: metal.UnknownSize().ID,
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {

			ds, mock := datastore.InitMockDB(t)

			if len(tt.dbmachines) > 0 {
				mock.On(r.DB("mockdb").Table("size").Get(tt.dbmachines[0].SizeID)).Return([]metal.Size{testdata.Sz1}, nil)
				mock.On(r.DB("mockdb").Table("machine").Get(tt.dbmachines[0].ID).Replace(r.MockAnything())).Return(testdata.EmptyResult, nil)
			} else {
				mock.On(r.DB("mockdb").Table("machine").Get("0")).Return(nil, nil)
				mock.On(r.DB("mockdb").Table("machine").Insert(r.MockAnything(), r.InsertOpts{
					Conflict: "replace",
				})).Return(testdata.EmptyResult, nil)
			}
			mock.On(r.DB("mockdb").Table("size").Get(metal.UnknownSize().ID)).Return([]metal.Size{*metal.UnknownSize()}, nil)
			mock.On(r.DB("mockdb").Table("switch").Filter(r.MockAnything(), r.FilterOpts{})).Return([]metal.Switch{testdata.Switch1, testdata.Switch2}, nil)
			mock.On(r.DB("mockdb").Table("event").Filter(r.MockAnything(), r.FilterOpts{})).Return([]metal.ProvisioningEventContainer{}, nil)
			mock.On(r.DB("mockdb").Table("event").Insert(r.MockAnything(), r.InsertOpts{})).Return(testdata.EmptyResult, nil)
			testdata.InitMockDBData(mock)

			req := &v1.BootServiceRegisterRequest{
				Uuid: tt.uuid,
				Hardware: &v1.MachineHardware{
					Memory: uint64(tt.memory),
					Disks: []*v1.MachineBlockDevice{
						{
							Size: 1000000000000,
						},
					},
					Cpus: []*v1.MachineCPU{
						{
							Model:   "Intel Xeon Silver",
							Cores:   uint32(tt.numcores),
							Threads: uint32(tt.numcores),
						},
					},
					Nics: []*v1.MachineNic{
						{
							Mac: "aa", Neighbors: []*v1.MachineNic{{Mac: string(tt.neighbormac1)}},
						},
						{
							Mac: "bb", Neighbors: []*v1.MachineNic{{Mac: string(tt.neighbormac2)}},
						},
					},
				},
				Bios: &v1.MachineBIOS{
					Version: "3.3",
					Vendor:  "Supermicro",
				},
				Ipmi: &v1.MachineIPMI{
					Address:   testdata.IPMI1.Address,
					Interface: testdata.IPMI1.Interface,
					Mac:       testdata.IPMI1.MacAddress,
					Fru: &v1.MachineFRU{
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
			}

			bootService := &BootService{
				log:              slog.Default(),
				ds:               ds,
				ipmiSuperUser:    metal.DisabledIPMISuperUser(),
				publisher:        &emptyPublisher{},
				consumer:         &bus.Consumer{},
				eventService:     &EventService{},
				queue:            sync.Map{},
				responseInterval: 0,
				checkInterval:    0,
			}

			result, err := bootService.Register(context.Background(), req)

			if tt.expectedErrorMessage != "" {
				require.Error(t, err)
				require.Regexp(t, tt.expectedErrorMessage, err.Error())
			} else {
				require.NoError(t, err)
				expectedid := "0"
				if len(tt.dbmachines) > 0 {
					expectedid = tt.dbmachines[0].ID
				}
				require.Equal(t, expectedid, result.Uuid)
				require.Equal(t, tt.expectedSizeId, result.Size)
				require.Equal(t, testdata.Partition1.ID, result.PartitionId)
			}
		})
	}
}

func TestBootService_Report(t *testing.T) {
	tests := []struct {
		name    string
		req     *v1.BootServiceReportRequest
		want    *v1.BootServiceReportResponse
		wantErr bool
		errMsg  string
	}{
		{
			name: "finalize successfully",
			req: &v1.BootServiceReportRequest{
				Uuid:     testdata.M1.ID,
				BootInfo: &v1.BootInfo{},
			},
			want: &v1.BootServiceReportResponse{},
		},
		{
			name:    "finalize unknown machine",
			req:     &v1.BootServiceReportRequest{Uuid: "999"},
			wantErr: true,
		},
		{
			name:    "finalize unallocated machine",
			req:     &v1.BootServiceReportRequest{Uuid: testdata.M3.ID},
			wantErr: true,
			errMsg:  "the machine \"3\" is not allocated",
		},
	}
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	mock.On(r.DB("mockdb").Table("switch").Filter(r.MockAnything(), r.FilterOpts{})).Return([]metal.Switch{testdata.Switch1}, nil)

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			b := &BootService{
				log:              slog.Default(),
				ds:               ds,
				ipmiSuperUser:    metal.DisabledIPMISuperUser(),
				publisher:        &emptyPublisher{},
				consumer:         &bus.Consumer{},
				eventService:     &EventService{},
				queue:            sync.Map{},
				responseInterval: 0,
				checkInterval:    0,
			}
			got, err := b.Report(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("BootService.Report() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BootService.Report() = %v, want %v", got, tt.want)
			}
		})
	}
}
