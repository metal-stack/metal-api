package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/metal-stack/metal-lib/bus"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"github.com/metal-stack/metal-lib/pkg/tag"

	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	mdmock "github.com/metal-stack/masterdata-api/api/v1/mocks"
	mdm "github.com/metal-stack/masterdata-api/pkg/client"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/ipam"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"

	"github.com/metal-stack/metal-lib/httperrors"

	"github.com/google/go-cmp/cmp"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	restful "github.com/emicklei/go-restful/v3"
)

func TestGetIPs(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)

	logger := slog.Default()
	ipservice, err := NewIP(logger, ds, bus.DirectEndpoints(), ipam.InitTestIpam(t), nil)
	require.NoError(t, err)

	container := restful.NewContainer().Add(ipservice)
	req := httptest.NewRequest("GET", "/v1/ip", nil)
	container = injectViewer(logger, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result []v1.IPResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Len(t, result, 4)
	require.Equal(t, testdata.IP1.IPAddress, result[0].IPAddress)
	require.Equal(t, testdata.IP1.Name, *result[0].Name)
	require.Equal(t, testdata.IP2.IPAddress, result[1].IPAddress)
	require.Equal(t, testdata.IP2.Name, *result[1].Name)
	require.Equal(t, testdata.IP3.IPAddress, result[2].IPAddress)
	require.Equal(t, testdata.IP3.Name, *result[2].Name)
	require.Equal(t, testdata.IP4.IPAddress, result[3].IPAddress)
	require.Equal(t, testdata.IP4.Name, *result[3].Name)
}

func TestGetIP(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)

	logger := slog.Default()
	ipservice, err := NewIP(logger, ds, bus.DirectEndpoints(), ipam.InitTestIpam(t), nil)
	require.NoError(t, err)
	container := restful.NewContainer().Add(ipservice)
	req := httptest.NewRequest("GET", "/v1/ip/1.2.3.4", nil)
	container = injectViewer(logger, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.IPResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.IP1.IPAddress, result.IPAddress)
	require.Equal(t, testdata.IP1.Name, *result.Name)
}

func TestGetIPv6(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)

	logger := slog.Default()
	ipservice, err := NewIP(logger, ds, bus.DirectEndpoints(), ipam.InitTestIpam(t), nil)
	require.NoError(t, err)
	container := restful.NewContainer().Add(ipservice)
	req := httptest.NewRequest("GET", "/v1/ip/2001:0db8:85a3::1", nil)
	container = injectViewer(logger, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.IPResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.IP4.IPAddress, result.IPAddress)
	require.Equal(t, testdata.IP4.Name, *result.Name)
}

func TestGetIPNotFound(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	logger := slog.Default()

	ipservice, err := NewIP(logger, ds, bus.DirectEndpoints(), ipam.InitTestIpam(t), nil)
	require.NoError(t, err)
	container := restful.NewContainer().Add(ipservice)
	req := httptest.NewRequest("GET", "/v1/ip/9.9.9.9", nil)
	container = injectViewer(logger, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode, w.Body.String())
	var result httperrors.HTTPErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Contains(t, result.Message, "9.9.9.9")
	require.Equal(t, 404, result.StatusCode)
}

func TestDeleteIP(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	ipamer, err := testdata.InitMockIpamData(mock, true)
	require.NoError(t, err)
	testdata.InitMockDBData(mock)
	logger := slog.Default()

	ipservice, err := NewIP(logger, ds, bus.DirectEndpoints(), ipamer, nil)
	require.NoError(t, err)
	container := restful.NewContainer().Add(ipservice)

	tests := []struct {
		name         string
		ip           string
		wantedStatus int
	}{
		{
			name:         "free an ip",
			ip:           testdata.IPAMIP.IPAddress,
			wantedStatus: http.StatusOK,
		},
		{
			name:         "free an machine-ip should fail",
			ip:           testdata.IP3.IPAddress,
			wantedStatus: http.StatusBadRequest,
		},
		{
			name:         "free an cluster-ip should fail",
			ip:           testdata.IP2.IPAddress,
			wantedStatus: http.StatusNotFound,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", "/v1/ip/free/"+tt.ip, nil)
			container = injectEditor(logger, container, req)
			req.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()
			container.ServeHTTP(w, req)

			resp := w.Result()
			require.Equal(t, tt.wantedStatus, resp.StatusCode, w.Body.String())
			defer resp.Body.Close()

			if tt.wantedStatus != 200 {
				return
			}

			var result v1.IPResponse
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)
		})
	}
}

