package service

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	restful "github.com/emicklei/go-restful/v3"
	"github.com/google/go-cmp/cmp"
	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	mdmv1mock "github.com/metal-stack/masterdata-api/api/v1/mocks"
	mdm "github.com/metal-stack/masterdata-api/pkg/client"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

func TestGetSizes(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)

	sizeservice := NewSize(slog.Default(), ds, nil)
	container := restful.NewContainer().Add(sizeservice)
	req := httptest.NewRequest("GET", "/v1/size", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result []v1.SizeResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Len(t, result, 3)
	require.Equal(t, testdata.Sz1.ID, result[0].ID)
	require.Equal(t, testdata.Sz1.Name, *result[0].Name)
	require.Equal(t, testdata.Sz1.Description, *result[0].Description)
	require.Equal(t, testdata.Sz2.ID, result[1].ID)
	require.Equal(t, testdata.Sz2.Name, *result[1].Name)
	require.Equal(t, testdata.Sz2.Description, *result[1].Description)
	require.Equal(t, testdata.Sz3.ID, result[2].ID)
	require.Equal(t, testdata.Sz3.Name, *result[2].Name)
	require.Equal(t, testdata.Sz3.Description, *result[2].Description)
}

func TestGetSize(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)

	sizeservice := NewSize(slog.Default(), ds, nil)
	container := restful.NewContainer().Add(sizeservice)
	req := httptest.NewRequest("GET", "/v1/size/1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.SizeResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.Sz1.ID, result.ID)
	require.Equal(t, testdata.Sz1.Name, *result.Name)
	require.Equal(t, testdata.Sz1.Description, *result.Description)
	require.Equal(t, len(testdata.Sz1.Constraints), len(result.SizeConstraints))
}

