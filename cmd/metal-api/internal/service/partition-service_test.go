package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"

	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"github.com/stretchr/testify/require"
)

type nopTopicCreator struct {
}

func (n nopTopicCreator) CreateTopic(topicFQN string) error {
	return nil
}

type expectingTopicCreator struct {
	t              *testing.T
	expectedTopics []string
}

func (n expectingTopicCreator) CreateTopic(topicFQN string) error {
	ass := assert.New(n.t)
	ass.NotEmpty(topicFQN)
	ass.Contains(n.expectedTopics, topicFQN, "Expectation %v contains %s failed.", n.expectedTopics, topicFQN)
	return nil
}

func TestGetPartitions(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)

	service := NewPartition(slog.Default(), ds, &nopTopicCreator{})
	container := restful.NewContainer().Add(service)
	req := httptest.NewRequest("GET", "/v1/partition", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result []v1.PartitionResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Len(t, result, 3)
	require.Equal(t, testdata.Partition1.ID, result[0].ID)
	require.Equal(t, testdata.Partition1.Name, *result[0].Name)
	require.Equal(t, testdata.Partition1.Description, *result[0].Description)
	require.Equal(t, testdata.Partition2.ID, result[1].ID)
	require.Equal(t, testdata.Partition2.Name, *result[1].Name)
	require.Equal(t, testdata.Partition2.Description, *result[1].Description)
	require.Equal(t, testdata.Partition3.ID, result[2].ID)
	require.Equal(t, testdata.Partition3.Name, *result[2].Name)
	require.Equal(t, testdata.Partition3.Description, *result[2].Description)
}

func TestGetPartition(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)

	service := NewPartition(slog.Default(), ds, &nopTopicCreator{})
	container := restful.NewContainer().Add(service)
	req := httptest.NewRequest("GET", "/v1/partition/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.PartitionResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.Partition1.ID, result.ID)
	require.Equal(t, testdata.Partition1.Name, *result.Name)
	require.Equal(t, testdata.Partition1.Description, *result.Description)
}

func TestGetPartitionNotFound(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	service := NewPartition(log, ds, &nopTopicCreator{})
	container := restful.NewContainer().Add(service)
	req := httptest.NewRequest("GET", "/v1/partition/999", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode, w.Body.String())
	var result httperrors.HTTPErrorResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Contains(t, result.Message, "999")
	require.Equal(t, 404, result.StatusCode)
}

func TestDeletePartition(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	service := NewPartition(log, ds, &nopTopicCreator{})
	container := restful.NewContainer().Add(service)
	req := httptest.NewRequest("DELETE", "/v1/partition/1", nil)
	container = injectAdmin(log, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.PartitionResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.Partition1.ID, result.ID)
	require.Equal(t, testdata.Partition1.Name, *result.Name)
	require.Equal(t, testdata.Partition1.Description, *result.Description)
}

func TestCreatePartition(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	topicCreator := expectingTopicCreator{
		t:              t,
		expectedTopics: []string{"1-switch", "1-machine"},
	}
	service := NewPartition(log, ds, topicCreator)
	container := restful.NewContainer().Add(service)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "I am a downloadable content")
	}))
	defer ts.Close()

	downloadableFile := ts.URL

	createRequest := v1.PartitionCreateRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: testdata.Partition1.ID,
			},
			Describable: v1.Describable{
				Name:        &testdata.Partition1.Name,
				Description: &testdata.Partition1.Description,
			},
		},
		PartitionBootConfiguration: v1.PartitionBootConfiguration{
			ImageURL:  &downloadableFile,
			KernelURL: &downloadableFile,
		},
	}
	js, err := json.Marshal(createRequest)
	require.NoError(t, err)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("PUT", "/v1/partition", body)
	req.Header.Add("Content-Type", "application/json")
	container = injectAdmin(log, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode, w.Body.String())
	var result v1.PartitionResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.Partition1.ID, result.ID)
	require.Equal(t, testdata.Partition1.Name, *result.Name)
	require.Equal(t, testdata.Partition1.Description, *result.Description)
}

func TestUpdatePartition(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	service := NewPartition(log, ds, &nopTopicCreator{})
	container := restful.NewContainer().Add(service)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "I am a downloadable content")
	}))
	defer ts.Close()

	mgmtService := "mgmt"
	downloadableFile := ts.URL
	updateRequest := v1.PartitionUpdateRequest{
		Common: v1.Common{
			Describable: v1.Describable{
				Name:        &testdata.Partition2.Name,
				Description: &testdata.Partition2.Description,
			},
			Identifiable: v1.Identifiable{
				ID: testdata.Partition1.ID,
			},
		},
		MgmtServiceAddress: &mgmtService,
		PartitionBootConfiguration: &v1.PartitionBootConfiguration{
			ImageURL: &downloadableFile,
		},
	}
	js, err := json.Marshal(updateRequest)
	require.NoError(t, err)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/partition", body)
	req.Header.Add("Content-Type", "application/json")
	container = injectAdmin(log, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.PartitionResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.Partition1.ID, result.ID)
	require.Equal(t, testdata.Partition2.Name, *result.Name)
	require.Equal(t, testdata.Partition2.Description, *result.Description)
	require.Equal(t, mgmtService, *result.MgmtServiceAddress)
	require.Equal(t, downloadableFile, *result.PartitionBootConfiguration.ImageURL)
}