func TestAllocateIP(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	ipamer, err := testdata.InitMockIpamData(mock, false)
	require.NoError(t, err)
	testdata.InitMockDBData(mock)
	logger := slog.Default()

	psc := mdmock.ProjectServiceClient{}
	psc.On("Get", testifymock.Anything, &mdmv1.ProjectGetRequest{Id: "123"}).Return(&mdmv1.ProjectResponse{
		Project: &mdmv1.Project{
			Meta: &mdmv1.Meta{Id: "project-1"},
		},
	}, nil,
	)
	tsc := mdmock.TenantServiceClient{}

	mdc := mdm.NewMock(&psc, &tsc, nil, nil)

	ipservice, err := NewIP(logger, ds, bus.DirectEndpoints(), ipamer, mdc)
	require.NoError(t, err)
	container := restful.NewContainer().Add(ipservice)

	tests := []struct {
		name            string
		allocateRequest v1.IPAllocateRequest
		specificIP      string
		wantedStatus    int
		wantedType      metal.IPType
		wantedIP        string
		wantErr         error
	}{
		{
			name: "allocate an ephemeral ip",
			allocateRequest: v1.IPAllocateRequest{
				Describable: v1.Describable{},
				IPBase:      v1.IPBase{ProjectID: "123", NetworkID: testdata.NwIPAM.ID, Type: metal.Ephemeral},
			},
			wantedStatus: http.StatusCreated,
			wantedType:   metal.Ephemeral,
			wantedIP:     "10.0.0.1",
		},
		{
			name: "allocate a static ip",
			allocateRequest: v1.IPAllocateRequest{
				Describable: v1.Describable{},
				IPBase:      v1.IPBase{ProjectID: "123", NetworkID: testdata.NwIPAM.ID, Type: metal.Static},
			},
			wantedStatus: http.StatusCreated,
			wantedType:   metal.Static,
			wantedIP:     "10.0.0.2",
		},
		{
			name: "allocate a static specific ip",
			allocateRequest: v1.IPAllocateRequest{
				Describable: v1.Describable{},
				IPBase: v1.IPBase{
					ProjectID: "123",
					NetworkID: testdata.NwIPAM.ID,
					Type:      metal.Static,
				},
			},
			specificIP:   "10.0.0.5",
			wantedStatus: http.StatusCreated,
			wantedType:   metal.Static,
			wantedIP:     "10.0.0.5",
		},
		{
			name: "allocate a specific ip which is already allocated",
			allocateRequest: v1.IPAllocateRequest{
				Describable: v1.Describable{},
				IPBase: v1.IPBase{
					ProjectID: "123",
					NetworkID: testdata.NwIPAM.ID,
				},
			},
			specificIP:   "10.0.0.5",
			wantedStatus: http.StatusConflict,
			wantErr:      errors.New("Conflict ip already allocated"),
		},
		{
			name: "allocate a static specific ip outside prefix",
			allocateRequest: v1.IPAllocateRequest{
				Describable: v1.Describable{},
				IPBase: v1.IPBase{
					ProjectID: "123",
					NetworkID: testdata.NwIPAM.ID,
					Type:      metal.Static,
				},
			},
			specificIP:   "11.0.0.5",
			wantedStatus: http.StatusUnprocessableEntity,
			wantErr:      errors.New("specific ip not contained in any of the defined prefixes"),
		},
		{
			name: "allocate a IPv4 address",
			allocateRequest: v1.IPAllocateRequest{
				Describable: v1.Describable{},
				IPBase: v1.IPBase{
					ProjectID: "123",
					NetworkID: testdata.NwIPAM.ID,
					Type:      metal.Ephemeral,
				},
				AddressFamily: pointer.Pointer(metal.IPv4AddressFamily),
			},
			wantedIP:     "10.0.0.3",
			wantedType:   metal.Ephemeral,
			wantedStatus: http.StatusCreated,
		},
		{
			name: "allocate a IPv6 address",
			allocateRequest: v1.IPAllocateRequest{
				Describable: v1.Describable{},
				IPBase: v1.IPBase{
					ProjectID: "123",
					NetworkID: testdata.NwIPAM.ID,
					Type:      metal.Ephemeral,
				},
				AddressFamily: pointer.Pointer(metal.IPv6AddressFamily),
			},
			wantedStatus: http.StatusBadRequest,
			wantErr:      errors.New("there is no prefix for the given addressfamily:IPv6 present in network:4"),
		},
		{
			name: "allocate a IPv4 (no addressfamily specified) address from a IPv6 Only network",
			allocateRequest: v1.IPAllocateRequest{
				Describable: v1.Describable{},
				IPBase: v1.IPBase{
					ProjectID: "123",
					NetworkID: testdata.Partition2PrivateSuperNetworkV6.ID,
					Type:      metal.Ephemeral,
				},
			},
			wantedIP:     "2001::1",
			wantedType:   metal.Ephemeral,
			wantedStatus: http.StatusCreated,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			tt.allocateRequest.Describable.Name = &tt.name
			js, err := json.Marshal(tt.allocateRequest)
			require.NoError(t, err)
			body := bytes.NewBuffer(js)
			var req *http.Request
			if tt.specificIP == "" {
				req = httptest.NewRequest("POST", "/v1/ip/allocate", body)
			} else {
				req = httptest.NewRequest("POST", "/v1/ip/allocate/"+tt.specificIP, body)
			}
			container = injectEditor(logger, container, req)
			req.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()
			container.ServeHTTP(w, req)
			resp := w.Result()
			defer resp.Body.Close()

			require.Equal(t, tt.wantedStatus, resp.StatusCode, w.Body.String())

			if tt.wantErr == nil {
				var result v1.IPResponse
				err = json.NewDecoder(resp.Body).Decode(&result)

				require.NoError(t, err)
				require.NotNil(t, result.IPAddress)
				require.NotNil(t, result.AllocationUUID)
				require.Equal(t, tt.wantedType, result.Type)
				require.Equal(t, tt.wantedIP, result.IPAddress)
				require.Equal(t, tt.name, *result.Name)
			} else {
				var result httperrors.HTTPErrorResponse
				err = json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)
				require.Equal(t, tt.wantedStatus, resp.StatusCode)
				require.Equal(t, tt.wantErr.Error(), result.Message)
			}
		})
	}
}

