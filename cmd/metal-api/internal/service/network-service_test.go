package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	restful "github.com/emicklei/go-restful/v3"
	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	mdmv1mock "github.com/metal-stack/masterdata-api/api/v1/mocks"
	mdm "github.com/metal-stack/masterdata-api/pkg/client"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/ipam"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"
	"github.com/metal-stack/metal-lib/httperrors"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

func TestGetNetworks(t *testing.T) {
	ds, mock := datastore.InitMockDB(t)
	ipamer, err := testdata.InitMockIpamData(mock, false)
	require.NoError(t, err)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	networkservice := NewNetwork(log, ds, ipamer, nil)
	container := restful.NewContainer().Add(networkservice)
	req := httptest.NewRequest("GET", "/v1/network", nil)
	container = injectViewer(log, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result []v1.NetworkResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

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
	ipamer, err := testdata.InitMockIpamData(mock, false)
	require.NoError(t, err)

	testdata.InitMockDBData(mock)
	log := slog.Default()

	networkservice := NewNetwork(log, ds, ipamer, nil)
	container := restful.NewContainer().Add(networkservice)
	req := httptest.NewRequest("GET", "/v1/network/1", nil)
	container = injectViewer(log, container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.NetworkResponse
	err = json.NewDecoder(resp.Body).Decode(&result)

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
	ipamer, err := testdata.InitMockIpamData(mock, false)
	require.NoError(t, err)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	networkservice := NewNetwork(log, ds, ipamer, nil)
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
	ipamer, err := testdata.InitMockIpamData(mock, false)
	require.NoError(t, err)
	mock.On(r.DB("mockdb").Table("network").Filter(r.MockAnything())).Return([]interface{}{testdata.Nw1}, nil)
	testdata.InitMockDBData(mock)
	log := slog.Default()

	networkService := NewNetwork(log, ds, ipamer, nil)
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
	err = json.NewDecoder(resp.Body).Decode(&results)

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
		childprefixlength    metal.ChildPrefixLength
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
			name:                "privatesuper IPv4",
			networkName:         testdata.Nw1.Name,
			partitionID:         testdata.Nw1.PartitionID,
			projectID:           testdata.Nw1.ProjectID,
			prefixes:            []string{"172.0.0.0/24"},
			destinationPrefixes: []string{"0.0.0.0/0"},
			childprefixlength: metal.ChildPrefixLength{
				metal.IPv4AddressFamily: 22,
			},
			privateSuper:         true,
			vrf:                  uint(10000),
			expectedStatus:       http.StatusBadRequest,
			expectedErrorMessage: "given defaultchildprefixlength 22 is not greater than prefix length of:172.0.0.0/24",
		},
		{
			name:                 "privatesuper IPv4 without defaultchildprefixlength",
			networkName:          testdata.Nw1.Name,
			partitionID:          testdata.Nw1.PartitionID,
			projectID:            testdata.Nw1.ProjectID,
			prefixes:             []string{"172.0.0.0/24"},
			destinationPrefixes:  []string{"0.0.0.0/0"},
			privateSuper:         true,
			vrf:                  uint(10001),
			expectedStatus:       http.StatusBadRequest,
			expectedErrorMessage: "private super network must always contain a defaultchildprefixlength",
		},
		{
			name:                "privatesuper Mixed",
			networkName:         "privatesuper mixed",
			partitionID:         "3",
			projectID:           "",
			prefixes:            []string{"fdaa:bbcc::/50", "172.0.0.0/16"},
			destinationPrefixes: []string{"::/0", "0.0.0.0/0"},
			childprefixlength: metal.ChildPrefixLength{
				metal.IPv4AddressFamily: 22,
				metal.IPv6AddressFamily: 64,
			},
			privateSuper:   true,
			vrf:            uint(10000),
			expectedStatus: http.StatusCreated,
		},
		{
			name:                "broken IPv4",
			networkName:         testdata.Nw1.Name,
			partitionID:         testdata.Nw1.PartitionID,
			projectID:           testdata.Nw1.ProjectID,
			prefixes:            []string{"192.168.265.0/24"},
			destinationPrefixes: []string{"0.0.0.0/0"},
			childprefixlength: metal.ChildPrefixLength{
				metal.IPv6AddressFamily: 64,
			},
			privateSuper:         true,
			vrf:                  uint(10000),
			expectedStatus:       http.StatusBadRequest,
			expectedErrorMessage: "given cidr 192.168.265.0/24 is not a valid ip with mask: netip.ParsePrefix(\"192.168.265.0/24\"): ParseAddr(\"192.168.265.0\"): IPv4 field has value >255",
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
			expectedErrorMessage: "given cidr fdaa:::/50 is not a valid ip with mask: netip.ParsePrefix(\"fdaa:::/50\"): ParseAddr(\"fdaa:::\"): each colon-separated field must have at least one digit (at \":\")",
		},
		{
			name:                "mixed prefix addressfamilies",
			networkName:         testdata.Nw1.Name,
			partitionID:         testdata.Nw1.PartitionID,
			projectID:           testdata.Nw1.ProjectID,
			prefixes:            []string{"172.0.0.0/24", "fdaa:bbcc::/50"},
			destinationPrefixes: []string{"0.0.0.0/0"},
			vrf:                 uint(10000),
			expectedStatus:      http.StatusCreated,
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
			expectedErrorMessage: "given cidr 0.0.0.0/33 is not a valid ip with mask: netip.ParsePrefix(\"0.0.0.0/33\"): prefix length out of range",
		},
		{
			name:        "broken childprefixlength",
			networkName: testdata.Nw1.Name,
			partitionID: testdata.Nw1.PartitionID,
			projectID:   testdata.Nw1.ProjectID,
			prefixes:    []string{"fdaa:bbcc::/50"},
			childprefixlength: metal.ChildPrefixLength{
				metal.IPv6AddressFamily: 50,
			},
			privateSuper:         true,
			vrf:                  uint(10000),
			expectedStatus:       http.StatusBadRequest,
			expectedErrorMessage: "given defaultchildprefixlength 50 is not greater than prefix length of:fdaa:bbcc::/50",
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
				createRequest.DefaultChildPrefixLength = tt.childprefixlength
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
					require.Equal(t, tt.childprefixlength, result.DefaultChildPrefixLength)
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
		childprefixlength    metal.ChildPrefixLength
		shared               bool
		expectedStatus       int
		expectedErrorMessage string
	}{
		{
			name:           "simple ipv4, default childprefixlength",
			networkName:    "tenantv4",
			partitionID:    testdata.Partition2.ID,
			projectID:      "project-1",
			expectedStatus: http.StatusCreated,
		},
		{
			name:        "simple ipv4, specific childprefixlength",
			networkName: "tenantv4.2",
			partitionID: testdata.Partition2.ID,
			projectID:   "project-1",
			childprefixlength: metal.ChildPrefixLength{
				metal.IPv4AddressFamily: 29,
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "ipv6 default childprefixlength",
			networkName:    "tenantv6",
			partitionID:    testdata.Partition2.ID,
			projectID:      "project-1",
			expectedStatus: http.StatusCreated,
		},
		{
			name:        "mixed, specific childprefixlength",
			networkName: "tenantv6.2",
			partitionID: "4",
			projectID:   "project-1",
			childprefixlength: metal.ChildPrefixLength{
				metal.IPv4AddressFamily: 22,
				metal.IPv6AddressFamily: 58,
			},
			expectedStatus: http.StatusCreated,
		},
	}
	for _, tt := range tests {
		ds, mock := datastore.InitMockDB(t)
		changes := []r.ChangeResponse{{OldValue: map[string]interface{}{"id": float64(42)}}}
		mock.On(r.DB("mockdb").Table("integerpool").Limit(1).Delete(r.
			DeleteOpts{ReturnChanges: true})).Return(r.WriteResponse{Changes: changes}, nil)

		ipamer, err := testdata.InitMockIpamData(mock, false)
		require.NoError(t, err)
		testdata.InitMockDBData(mock)

		psc := mdmv1mock.ProjectServiceClient{}
		psc.On("Get", testifymock.Anything, &mdmv1.ProjectGetRequest{Id: "project-1"}).Return(&mdmv1.ProjectResponse{
			Project: &mdmv1.Project{
				Meta: &mdmv1.Meta{Id: tt.projectID},
			},
		}, nil,
		)
		tsc := mdmv1mock.TenantServiceClient{}

		mdc := mdm.NewMock(&psc, &tsc, nil, nil, nil)

		networkservice := NewNetwork(log, ds, ipamer, mdc)
		container := restful.NewContainer().Add(networkservice)

		allocateRequest := &v1.NetworkAllocateRequest{
			Describable: v1.Describable{Name: &tt.networkName},
			NetworkBase: v1.NetworkBase{PartitionID: &tt.partitionID, ProjectID: &tt.projectID},
			Length:      tt.childprefixlength,
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

			require.GreaterOrEqual(t, len(result.Prefixes), 1)

			require.NoError(t, err)
			require.Equal(t, tt.networkName, *result.Name)
			require.Equal(t, tt.partitionID, *result.PartitionID)
			require.Equal(t, tt.projectID, *result.ProjectID)
		}
	}
}

func Test_validatePrefixesAndAddressFamilies(t *testing.T) {
	tests := []struct {
		name                     string
		prefixes                 metal.Prefixes
		prefixesAfs              metal.AddressFamilies
		destPrefixesAfs          metal.AddressFamilies
		defaultChildPrefixLength metal.ChildPrefixLength
		privateSuper             bool
		wantErr                  bool
		wantErrMsg               string
	}{
		{
			name:                     "all is valid",
			prefixes:                 metal.Prefixes{{IP: "1.3.4.0", Length: "24"}},
			prefixesAfs:              metal.AddressFamilies{metal.IPv4AddressFamily},
			destPrefixesAfs:          metal.AddressFamilies{metal.IPv4AddressFamily},
			defaultChildPrefixLength: metal.ChildPrefixLength{metal.IPv4AddressFamily: 28},
			privateSuper:             true,
			wantErr:                  false,
		},
		{
			name:                     "mixed afs in prefixes and destination prefixes",
			prefixes:                 metal.Prefixes{{IP: "1.3.4.0", Length: "24"}},
			prefixesAfs:              metal.AddressFamilies{metal.IPv4AddressFamily},
			destPrefixesAfs:          metal.AddressFamilies{metal.IPv6AddressFamily},
			defaultChildPrefixLength: metal.ChildPrefixLength{metal.IPv4AddressFamily: 28},
			privateSuper:             true,
			wantErr:                  true,
			wantErrMsg:               "addressfamily:IPv6 of destination prefixes is not present in existing prefixes",
		},
		{
			name:            "defaultChildPrefixLength not set for private super",
			prefixes:        metal.Prefixes{{IP: "1.3.4.0", Length: "24"}},
			prefixesAfs:     metal.AddressFamilies{metal.IPv4AddressFamily},
			destPrefixesAfs: metal.AddressFamilies{metal.IPv4AddressFamily},
			privateSuper:    true,
			wantErr:         true,
			wantErrMsg:      "private super network must always contain a defaultchildprefixlength",
		},
		{
			name:                     "defaultChildPrefixLength has invalid AF",
			prefixes:                 metal.Prefixes{{IP: "1.3.4.0", Length: "24"}},
			prefixesAfs:              metal.AddressFamilies{metal.IPv4AddressFamily},
			destPrefixesAfs:          metal.AddressFamilies{metal.IPv4AddressFamily},
			defaultChildPrefixLength: metal.ChildPrefixLength{metal.AddressFamily("ipv5"): 28},
			privateSuper:             true,
			wantErr:                  true,
			wantErrMsg:               "addressfamily of defaultchildprefixlength is invalid given addressfamily:\"ipv5\" is invalid",
		},
		{
			name:                     "defaultChildPrefixLength does not contain prefixes AF",
			prefixes:                 metal.Prefixes{{IP: "1.3.4.0", Length: "24"}},
			prefixesAfs:              metal.AddressFamilies{metal.IPv4AddressFamily, metal.IPv6AddressFamily},
			destPrefixesAfs:          metal.AddressFamilies{metal.IPv4AddressFamily},
			defaultChildPrefixLength: metal.ChildPrefixLength{metal.IPv6AddressFamily: 64},
			privateSuper:             true,
			wantErr:                  true,
			wantErrMsg:               "private super network must always contain a defaultchildprefixlength per addressfamily:IPv4",
		},
		{
			name:                     "defaultChildPrefixLength does not contain prefixes AF",
			prefixes:                 metal.Prefixes{{IP: "1.3.4.0", Length: "24"}},
			prefixesAfs:              metal.AddressFamilies{metal.IPv4AddressFamily},
			destPrefixesAfs:          metal.AddressFamilies{metal.IPv4AddressFamily},
			defaultChildPrefixLength: metal.ChildPrefixLength{metal.IPv4AddressFamily: 24},
			privateSuper:             true,
			wantErr:                  true,
			wantErrMsg:               "given defaultchildprefixlength 24 is not greater than prefix length of:1.3.4.0/24",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePrefixesAndAddressFamilies(tt.prefixes, tt.destPrefixesAfs, tt.defaultChildPrefixLength, tt.privateSuper)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePrefixesAndAddressFamilies() error = %v, wantErr %v", err, tt.wantErr)
			}
			if (err != nil) && tt.wantErr {
				require.Equal(t, tt.wantErrMsg, err.Error())
			}
		})
	}
}
