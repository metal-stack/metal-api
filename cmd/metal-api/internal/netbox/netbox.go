package netbox

import (
	"fmt"

	"git.f-i-ts.de/cloud-native/maas/metal-api/metal"
	"git.f-i-ts.de/cloud-native/maas/metal-api/netbox-api/client"
	nbdevice "git.f-i-ts.de/cloud-native/maas/metal-api/netbox-api/client/devices"
	"git.f-i-ts.de/cloud-native/maas/metal-api/netbox-api/models"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/spf13/viper"
)

type APIProxy struct {
	*client.NetboxAPIProxy
	apitoken string
}

func New() *APIProxy {
	apitoken := viper.GetString("netbox-api-token")
	return &APIProxy{
		NetboxAPIProxy: initNetboxProxy(),
		apitoken:       apitoken,
	}
}

func initNetboxProxy() *client.NetboxAPIProxy {
	netboxAddr := viper.GetString("netbox-addr")
	cfg := client.DefaultTransportConfig().WithHost(netboxAddr)
	return client.NewHTTPClientWithConfig(strfmt.Default, cfg)
}

func (nb *APIProxy) authenticate(rq runtime.ClientRequest, rg strfmt.Registry) error {
	auth := "Token " + nb.apitoken
	rq.SetHeaderParam("Authorization", auth)
	return nil
}

func (nb *APIProxy) Register(siteid, rackid, size, uuid string, hwnics []metal.Nic) error {
	parms := nbdevice.NewNetboxAPIProxyAPIDeviceRegisterParams()
	parms.UUID = uuid
	var nics []*models.Nic
	for i := range hwnics {
		nic := hwnics[i]
		newnic := new(models.Nic)
		newnic.Mac = &nic.MacAddress
		newnic.Name = &nic.Name
		nics = append(nics, newnic)
	}
	parms.Request = &models.DeviceRegistrationRequest{
		Rack: &rackid,
		Site: &siteid,
		Size: &size,
		Nics: nics,
	}

	_, err := nb.NetboxAPIProxy.Devices.NetboxAPIProxyAPIDeviceRegister(parms, runtime.ClientAuthInfoWriterFunc(nb.authenticate))
	if err != nil {
		return fmt.Errorf("error calling netbox: %v", err)
	}
	return nil
}

// Allocate uses the given devices for the given tenant. On success it returns
// the CIDR which must be used in the new machine.
func (nb *APIProxy) Allocate(uuid, tenant, project, name, description, os string) (string, error) {
	parms := nbdevice.NewNetboxAPIProxyAPIDeviceAllocateParams()
	parms.UUID = uuid
	parms.Request = &models.DeviceAllocationRequest{
		Name:        &name,
		Tenant:      &tenant,
		Project:     &project,
		Description: description,
		Os:          os,
	}

	rsp, err := nb.NetboxAPIProxy.Devices.NetboxAPIProxyAPIDeviceAllocate(parms, runtime.ClientAuthInfoWriterFunc(nb.authenticate))
	if err != nil {
		return "", fmt.Errorf("error calling netbox: %v", err)
	}
	return *rsp.Payload.Cidr, nil
}

func (nb *APIProxy) Release(uuid string) error {
	parms := nbdevice.NewNetboxAPIProxyAPIDeviceReleaseParams()
	parms.UUID = uuid

	_, err := nb.NetboxAPIProxy.Devices.NetboxAPIProxyAPIDeviceRelease(parms, runtime.ClientAuthInfoWriterFunc(nb.authenticate))
	if err != nil {
		return fmt.Errorf("error calling netbox: %v", err)
	}
	return nil
}
