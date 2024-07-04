package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	mdmv1mock "github.com/metal-stack/masterdata-api/api/v1/mocks"
	mdm "github.com/metal-stack/masterdata-api/pkg/client"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/ipam"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"

	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetNetworks(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	networkservice := NewNetwork(log, ds, ipam.InitTestIpam(t), nil)
	container := restful.NewContainer().Add(networkservice)
	req := httptest.NewRequest("GET", "/v1/network", nil)
	container = injectViewer(log, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result []v1.NetworkResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Len(t, result, 4)
	require.Equal(t, testdata.Nw1.ID, result[0].ID)
	require.Equal(t, testdata.Nw1.Name, *result[0].Name)
	require.Equal(t, testdata.Nw2.ID, result[1].ID)
	require.Equal(t, testdata.Nw2.Name, *result[1].Name)
	require.Equal(t, testdata.Nw3.ID, result[2].ID)
	require.Equal(t, testdata.Nw3.Name, *result[2].Name)
}

func TestGetNetwork(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	networkservice := NewNetwork(log, ds, ipam.InitTestIpam(t), nil)
	container := restful.NewContainer().Add(networkservice)
	req := httptest.NewRequest("GET", "/v1/network/1", nil)
	container = injectViewer(log, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.NetworkResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.Nw1.ID, result.ID)
	require.Equal(t, testdata.Nw1.Name, *result.Name)
}

func TestGetNetworkNotFound(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	networkservice := NewNetwork(log, ds, ipam.InitTestIpam(t), nil)
	container := restful.NewContainer().Add(networkservice)
	req := httptest.NewRequest("GET", "/v1/network/999", nil)
	container = injectViewer(log, container, req)
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

func TestDeleteNetwork(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	mock.On(r.DB("mockdb").Table("network").Filter(r.MockAnything())).Return([]interface{}{}, nil)
	ipamer, err := testdata.InitMockIpamData(mock, false)
	require.NoError(t, err)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	networkservice := NewNetwork(log, ds, ipamer, nil)
	container := restful.NewContainer().Add(networkservice)
	req := httptest.NewRequest("DELETE", "/v1/network/"+testdata.NwIPAM.ID, nil)
	container = injectAdmin(log, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.NetworkResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.NwIPAM.ID, result.ID)
	require.Equal(t, testdata.NwIPAM.Name, *result.Name)
}

func TestDeleteNetworkIPInUse(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	mock.On(r.DB("mockdb").Table("network").Filter(r.MockAnything())).Return([]interface{}{}, nil)
	ipamer, err := testdata.InitMockIpamData(mock, true)
	require.NoError(t, err)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	networkservice := NewNetwork(log, ds, ipamer, nil)
	container := restful.NewContainer().Add(networkservice)
	req := httptest.NewRequest("DELETE", "/v1/network/"+testdata.NwIPAM.ID, nil)
	container = injectAdmin(log, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode, w.Body.String())
	var result httperrors.HTTPErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, 422, result.StatusCode)
	require.Contains(t, result.Message, "unable to delete network: prefix 10.0.0.0/16 has ip 10.0.0.1 in use")
}

func TestCreateNetwork(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	ipamer, err := testdata.InitMockIpamData(mock, false)
	require.NoError(t, err)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	networkservice := NewNetwork(log, ds, ipamer, nil)
	container := restful.NewContainer().Add(networkservice)

	prefixes := []string{"172.0.0.0/24"}
	destPrefixes := []string{"0.0.0.0/0"}
	vrf := uint(10000)
	createRequest := &v1.NetworkCreateRequest{
		Describable:      v1.Describable{Name: &testdata.Nw1.Name},
		NetworkBase:      v1.NetworkBase{PartitionID: &testdata.Nw1.PartitionID, ProjectID: &testdata.Nw1.ProjectID},
		NetworkImmutable: v1.NetworkImmutable{Prefixes: prefixes, DestinationPrefixes: destPrefixes, Vrf: &vrf},
	}
	js, err := json.Marshal(createRequest)
	require.NoError(t, err)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("PUT", "/v1/network", body)
	req.Header.Add("Content-Type", "application/json")
	container = injectAdmin(log, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode, w.Body.String())
	var result v1.NetworkResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.Nw1.Name, *result.Name)
	require.Equal(t, testdata.Nw1.PartitionID, *result.PartitionID)
	require.Equal(t, testdata.Nw1.ProjectID, *result.ProjectID)
	require.Equal(t, destPrefixes, result.DestinationPrefixes)
}

