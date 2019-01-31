package netbox

import (
	"reflect"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/testdata"
	"git.f-i-ts.de/cloud-native/metal/metal-api/netbox-api/client"
	nbdevice "git.f-i-ts.de/cloud-native/metal/metal-api/netbox-api/client/devices"
	"git.f-i-ts.de/cloud-native/metal/metal-api/netbox-api/models"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	testdata := []struct {
		name   string
		siteid string
		rackid string
		size   string
		uuid   string
		hw     []metal.Nic
	}{
		{
			name:   "first register",
			siteid: "a site",
			rackid: "a rack",
			size:   "asize",
			uuid:   "a uuid",
			hw: []metal.Nic{
				metal.Nic{
					Name:       "nicname",
					MacAddress: metal.MacAddress("12345"),
				},
			}},
	}
	proxy := New()
	for _, td := range testdata {
		t.Run(td.name, func(t *testing.T) {
			proxy.DoRegister = func(params *nbdevice.NetboxAPIProxyAPIDeviceRegisterParams, authInfo runtime.ClientAuthInfoWriter) (*nbdevice.NetboxAPIProxyAPIDeviceRegisterOK, error) {
				require.Equal(t, td.uuid, params.UUID)
				require.Equal(t, td.siteid, *params.Request.Site)
				require.Equal(t, td.rackid, *params.Request.Rack)
				require.Equal(t, td.size, *params.Request.Size)
				for i, n := range td.hw {
					require.Equal(t, n.Name, *params.Request.Nics[i].Name)
					require.Equal(t, string(n.MacAddress), *params.Request.Nics[i].Mac)
				}
				return &nbdevice.NetboxAPIProxyAPIDeviceRegisterOK{}, nil
			}
			proxy.Register(td.siteid, td.rackid, td.size, td.uuid, td.hw)
		})
	}
}

func TestAllocate(t *testing.T) {
	testdata := []struct {
		name        string
		uuid        string
		tenant      string
		project     string
		description string
		os          string
	}{
		{
			name:        "first allocate",
			uuid:        "a uuid",
			tenant:      "a tenant",
			project:     "a project",
			description: "a description",
			os:          "an os",
		},
	}
	proxy := New()
	for _, td := range testdata {
		t.Run(td.name, func(t *testing.T) {
			proxy.DoAllocate = func(params *nbdevice.NetboxAPIProxyAPIDeviceAllocateParams, authInfo runtime.ClientAuthInfoWriter) (*nbdevice.NetboxAPIProxyAPIDeviceAllocateOK, error) {
				require.Equal(t, td.uuid, params.UUID)
				require.Equal(t, td.tenant, *(params.Request.Tenant))
				require.Equal(t, td.name, *(params.Request.Name))
				require.Equal(t, td.project, *(params.Request.Project))
				require.Equal(t, td.description, params.Request.Description)
				require.Equal(t, td.os, params.Request.Os)
				cidr := "a cidr"
				vrfrd := "00:123456789012"
				return &nbdevice.NetboxAPIProxyAPIDeviceAllocateOK{Payload: &models.DeviceAllocationResponse{
					Cidr:  &cidr,
					VrfRd: &vrfrd,
				}}, nil
			}
			proxy.Allocate(td.uuid, td.tenant, td.project, td.name, td.description, td.os)
		})
	}
}

func TestRelease(t *testing.T) {
	testdata := []struct {
		name string
		uuid string
	}{
		{
			name: "first release",
			uuid: "a uuid",
		},
	}
	proxy := New()
	for _, td := range testdata {
		t.Run(td.name, func(t *testing.T) {
			proxy.DoRelease = func(params *nbdevice.NetboxAPIProxyAPIDeviceReleaseParams, authInfo runtime.ClientAuthInfoWriter) (*nbdevice.NetboxAPIProxyAPIDeviceReleaseOK, error) {
				require.Equal(t, td.uuid, params.UUID)
				return &nbdevice.NetboxAPIProxyAPIDeviceReleaseOK{}, nil
			}
			proxy.Release(td.uuid)
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		want *APIProxy
	}{
		// Test-Data List / Test Cases:
		// To be done
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_initNetboxProxy(t *testing.T) {
	tests := []struct {
		name string
		want *client.NetboxAPIProxy
	}{
		// Test-Data List / Test Cases:
		// To be done
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := initNetboxProxy(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("initNetboxProxy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_transformNicList(t *testing.T) {
	type args struct {
		hwnics []metal.Nic
	}
	tests := []struct {
		name string
		args args
		want []*models.Nic
	}{
		// Test-Data List / Test Cases:
		// To be done
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := transformNicList(tt.args.hwnics); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("transformNicList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAPIProxy_authenticate(t *testing.T) {
	type args struct {
		rq runtime.ClientRequest
		rg strfmt.Registry
	}
	tests := []struct {
		name    string
		nb      *APIProxy
		args    args
		wantErr bool
	}{
		// Test-Data List / Test Cases:
		// To be done
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.nb.authenticate(tt.args.rq, tt.args.rg); (err != nil) != tt.wantErr {
				t.Errorf("APIProxy.authenticate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAPIProxy_Register(t *testing.T) {
	type args struct {
		siteid string
		rackid string
		size   string
		uuid   string
		hwnics []metal.Nic
	}
	tests := []struct {
		name    string
		nb      *APIProxy
		args    args
		wantErr bool
	}{
		// Test-Data List / Test Cases:
		// To be done
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.nb.Register(tt.args.siteid, tt.args.rackid, tt.args.size, tt.args.uuid, tt.args.hwnics); (err != nil) != tt.wantErr {
				t.Errorf("APIProxy.Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAPIProxy_Allocate(t *testing.T) {
	type args struct {
		uuid        string
		tenant      string
		project     string
		name        string
		description string
		os          string
	}
	tests := []struct {
		name    string
		nb      *APIProxy
		args    args
		want    string
		wantErr bool
	}{
		// Test-Data List / Test Cases:
		// To be done
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.nb.Allocate(tt.args.uuid, tt.args.tenant, tt.args.project, tt.args.name, tt.args.description, tt.args.os)
			if (err != nil) != tt.wantErr {
				t.Errorf("APIProxy.Allocate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("APIProxy.Allocate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAPIProxy_Release(t *testing.T) {
	type args struct {
		uuid string
	}
	tests := []struct {
		name    string
		nb      *APIProxy
		args    args
		wantErr bool
	}{
		// Test-Data List / Test Cases:
		// To be done
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.nb.Release(tt.args.uuid); (err != nil) != tt.wantErr {
				t.Errorf("APIProxy.Release() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAPIProxy_RegisterSwitch(t *testing.T) {
	type args struct {
		siteid string
		rackid string
		uuid   string
		name   string
		hwnics []metal.Nic
	}
	tests := []struct {
		name    string
		nb      *APIProxy
		args    args
		wantErr bool
	}{
		// Test-Data List / Test Cases: "wantErr=false" is missing
		{
			name: "TestAPIProxy_RegisterSwitch Test 1",
			nb:   New(),
			args: args{
				siteid: "1",
				rackid: "1",
				uuid:   "1",
				name:   "1",
				hwnics: testdata.TestNics,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.nb.RegisterSwitch(tt.args.siteid, tt.args.rackid, tt.args.uuid, tt.args.name, tt.args.hwnics); (err != nil) != tt.wantErr {
				t.Errorf("APIProxy.RegisterSwitch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