func TestSuggest(t *testing.T) {
	tests := []struct {
		name   string
		mockFn func(mock *r.Mock)
		want   []metal.Constraint
	}{
		{
			name: "size",
			mockFn: func(mock *r.Mock) {
				mock.On(r.DB("mockdb").Table("machine").Get("1")).Return(&metal.Machine{
					Hardware: metal.MachineHardware{
						MetalCPUs: []metal.MetalCPU{
							{
								Model:   "Intel Xeon Silver",
								Cores:   8,
								Threads: 8,
							},
						},
						MetalGPUs: []metal.MetalGPU{
							{
								Vendor: "NVIDIA Corporation",
								Model:  "AD102GL [RTX 6000 Ada Generation]",
							},
						},
						Memory: 1 << 30,
						Disks: []metal.BlockDevice{
							{
								Size: 1000,
								Name: "/dev/nvme0n1",
							},
							{
								Size: 1000,
								Name: "/dev/nvme1n1",
							},
							{
								Size: 1000,
								Name: "/dev/nvme2n1",
							},
						},
					},
				}, nil)
			},
			want: []metal.Constraint{
				{
					Type:       metal.CoreConstraint,
					Min:        8,
					Max:        8,
					Identifier: "Intel Xeon Silver",
				},
				{
					Type: metal.MemoryConstraint,
					Min:  1 << 30,
					Max:  1 << 30,
				},
				{
					Type:       metal.StorageConstraint,
					Min:        3000,
					Max:        3000,
					Identifier: "/dev/nvme*",
				},
				{
					Type:       metal.GPUConstraint,
					Min:        1,
					Max:        1,
					Identifier: "AD102GL [RTX 6000 Ada Generation]",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var (
				ds, mock = datastore.InitMockDB(t)
				body     = &v1.SizeSuggestRequest{
					MachineID: "1",
				}
				ws = NewSize(slog.Default(), ds, nil)
			)

			if tt.mockFn != nil {
				tt.mockFn(mock)
			}

			code, got := genericWebRequest[[]metal.Constraint](t, ws, testViewUser, body, "POST", "/v1/size/suggest")
			assert.Equal(t, http.StatusOK, code)

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetSizeNotFound(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)

	sizeservice := NewSize(slog.Default(), ds, nil)
	container := restful.NewContainer().Add(sizeservice)
	req := httptest.NewRequest("GET", "/v1/size/999", nil)
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

func TestDeleteSize(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	sizeservice := NewSize(log, ds, nil)
	container := restful.NewContainer().Add(sizeservice)
	req := httptest.NewRequest("DELETE", "/v1/size/1", nil)
	container = injectAdmin(log, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.SizeResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.Sz1.ID, result.ID)
	require.Equal(t, testdata.Sz1.Name, *result.Name)
	require.Equal(t, testdata.Sz1.Description, *result.Description)
}

func TestCreateSize(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	psc := &mdmv1mock.ProjectServiceClient{}
	psc.On("Find", testifymock.Anything, &mdmv1.ProjectFindRequest{}).Return(&mdmv1.ProjectListResponse{Projects: []*mdmv1.Project{
		{Meta: &mdmv1.Meta{Id: "a"}},
	}}, nil)
	mdc := mdm.NewMock(psc, &mdmv1mock.TenantServiceClient{}, nil, nil)

	sizeservice := NewSize(log, ds, mdc)
	container := restful.NewContainer().Add(sizeservice)

	createRequest := v1.SizeCreateRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: testdata.Sz1.ID,
			},
			Describable: v1.Describable{
				Name:        &testdata.Sz1.Name,
				Description: &testdata.Sz1.Description,
			},
		},
		SizeConstraints: []v1.SizeConstraint{
			{
				Type: metal.CoreConstraint,
				Min:  15,
				Max:  27,
			},
			{
				Type: metal.MemoryConstraint,
				Min:  100,
				Max:  100,
			},
		},
	}
	js, err := json.Marshal(createRequest)
	require.NoError(t, err)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("PUT", "/v1/size", body)
	req.Header.Add("Content-Type", "application/json")
	container = injectAdmin(log, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode, w.Body.String())
	var result v1.SizeResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.Sz1.ID, result.ID)
	require.Equal(t, testdata.Sz1.Name, *result.Name)
	require.Equal(t, testdata.Sz1.Description, *result.Description)
}

func TestUpdateSize(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	psc := &mdmv1mock.ProjectServiceClient{}
	psc.On("Find", testifymock.Anything, &mdmv1.ProjectFindRequest{}).Return(&mdmv1.ProjectListResponse{Projects: []*mdmv1.Project{
		{Meta: &mdmv1.Meta{Id: "p1"}},
	}}, nil)
	mdc := mdm.NewMock(psc, &mdmv1mock.TenantServiceClient{}, nil, nil)

	sizeservice := NewSize(log, ds, mdc)
	container := restful.NewContainer().Add(sizeservice)

	minCores := uint64(8)
	maxCores := uint64(16)
	updateRequest := v1.SizeUpdateRequest{
		Common: v1.Common{
			Describable: v1.Describable{
				Name:        &testdata.Sz2.Name,
				Description: &testdata.Sz2.Description,
			},
			Identifiable: v1.Identifiable{
				ID: testdata.Sz1.ID,
			},
		},
		SizeConstraints: &[]v1.SizeConstraint{
			{
				Type: metal.CoreConstraint,
				Min:  minCores,
				Max:  maxCores,
			},
		},
	}
	js, err := json.Marshal(updateRequest)
	require.NoError(t, err)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/size", body)
	req.Header.Add("Content-Type", "application/json")
	container = injectAdmin(log, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.SizeResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.Sz1.ID, result.ID)
	require.Equal(t, testdata.Sz2.Name, *result.Name)
	require.Equal(t, testdata.Sz2.Description, *result.Description)
	require.Equal(t, metal.CoreConstraint, result.SizeConstraints[0].Type)
	require.Equal(t, minCores, result.SizeConstraints[0].Min)
	require.Equal(t, maxCores, result.SizeConstraints[0].Max)
}

func TestListSizeReservationsUsage(t *testing.T) {
	tests := []struct {
		name          string
		req           *v1.SizeReservationListRequest
		dbMockFn      func(mock *r.Mock)
		projectMockFn func(mock *testifymock.Mock)
		want          []*v1.SizeReservationUsageResponse
	}{
		{
			name: "list reservations usage",
			req: &v1.SizeReservationListRequest{
				SizeID:      pointer.Pointer("1"),
				ProjectID:   pointer.Pointer("p1"),
				PartitionID: pointer.Pointer("a"),
			},
			dbMockFn: func(mock *r.Mock) {
				mock.On(r.DB("mockdb").Table("sizereservation").Filter(r.MockAnything()).Filter(r.MockAnything()).Filter(r.MockAnything())).Return(metal.SizeReservations{
					{
						Base: metal.Base{
							ID: "1",
						},
						SizeID:       "1",
						Amount:       3,
						PartitionIDs: []string{"a"},
						ProjectID:    "p1",
					},
				}, nil)
				mock.On(r.DB("mockdb").Table("machine").Filter(r.MockAnything())).Return(metal.Machines{
					{
						Base: metal.Base{
							ID: "1",
						},
						SizeID:      "1",
						PartitionID: "a",
						Allocation: &metal.MachineAllocation{
							Project: "p1",
						},
					},
				}, nil)
			},
			want: []*v1.SizeReservationUsageResponse{
				{
					Common: v1.Common{
						Identifiable: v1.Identifiable{
							ID: "1",
						},
						Describable: v1.Describable{
							Name:        pointer.Pointer(""),
							Description: pointer.Pointer(""),
						},
					},
					SizeID:             "1",
					PartitionID:        "a",
					ProjectID:          "p1",
					Amount:             3,
					UsedAmount:         1,
					ProjectAllocations: 1,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				projectMock = mdmv1mock.NewProjectServiceClient(t)
				m           = mdm.NewMock(projectMock, nil, nil, nil)
				ds, dbMock  = datastore.InitMockDB(t)
				ws          = NewSize(slog.Default(), ds, m)
			)

			if tt.dbMockFn != nil {
				tt.dbMockFn(dbMock)
			}
			if tt.projectMockFn != nil {
				tt.projectMockFn(&projectMock.Mock)
			}

			code, got := genericWebRequest[[]*v1.SizeReservationUsageResponse](t, ws, testViewUser, tt.req, "POST", "/v1/size/reservations/usage")
			assert.Equal(t, http.StatusOK, code)

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("diff (-want +got):\n%s", diff)
			}
		})
	}

}

func Test_longestCommonPrefix(t *testing.T) {
	tests := []struct {
		name string
		strs []string
		want string
	}{
		{
			name: "no strings",
			strs: nil,
			want: "",
		},
		{
			name: "single string",
			strs: []string{"foo"},
			want: "foo",
		},
		{
			name: "two same strings",
			strs: []string{"foo", "foo"},
			want: "foo",
		},
		{
			name: "one string is longer",
			strs: []string{"foo", "foobar", "foo"},
			want: "foo*",
		},
		{
			name: "no common prefix",
			strs: []string{"foo", "bar"},
			want: "*",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := longestCommonPrefix(tt.strs); got != tt.want {
				t.Errorf("longestCommonPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}