func TestUpdateNetwork(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	networkservice := NewNetwork(log, ds, ipam.InitTestIpam(t), nil)
	container := restful.NewContainer().Add(networkservice)

	newName := "new"
	updateRequest := &v1.NetworkUpdateRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{ID: testdata.Nw1.GetID()},
			Describable:  v1.Describable{Name: &newName},
		},
	}
	js, err := json.Marshal(updateRequest)
	require.NoError(t, err)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/network", body)
	req.Header.Add("Content-Type", "application/json")
	container = injectAdmin(log, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result metal.Partition
	err = json.NewDecoder(resp.Body).Decode(&result)

	require.NoError(t, err)
	require.Equal(t, testdata.Nw1.ID, result.ID)
	require.Equal(t, newName, result.Name)
}

func TestSearchNetwork(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	mock.On(r.DB("mockdb").Table("network").Filter(r.MockAnything())).Return([]interface{}{testdata.Nw1}, nil)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	networkService := NewNetwork(log, ds, ipam.InitTestIpam(t), nil)
	container := restful.NewContainer().Add(networkService)
	requestJSON := fmt.Sprintf("{%q:%q}", "partitionid", "1")
	req := httptest.NewRequest("POST", "/v1/network/find", bytes.NewBufferString(requestJSON))
	req.Header.Add("Content-Type", "application/json")
	container = injectViewer(log, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var results []v1.NetworkResponse
	err := json.NewDecoder(resp.Body).Decode(&results)

	require.NoError(t, err)
	require.Len(t, results, 1)
	result := results[0]
	require.Equal(t, testdata.Nw1.ID, result.ID)
	require.Equal(t, testdata.Nw1.PartitionID, *result.PartitionID)
	require.Equal(t, testdata.Nw1.Name, *result.Name)
}

func Test_networkResource_createNetwork(t *testing.T) {
	log := slog.Default()
	tests := []struct {
		name                 string
		networkName          string
		networkID            string
		partitionID          string
		projectID            string
		prefixes             []string
		destinationPrefixes  []string
		vrf                  uint
		childprefixlength    *uint8
		privateSuper         bool
		underlay             bool
		nat                  bool
		expectedStatus       int
		expectedErrorMessage string
	}{
		{
			name:                "simple IPv4",
			networkName:         testdata.Nw1.Name,
			partitionID:         testdata.Nw1.PartitionID,
			projectID:           testdata.Nw1.ProjectID,
			prefixes:            []string{"172.0.0.0/24"},
			destinationPrefixes: []string{"0.0.0.0/0"},
			vrf:                 uint(10000),
			expectedStatus:      http.StatusCreated,
		},
		{
			name:                 "privatesuper IPv4",
			networkName:          testdata.Nw1.Name,
			partitionID:          testdata.Nw1.PartitionID,
			projectID:            testdata.Nw1.ProjectID,
			prefixes:             []string{"172.0.0.0/24"},
			destinationPrefixes:  []string{"0.0.0.0/0"},
			privateSuper:         true,
			vrf:                  uint(10000),
			expectedStatus:       http.StatusBadRequest,
			expectedErrorMessage: "partition with id \"1\" already has a private super network for this addressfamily",
		},
		{
			name:                "privatesuper IPv6",
			networkName:         testdata.Nw1.Name,
			partitionID:         testdata.Nw1.PartitionID,
			projectID:           testdata.Nw1.ProjectID,
			prefixes:            []string{"fdaa:bbcc::/50"},
			destinationPrefixes: []string{"::/0"},
			privateSuper:        true,
			vrf:                 uint(10000),
			expectedStatus:      http.StatusCreated,
		},
		{
			name:                 "broken IPv4",
			networkName:          testdata.Nw1.Name,
			partitionID:          testdata.Nw1.PartitionID,
			projectID:            testdata.Nw1.ProjectID,
			prefixes:             []string{"192.168.265.0/24"},
			destinationPrefixes:  []string{"0.0.0.0/0"},
			privateSuper:         true,
			vrf:                  uint(10000),
			expectedStatus:       http.StatusBadRequest,
			expectedErrorMessage: "given prefix 192.168.265.0/24 is not a valid ip with mask: netip.ParsePrefix(\"192.168.265.0/24\"): ParseAddr(\"192.168.265.0\"): IPv4 field has value >255",
		},
		{
			name:                 "broken IPv6",
			networkName:          testdata.Nw1.Name,
			partitionID:          testdata.Nw1.PartitionID,
			projectID:            testdata.Nw1.ProjectID,
			prefixes:             []string{"fdaa:::/50"},
			destinationPrefixes:  []string{"::/0"},
			privateSuper:         true,
			vrf:                  uint(10000),
			expectedStatus:       http.StatusBadRequest,
			expectedErrorMessage: "given prefix fdaa:::/50 is not a valid ip with mask: netip.ParsePrefix(\"fdaa:::/50\"): ParseAddr(\"fdaa:::\"): each colon-separated field must have at least one digit (at \":\")",
		},
		{
			name:                 "mixed prefix addressfamilies",
			networkName:          testdata.Nw1.Name,
			partitionID:          testdata.Nw1.PartitionID,
			projectID:            testdata.Nw1.ProjectID,
			prefixes:             []string{"172.0.0.0/24", "fdaa:bbcc::/50"},
			destinationPrefixes:  []string{"0.0.0.0/0"},
			vrf:                  uint(10000),
			expectedStatus:       http.StatusBadRequest,
			expectedErrorMessage: "given prefixes have different addressfamilies",
		},
		{
			name:                 "broken destinationprefix",
			networkName:          testdata.Nw1.Name,
			partitionID:          testdata.Nw1.PartitionID,
			projectID:            testdata.Nw1.ProjectID,
			prefixes:             []string{"172.0.0.0/24"},
			destinationPrefixes:  []string{"0.0.0.0/33"},
			vrf:                  uint(10000),
			expectedStatus:       http.StatusBadRequest,
			expectedErrorMessage: "given prefix 0.0.0.0/33 is not a valid ip with mask: netip.ParsePrefix(\"0.0.0.0/33\"): prefix length out of range",
		},
		{
			name:                 "broken childprefixlength",
			networkName:          testdata.Nw1.Name,
			partitionID:          testdata.Nw1.PartitionID,
			projectID:            testdata.Nw1.ProjectID,
			prefixes:             []string{"fdaa:bbcc::/50"},
			childprefixlength:    pointer.Pointer(uint8(50)),
			privateSuper:         true,
			vrf:                  uint(10000),
			expectedStatus:       http.StatusBadRequest,
			expectedErrorMessage: "given childprefixlength 50 is not greater than prefix length of:fdaa:bbcc::/50",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds, mock := datastore.InitMockDB(t)
			ipamer, err := testdata.InitMockIpamData(mock, false)
			require.NoError(t, err)
			testdata.InitMockDBData(mock)

			networkservice := NewNetwork(log, ds, ipamer, nil)
			container := restful.NewContainer().Add(networkservice)

			createRequest := &v1.NetworkCreateRequest{
				Describable: v1.Describable{Name: &tt.networkName},
				NetworkBase: v1.NetworkBase{PartitionID: &tt.partitionID, ProjectID: &tt.projectID},
				NetworkImmutable: v1.NetworkImmutable{
					Prefixes:            tt.prefixes,
					DestinationPrefixes: tt.destinationPrefixes,
					Vrf:                 &tt.vrf, Nat: tt.nat, PrivateSuper: tt.privateSuper, Underlay: tt.underlay,
				},
			}
			if tt.childprefixlength != nil {
				createRequest.ChildPrefixLength = tt.childprefixlength
			}
			js, _ := json.Marshal(createRequest)
			body := bytes.NewBuffer(js)
			req := httptest.NewRequest("PUT", "/v1/network", body)
			req.Header.Add("Content-Type", "application/json")
			container = injectAdmin(log, container, req)
			w := httptest.NewRecorder()
			container.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			require.Equal(t, tt.expectedStatus, resp.StatusCode, w.Body.String())
			if tt.expectedStatus > 300 {
				var result httperrors.HTTPErrorResponse
				err := json.NewDecoder(resp.Body).Decode(&result)

				require.NoError(t, err)
				require.Equal(t, tt.expectedErrorMessage, result.Message)
			} else {
				var result v1.NetworkResponse
				err = json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)
				require.Equal(t, tt.networkName, *result.Name)
				require.Equal(t, tt.partitionID, *result.PartitionID)
				require.Equal(t, tt.projectID, *result.ProjectID)
				require.Equal(t, tt.destinationPrefixes, result.DestinationPrefixes)
				if tt.childprefixlength != nil {
					require.Equal(t, tt.childprefixlength, result.ChildPrefixLength)
				}
			}
		})
	}
}