func TestUpdateIP(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	logger := slog.Default()

	ipservice, err := NewIP(logger, ds, bus.DirectEndpoints(), ipam.InitTestIpam(t), nil)
	require.NoError(t, err)
	container := restful.NewContainer().Add(ipservice)
	machineIDTag1 := tag.MachineID + "=" + "1"
	tests := []struct {
		name                 string
		updateRequest        v1.IPUpdateRequest
		wantedStatus         int
		wantedIPIdentifiable *v1.IPIdentifiable
		wantedIPBase         *v1.IPBase
		wantedDescribable    *v1.Describable
	}{
		{
			name: "update ip name",
			updateRequest: v1.IPUpdateRequest{
				Describable: v1.Describable{
					Name:        &testdata.IP2.Name,
					Description: &testdata.IP2.Description,
				},
				IPAddress: testdata.IP1.IPAddress,
			},
			wantedStatus: http.StatusOK,
			wantedDescribable: &v1.Describable{
				Name:        &testdata.IP2.Name,
				Description: &testdata.IP2.Description,
			},
		},
		{
			name: "moving from ephemeral to static",
			updateRequest: v1.IPUpdateRequest{
				IPAddress: testdata.IP1.IPAddress,
				Type:      "static",
			},
			wantedStatus: http.StatusOK,
			wantedIPBase: &v1.IPBase{
				ProjectID: testdata.IP1.ProjectID,
				Type:      "static",
				Tags:      []string{},
			},
		},
		{
			name: "moving from static to ephemeral must not be allowed",
			updateRequest: v1.IPUpdateRequest{
				IPAddress: testdata.IP2.IPAddress,
				Type:      "ephemeral",
			},
			wantedStatus: http.StatusBadRequest,
		},
		{
			name: "internal tag machine is allowed",
			updateRequest: v1.IPUpdateRequest{
				IPAddress: testdata.IP3.IPAddress,
				Type:      "static",
				Tags:      []string{machineIDTag1},
			},
			wantedStatus: http.StatusOK,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			js, err := json.Marshal(tt.updateRequest)
			require.NoError(t, err)
			body := bytes.NewBuffer(js)
			req := httptest.NewRequest("POST", "/v1/ip", body)
			container = injectEditor(logger, container, req)
			req.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()
			container.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()
			require.Equal(t, tt.wantedStatus, resp.StatusCode, w.Body.String())
			var result v1.IPResponse
			err = json.NewDecoder(resp.Body).Decode(&result)

			if tt.wantedStatus != 200 {
				return
			}
			require.NoError(t, err)

			if tt.wantedIPIdentifiable != nil {
				require.Equal(t, *tt.wantedIPIdentifiable, result.IPIdentifiable)
			}
			if tt.wantedIPBase != nil {
				require.Equal(t, *tt.wantedIPBase, result.IPBase)
			}
			if tt.wantedDescribable != nil {
				require.Equal(t, *tt.wantedDescribable, result.Describable)
			}
		})
	}
}

func TestProcessTags(t *testing.T) {
	tests := []struct {
		name   string
		tags   []string
		wanted []string
	}{
		{
			name:   "distinct and sorted",
			tags:   []string{"2", "1", "2"},
			wanted: []string{"1", "2"},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got := processTags(tt.tags)
			if !cmp.Equal(got, tt.wanted) {
				t.Errorf("%v", cmp.Diff(got, tt.wanted))
			}
		})
	}
}