func TestPartitionCapacity(t *testing.T) {
	var (
		mockMachines = func(mock *r.Mock, reservations []metal.Reservation, ms ...metal.Machine) {
			var (
				sizes      metal.Sizes
				events     metal.ProvisioningEventContainers
				partitions metal.Partitions
			)

			for _, m := range ms {
				ec := metal.ProvisioningEventContainer{Base: metal.Base{ID: m.ID}, Liveliness: metal.MachineLivelinessAlive}
				if m.Waiting {
					ec.Events = append(ec.Events, metal.ProvisioningEvent{
						Event: metal.ProvisioningEventWaiting,
					})
				}
				if m.Allocation != nil {
					ec.Events = append(ec.Events, metal.ProvisioningEvent{
						Event: metal.ProvisioningEventPhonedHome,
					})
				}
				events = append(events, ec)
				if !slices.ContainsFunc(sizes, func(s metal.Size) bool {
					return s.ID == m.SizeID
				}) {
					s := metal.Size{Base: metal.Base{ID: m.SizeID}}
					sizes = append(sizes, s)
				}
				if !slices.ContainsFunc(partitions, func(p metal.Partition) bool {
					return p.ID == m.PartitionID
				}) {
					partitions = append(partitions, metal.Partition{Base: metal.Base{ID: m.PartitionID}})
				}
			}

			if len(reservations) > 0 {
				for i := range sizes {
					sizes[i].Reservations = append(sizes[i].Reservations, reservations...)
				}
			}

			mock.On(r.DB("mockdb").Table("machine")).Return(ms, nil)
			mock.On(r.DB("mockdb").Table("event")).Return(events, nil)
			mock.On(r.DB("mockdb").Table("partition")).Return(partitions, nil)
			mock.On(r.DB("mockdb").Table("size")).Return(sizes, nil)
		}

		machineTpl = func(id, partition, size, project string) metal.Machine {
			m := metal.Machine{
				Base:        metal.Base{ID: id},
				PartitionID: partition,
				SizeID:      size,
				IPMI: metal.IPMI{ // required for healthy machine state
					Address:     "1.2.3." + id,
					MacAddress:  "aa:bb:0" + id,
					LastUpdated: time.Now().Add(-1 * time.Minute),
				},
				State: metal.MachineState{
					Value: metal.AvailableState,
				},
			}
			if project != "" {
				m.Allocation = &metal.MachineAllocation{
					Project: project,
				}
			}
			return m
		}
	)

	tests := []struct {
		name   string
		mockFn func(mock *r.Mock)
		want   []*v1.PartitionCapacity
	}{
		{
			name: "one allocated machine",
			mockFn: func(mock *r.Mock) {
				m1 := machineTpl("1", "partition-a", "size-a", "project-123")
				mockMachines(mock, nil, m1)
			},
			want: []*v1.PartitionCapacity{
				{
					Common: v1.Common{
						Identifiable: v1.Identifiable{ID: "partition-a"}, Describable: v1.Describable{Name: pointer.Pointer(""), Description: pointer.Pointer("")},
					},
					ServerCapacities: v1.ServerCapacities{
						{
							Size:       "size-a",
							Total:      1,
							PhonedHome: 1,
							Allocated:  1,
						},
					},
				},
			},
		},
		{
			name: "two allocated machines",
			mockFn: func(mock *r.Mock) {
				m1 := machineTpl("1", "partition-a", "size-a", "project-123")
				m2 := machineTpl("2", "partition-a", "size-a", "project-123")
				mockMachines(mock, nil, m1, m2)
			},
			want: []*v1.PartitionCapacity{
				{
					Common: v1.Common{
						Identifiable: v1.Identifiable{ID: "partition-a"}, Describable: v1.Describable{Name: pointer.Pointer(""), Description: pointer.Pointer("")},
					},
					ServerCapacities: v1.ServerCapacities{
						{
							Size:       "size-a",
							Total:      2,
							PhonedHome: 2,
							Allocated:  2,
						},
					},
				},
			},
		},
		{
			name: "one faulty, allocated machine",
			mockFn: func(mock *r.Mock) {
				m1 := machineTpl("1", "partition-a", "size-a", "project-123")
				m1.IPMI.Address = ""
				mockMachines(mock, nil, m1)
			},
			want: []*v1.PartitionCapacity{
				{
					Common: v1.Common{
						Identifiable: v1.Identifiable{ID: "partition-a"}, Describable: v1.Describable{Name: pointer.Pointer(""), Description: pointer.Pointer("")},
					},
					ServerCapacities: v1.ServerCapacities{
						{
							Size:           "size-a",
							Total:          1,
							PhonedHome:     1,
							Faulty:         1,
							Allocated:      1,
							FaultyMachines: []string{"1"},
						},
					},
				},
			},
		},
		{
			name: "one waiting machine",
			mockFn: func(mock *r.Mock) {
				m1 := machineTpl("1", "partition-a", "size-a", "")
				m1.Waiting = true
				mockMachines(mock, nil, m1)
			},
			want: []*v1.PartitionCapacity{
				{
					Common: v1.Common{
						Identifiable: v1.Identifiable{ID: "partition-a"}, Describable: v1.Describable{Name: pointer.Pointer(""), Description: pointer.Pointer("")},
					},
					ServerCapacities: v1.ServerCapacities{
						{
							Size:    "size-a",
							Total:   1,
							Waiting: 1,
							Free:    1,
						},
					},
				},
			},
		},
		{
			name: "one waiting, one allocated machine",
			mockFn: func(mock *r.Mock) {
				m1 := machineTpl("1", "partition-a", "size-a", "")
				m1.Waiting = true
				m2 := machineTpl("2", "partition-a", "size-a", "project-123")
				mockMachines(mock, nil, m1, m2)
			},
			want: []*v1.PartitionCapacity{
				{
					Common: v1.Common{
						Identifiable: v1.Identifiable{ID: "partition-a"}, Describable: v1.Describable{Name: pointer.Pointer(""), Description: pointer.Pointer("")},
					},
					ServerCapacities: v1.ServerCapacities{
						{
							Size:       "size-a",
							Total:      2,
							Allocated:  1,
							Waiting:    1,
							PhonedHome: 1,
							Free:       1,
						},
					},
				},
			},
		},
		{
			name: "one free machine",
			mockFn: func(mock *r.Mock) {
				m1 := machineTpl("1", "partition-a", "size-a", "")
				m1.Waiting = true
				m1.State.Value = metal.AvailableState
				mockMachines(mock, nil, m1)
			},
			want: []*v1.PartitionCapacity{
				{
					Common: v1.Common{
						Identifiable: v1.Identifiable{ID: "partition-a"}, Describable: v1.Describable{Name: pointer.Pointer(""), Description: pointer.Pointer("")},
					},
					ServerCapacities: v1.ServerCapacities{
						{
							Size:    "size-a",
							Total:   1,
							Waiting: 1,
							Free:    1,
						},
					},
				},
			},
		},
		{
			name: "one machine rebooting",
			mockFn: func(mock *r.Mock) {
				m1 := machineTpl("1", "partition-a", "size-a", "")
				m1.Waiting = false
				mockMachines(mock, nil, m1)
			},
			want: []*v1.PartitionCapacity{
				{
					Common: v1.Common{
						Identifiable: v1.Identifiable{ID: "partition-a"}, Describable: v1.Describable{Name: pointer.Pointer(""), Description: pointer.Pointer("")},
					},
					ServerCapacities: v1.ServerCapacities{
						{
							Size:          "size-a",
							Total:         1,
							Other:         1,
							Unavailable:   1,
							OtherMachines: []string{"1"},
						},
					},
				},
			},
		},
		{
			name: "reserved machine does not count as free",
			mockFn: func(mock *r.Mock) {
				m1 := machineTpl("1", "partition-a", "size-a", "")
				m1.Waiting = true

				reservations := []metal.Reservation{
					{
						Amount:       1,
						ProjectID:    "project-123",
						PartitionIDs: []string{"partition-a"},
					},
				}

				mockMachines(mock, reservations, m1)
			},
			want: []*v1.PartitionCapacity{
				{
					Common: v1.Common{
						Identifiable: v1.Identifiable{ID: "partition-a"}, Describable: v1.Describable{Name: pointer.Pointer(""), Description: pointer.Pointer("")},
					},
					ServerCapacities: v1.ServerCapacities{
						{
							Size:             "size-a",
							Total:            1,
							Waiting:          1,
							Free:             0,
							Reservations:     1,
							UsedReservations: 0,
						},
					},
				},
			},
		},
		{
			name: "overbooked partition, free count capped at 0",
			mockFn: func(mock *r.Mock) {
				m1 := machineTpl("1", "partition-a", "size-a", "")
				m1.Waiting = true

				reservations := []metal.Reservation{
					{
						Amount:       1,
						ProjectID:    "project-123",
						PartitionIDs: []string{"partition-a"},
					},
					{
						Amount:       2,
						ProjectID:    "project-456",
						PartitionIDs: []string{"partition-a"},
					},
				}

				mockMachines(mock, reservations, m1)
			},
			want: []*v1.PartitionCapacity{
				{
					Common: v1.Common{
						Identifiable: v1.Identifiable{ID: "partition-a"}, Describable: v1.Describable{Name: pointer.Pointer(""), Description: pointer.Pointer("")},
					},
					ServerCapacities: v1.ServerCapacities{
						{
							Size:             "size-a",
							Total:            1,
							Waiting:          1,
							Free:             0,
							Reservations:     3,
							UsedReservations: 0,
						},
					},
				},
			},
		},
		{
			name: "reservations already used up (edge)",
			mockFn: func(mock *r.Mock) {
				m1 := machineTpl("1", "partition-a", "size-a", "project-123")
				m2 := machineTpl("2", "partition-a", "size-a", "project-123")
				m3 := machineTpl("3", "partition-a", "size-a", "")
				m3.Waiting = true

				reservations := []metal.Reservation{
					{
						Amount:       2,
						ProjectID:    "project-123",
						PartitionIDs: []string{"partition-a"},
					},
				}

				mockMachines(mock, reservations, m1, m2, m3)
			},
			want: []*v1.PartitionCapacity{
				{
					Common: v1.Common{
						Identifiable: v1.Identifiable{ID: "partition-a"}, Describable: v1.Describable{Name: pointer.Pointer(""), Description: pointer.Pointer("")},
					},
					ServerCapacities: v1.ServerCapacities{
						{
							Size:             "size-a",
							Total:            3,
							Allocated:        2,
							Waiting:          1,
							Free:             1,
							Reservations:     2,
							UsedReservations: 2,
							PhonedHome:       2,
						},
					},
				},
			},
		},
		{
			name: "reservations already used up",
			mockFn: func(mock *r.Mock) {
				m1 := machineTpl("1", "partition-a", "size-a", "project-123")
				m2 := machineTpl("2", "partition-a", "size-a", "project-123")
				m3 := machineTpl("3", "partition-a", "size-a", "")
				m3.Waiting = true

				reservations := []metal.Reservation{
					{
						Amount:       1,
						ProjectID:    "project-123",
						PartitionIDs: []string{"partition-a"},
					},
				}

				mockMachines(mock, reservations, m1, m2, m3)
			},
			want: []*v1.PartitionCapacity{
				{
					Common: v1.Common{
						Identifiable: v1.Identifiable{ID: "partition-a"}, Describable: v1.Describable{Name: pointer.Pointer(""), Description: pointer.Pointer("")},
					},
					ServerCapacities: v1.ServerCapacities{
						{
							Size:             "size-a",
							Total:            3,
							Allocated:        2,
							Waiting:          1,
							Free:             1,
							Reservations:     1,
							UsedReservations: 1,
							PhonedHome:       2,
						},
					},
				},
			},
		},
		{
			name: "other partition size reservation has no influence",
			mockFn: func(mock *r.Mock) {
				m1 := machineTpl("1", "partition-a", "size-a", "project-123")
				m2 := machineTpl("2", "partition-a", "size-a", "project-123")
				m3 := machineTpl("3", "partition-a", "size-a", "")
				m3.Waiting = true

				reservations := []metal.Reservation{
					{
						Amount:       2,
						ProjectID:    "project-123",
						PartitionIDs: []string{"partition-a"},
					},
					{
						Amount:       2,
						ProjectID:    "project-123",
						PartitionIDs: []string{"partition-b"},
					},
				}

				mockMachines(mock, reservations, m1, m2, m3)
			},
			want: []*v1.PartitionCapacity{
				{
					Common: v1.Common{
						Identifiable: v1.Identifiable{ID: "partition-a"}, Describable: v1.Describable{Name: pointer.Pointer(""), Description: pointer.Pointer("")},
					},
					ServerCapacities: v1.ServerCapacities{
						{
							Size:             "size-a",
							Total:            3,
							Allocated:        2,
							Waiting:          1,
							Free:             1,
							Reservations:     2,
							UsedReservations: 2,
							PhonedHome:       2,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				ds, mock = datastore.InitMockDB(t)
				body     = &v1.PartitionCapacityRequest{}
				ws       = NewPartition(slog.Default(), ds, nil)
			)

			if tt.mockFn != nil {
				tt.mockFn(mock)
			}

			code, got := genericWebRequest[[]*v1.PartitionCapacity](t, ws, testViewUser, body, "POST", "/v1/partition/capacity")
			assert.Equal(t, http.StatusOK, code)

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("diff (-want +got):\n%s", diff)
			}
		})
	}
}