func Test_networkResource_allocateNetwork(t *testing.T) {
	log := slog.Default()
	tests := []struct {
		name                 string
		networkName          string
		partitionID          string
		projectID            string
		childprefixlength    *uint8
		addressFamily        *string
		shared               bool
		expectedStatus       int
		expectedErrorMessage string
	}{
		{
			name:           "simple ipv4, default childprefixlength",
			networkName:    "tenantv4",
			partitionID:    testdata.Partition1.ID,
			projectID:      "project-1",
			expectedStatus: http.StatusCreated,
		},
		{
			name:              "simple ipv4, specific childprefixlength",
			networkName:       "tenantv4.2",
			partitionID:       testdata.Partition1.ID,
			projectID:         "project-1",
			childprefixlength: pointer.Pointer(uint8(29)),
			expectedStatus:    http.StatusCreated,
		},
		{
			name:                 "ipv6 without ipv6 super",
			networkName:          "tenantv6",
			partitionID:          testdata.Partition1.ID,
			projectID:            "project-1",
			addressFamily:        pointer.Pointer("ipv6"),
			expectedStatus:       http.StatusBadRequest,
			expectedErrorMessage: "no supernetwork for addressfamily:IPv6 found",
		},
	}
	for _, tt := range tests {
		ds, mock := datastore.InitMockDB(t)

		supernetwork := testdata.Nw1
		ipamer, err := testdata.InitMockIpamData(mock, false)
		require.NoError(t, err)
		mock.On(r.DB("mockdb").Table("network").Filter(r.MockAnything()).Filter(r.MockAnything())).Return(metal.Networks{supernetwork}, nil)
		changes := []r.ChangeResponse{{OldValue: map[string]interface{}{"id": float64(42)}}}
		mock.On(r.DB("mockdb").Table("integerpool").Limit(1).Delete(r.
			DeleteOpts{ReturnChanges: true})).Return(r.WriteResponse{Changes: changes}, nil)

		mock.On(r.DB("mockdb").Table("partition").Get(r.MockAnything())).Return(
			metal.Partition{
				Base: metal.Base{ID: tt.partitionID},
			},
			nil,
		)
		testdata.InitMockDBData(mock)

		psc := mdmv1mock.ProjectServiceClient{}
		psc.On("Get", testifymock.Anything, &mdmv1.ProjectGetRequest{Id: "project-1"}).Return(&mdmv1.ProjectResponse{
			Project: &mdmv1.Project{
				Meta: &mdmv1.Meta{Id: tt.projectID},
			},
		}, nil,
		)
		tsc := mdmv1mock.TenantServiceClient{}

		mdc := mdm.NewMock(&psc, &tsc, nil, nil)

		networkservice := NewNetwork(log, ds, ipamer, mdc)
		container := restful.NewContainer().Add(networkservice)

		allocateRequest := &v1.NetworkAllocateRequest{
			Describable:   v1.Describable{Name: &tt.networkName},
			NetworkBase:   v1.NetworkBase{PartitionID: &tt.partitionID, ProjectID: &tt.projectID},
			AddressFamily: tt.addressFamily,
			Length:        tt.childprefixlength,
		}

		js, err := json.Marshal(allocateRequest)
		require.NoError(t, err)

		body := bytes.NewBuffer(js)
		req := httptest.NewRequest("POST", "/v1/network/allocate", body)
		req.Header.Add("Content-Type", "application/json")
		container = injectAdmin(log, container, req)
		w := httptest.NewRecorder()
		container.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()
		require.Equal(t, tt.expectedStatus, resp.StatusCode, w.Body.String())
		if tt.expectedStatus > 300 {
			var result httperrors.HTTPErrorResponse
			err := json.NewDecoder(resp.Body).Decode(&result)

			require.NoError(t, err)
			require.Equal(t, tt.expectedErrorMessage, result.Message)
		} else {
			var result v1.NetworkResponse
			err = json.NewDecoder(resp.Body).Decode(&result)

			requestAF := "ipv4"
			if tt.addressFamily != nil {
				requestAF = "ipv6"
			}

			require.GreaterOrEqual(t, len(result.Prefixes), 1)
			resultFirstPrefix := netip.MustParsePrefix(result.Prefixes[0])
			af := "ipv4"
			if resultFirstPrefix.Addr().Is6() {
				af = "ipv6"
			}
			expectedLength := *supernetwork.ChildPrefixLength
			if tt.childprefixlength != nil {
				expectedLength = *tt.childprefixlength
			}
			require.NoError(t, err)
			require.Equal(t, tt.networkName, *result.Name)
			require.Equal(t, tt.partitionID, *result.PartitionID)
			require.Equal(t, tt.projectID, *result.ProjectID)
			require.Equal(t, requestAF, af)
			require.Equal(t, int(expectedLength), resultFirstPrefix.Bits())
		}
	}
}
