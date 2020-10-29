package service

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/metal-stack/metal-lib/bus"
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
	goipam "github.com/metal-stack/go-ipam"
	"github.com/stretchr/testify/require"

	restful "github.com/emicklei/go-restful/v3"
)

func TestGetIPs(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	ipservice, err := NewIP(ds, nil, bus.DirectEndpoints(), ipam.New(goipam.New()), nil)
	require.NoError(t, err)

	container := restful.NewContainer().Add(ipservice)
	req := httptest.NewRequest("GET", "/v1/ip", nil)
	container = injectViewer(container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result []v1.IPResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Len(t, result, 3)
	require.Equal(t, testdata.IP1.IPAddress, result[0].IPAddress)
	require.Equal(t, testdata.IP1.Name, *result[0].Name)
	require.Equal(t, testdata.IP2.IPAddress, result[1].IPAddress)
	require.Equal(t, testdata.IP2.Name, *result[1].Name)
	require.Equal(t, testdata.IP3.IPAddress, result[2].IPAddress)
	require.Equal(t, testdata.IP3.Name, *result[2].Name)
}

func TestGetIP(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	ipservice, err := NewIP(ds, nil, bus.DirectEndpoints(), ipam.New(goipam.New()), nil)
	require.NoError(t, err)
	container := restful.NewContainer().Add(ipservice)
	req := httptest.NewRequest("GET", "/v1/ip/1.2.3.4", nil)
	container = injectViewer(container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.IPResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, testdata.IP1.IPAddress, result.IPAddress)
	require.Equal(t, testdata.IP1.Name, *result.Name)
}

func TestGetIPNotFound(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	ipservice, err := NewIP(ds, nil, bus.DirectEndpoints(), ipam.New(goipam.New()), nil)
	require.NoError(t, err)
	container := restful.NewContainer().Add(ipservice)
	req := httptest.NewRequest("GET", "/v1/ip/9.9.9.9", nil)
	container = injectViewer(container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusNotFound, resp.StatusCode, w.Body.String())
	var result httperrors.HTTPErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Contains(t, result.Message, "9.9.9.9")
	require.Equal(t, 404, result.StatusCode)
}

func TestDeleteIP(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	ipamer, err := testdata.InitMockIpamData(mock, true)
	require.Nil(t, err)
	testdata.InitMockDBData(mock)

	ipservice, err := NewIP(ds, nil, bus.DirectEndpoints(), ipamer, nil)
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
			wantedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:         "free an cluster-ip should fail",
			ip:           testdata.IP2.IPAddress,
			wantedStatus: http.StatusUnprocessableEntity,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/ip/free/"+testdata.IPAMIP.IPAddress, nil)
			container = injectEditor(container, req)
			req.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()
			container.ServeHTTP(w, req)

			resp := w.Result()
			require.Equal(t, tt.wantedStatus, resp.StatusCode, w.Body.String())
			var result v1.IPResponse
			err = json.NewDecoder(resp.Body).Decode(&result)

			require.Nil(t, err)
		})
	}
}

func TestAllocateIP(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	ipamer, err := testdata.InitMockIpamData(mock, false)
	require.Nil(t, err)
	testdata.InitMockDBData(mock)

	psc := mdmock.ProjectServiceClient{}
	psc.On("Get", context.Background(), &mdmv1.ProjectGetRequest{Id: "123"}).Return(&mdmv1.ProjectResponse{
		Project: &mdmv1.Project{
			Meta: &mdmv1.Meta{Id: "project-1"},
		},
	}, nil,
	)
	tsc := mdmock.TenantServiceClient{}

	mdc := mdm.NewMock(&psc, &tsc)

	ipservice, err := NewIP(ds, nil, bus.DirectEndpoints(), ipamer, mdc)
	require.NoError(t, err)
	container := restful.NewContainer().Add(ipservice)

	tests := []struct {
		name            string
		allocateRequest v1.IPAllocateRequest
		wantedStatus    int
		wantedType      metal.IPType
		wantedIP        string
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.allocateRequest.Describable.Name = &tt.name
			js, _ := json.Marshal(tt.allocateRequest)
			body := bytes.NewBuffer(js)
			req := httptest.NewRequest("POST", "/v1/ip/allocate", body)
			container = injectEditor(container, req)
			req.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()
			container.ServeHTTP(w, req)
			resp := w.Result()

			require.Equal(t, tt.wantedStatus, resp.StatusCode, w.Body.String())
			var result v1.IPResponse
			err = json.NewDecoder(resp.Body).Decode(&result)

			require.Nil(t, err)
			require.Equal(t, tt.wantedType, result.Type)
			require.Equal(t, tt.wantedIP, result.IPAddress)
			require.Equal(t, tt.name, *result.Name)
		})
	}
}

func TestUpdateIP(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	ipservice, err := NewIP(ds, nil, bus.DirectEndpoints(), ipam.New(goipam.New()), nil)
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
				IPIdentifiable: v1.IPIdentifiable{
					IPAddress: testdata.IP1.IPAddress,
				},
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
				IPIdentifiable: v1.IPIdentifiable{
					IPAddress: testdata.IP1.IPAddress,
				},
				Type: "static",
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
				IPIdentifiable: v1.IPIdentifiable{
					IPAddress: testdata.IP2.IPAddress,
				},
				Type: "ephemeral",
			},
			wantedStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "internal tag machine is allowed",
			updateRequest: v1.IPUpdateRequest{
				IPIdentifiable: v1.IPIdentifiable{
					IPAddress: testdata.IP3.IPAddress,
				},
				Type: "static",
				Tags: []string{machineIDTag1},
			},
			wantedStatus: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			js, _ := json.Marshal(tt.updateRequest)
			body := bytes.NewBuffer(js)
			req := httptest.NewRequest("POST", "/v1/ip", body)
			container = injectEditor(container, req)
			req.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()
			container.ServeHTTP(w, req)

			resp := w.Result()
			require.Equal(t, tt.wantedStatus, resp.StatusCode, w.Body.String())
			var result v1.IPResponse
			err := json.NewDecoder(resp.Body).Decode(&result)

			require.Nil(t, err)
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
		name    string
		tags    []string
		wanted  []string
		wantErr bool
	}{
		{
			name:   "distinct and sorted",
			tags:   []string{"2", "1", "2"},
			wanted: []string{"1", "2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := processTags(tt.tags)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !cmp.Equal(got, tt.wanted) {
				t.Errorf("%v", cmp.Diff(got, tt.wanted))
			}
		})
	}
}
