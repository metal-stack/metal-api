package netbox

import (
	"testing"

	"github.com/stretchr/testify/require"

	"git.f-i-ts.de/cloud-native/metal/metal-api/metal"
	nbdevice "git.f-i-ts.de/cloud-native/metal/metal-api/netbox-api/client/devices"
	"git.f-i-ts.de/cloud-native/metal/metal-api/netbox-api/models"

	"github.com/go-openapi/runtime"
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
					MacAddress: "12345",
				},
			}},
	}
	proxy := New()
	for _, td := range testdata {
		t.Run(td.name, func(t *testing.T) {
			proxy.register = func(params *nbdevice.NetboxAPIProxyAPIDeviceRegisterParams, authInfo runtime.ClientAuthInfoWriter) (*nbdevice.NetboxAPIProxyAPIDeviceRegisterOK, error) {
				require.Equal(t, td.uuid, params.UUID)
				require.Equal(t, td.siteid, *params.Request.Site)
				require.Equal(t, td.rackid, *params.Request.Rack)
				require.Equal(t, td.size, *params.Request.Size)
				for i, n := range td.hw {
					require.Equal(t, n.Name, *params.Request.Nics[i].Name)
					require.Equal(t, n.MacAddress, *params.Request.Nics[i].Mac)
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
			proxy.allocate = func(params *nbdevice.NetboxAPIProxyAPIDeviceAllocateParams, authInfo runtime.ClientAuthInfoWriter) (*nbdevice.NetboxAPIProxyAPIDeviceAllocateOK, error) {
				require.Equal(t, td.uuid, params.UUID)
				require.Equal(t, td.tenant, *(params.Request.Tenant))
				require.Equal(t, td.name, *(params.Request.Name))
				require.Equal(t, td.project, *(params.Request.Project))
				require.Equal(t, td.description, params.Request.Description)
				require.Equal(t, td.os, params.Request.Os)
				cidr := "a cidr"
				return &nbdevice.NetboxAPIProxyAPIDeviceAllocateOK{Payload: &models.DeviceAllocationResponse{
					Cidr: &cidr,
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
			proxy.release = func(params *nbdevice.NetboxAPIProxyAPIDeviceReleaseParams, authInfo runtime.ClientAuthInfoWriter) (*nbdevice.NetboxAPIProxyAPIDeviceReleaseOK, error) {
				require.Equal(t, td.uuid, params.UUID)
				return &nbdevice.NetboxAPIProxyAPIDeviceReleaseOK{}, nil
			}
			proxy.Release(td.uuid)
		})
	}
}
