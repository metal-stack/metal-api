package netbox

import (
	"fmt"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metal/metal-api/netbox-api/client"
	nbmachine "git.f-i-ts.de/cloud-native/metal/metal-api/netbox-api/client/machines"
	nbswitch "git.f-i-ts.de/cloud-native/metal/metal-api/netbox-api/client/switches"
	"git.f-i-ts.de/cloud-native/metal/metal-api/netbox-api/models"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/spf13/viper"
)

const (
	problemJSONMime = "application/problem+json"
)

// An APIProxy can be used to call netbox api functions. It wraps the token
// and has fields for the different functions. One can override the functions
// if this is needed (for example in testing code).
type APIProxy struct {
	*client.NetboxAPIProxy
	apitoken         string
	DoRegister       func(params *nbmachine.NetboxAPIProxyAPIMachineRegisterParams, authInfo runtime.ClientAuthInfoWriter) (*nbmachine.NetboxAPIProxyAPIMachineRegisterOK, error)
	DoAllocate       func(params *nbmachine.NetboxAPIProxyAPIMachineAllocateParams, authInfo runtime.ClientAuthInfoWriter) (*nbmachine.NetboxAPIProxyAPIMachineAllocateOK, error)
	DoRelease        func(params *nbmachine.NetboxAPIProxyAPIMachineReleaseParams, authInfo runtime.ClientAuthInfoWriter) (*nbmachine.NetboxAPIProxyAPIMachineReleaseOK, error)
	DoRegisterSwitch func(params *nbswitch.NetboxAPIProxyAPISwitchRegisterParams, authInfo runtime.ClientAuthInfoWriter) (*nbswitch.NetboxAPIProxyAPISwitchRegisterOK, error)
}

// New creates a new API proxy and uses the token from the viper-environment "netbox-api-token".
func New() *APIProxy {
	apitoken := viper.GetString("netbox-api-token")
	proxy := initNetboxProxy()
	return &APIProxy{
		NetboxAPIProxy:   proxy,
		apitoken:         apitoken,
		DoRegister:       proxy.Machines.NetboxAPIProxyAPIMachineRegister,
		DoAllocate:       proxy.Machines.NetboxAPIProxyAPIMachineAllocate,
		DoRelease:        proxy.Machines.NetboxAPIProxyAPIMachineRelease,
		DoRegisterSwitch: proxy.Switches.NetboxAPIProxyAPISwitchRegister,
	}
}

func initNetboxProxy() *client.NetboxAPIProxy {
	netboxAddr := viper.GetString("netbox-addr")
	cfg := client.DefaultTransportConfig().WithHost(netboxAddr)
	transport := httptransport.New(cfg.Host, cfg.BasePath, cfg.Schemes)
	// our nb-proxy returns this mimetype when a problem/error is returned
	transport.Consumers[problemJSONMime] = runtime.JSONConsumer()
	return client.New(transport, strfmt.Default)
}

func transformNicList(hwnics []metal.Nic) []*models.Nic {
	var nics []*models.Nic
	for i := range hwnics {
		nic := hwnics[i]
		m := string(nic.MacAddress)
		newnic := new(models.Nic)
		newnic.Mac = &m
		newnic.Name = &nic.Name
		nics = append(nics, newnic)
	}
	return nics
}

func (nb *APIProxy) authenticate(rq runtime.ClientRequest, rg strfmt.Registry) error {
	auth := "Token " + nb.apitoken
	rq.SetHeaderParam("Authorization", auth)
	return nil
}

// Register registers the given machine in netbox.
func (nb *APIProxy) Register(partid, rackid, size, uuid string, hwnics []metal.Nic) error {
	parms := nbmachine.NewNetboxAPIProxyAPIMachineRegisterParams()
	parms.UUID = uuid
	nics := transformNicList(hwnics)
	parms.Request = &models.MachineRegistrationRequest{
		Rack:      &rackid,
		Partition: &partid,
		Size:      &size,
		Nics:      nics,
	}

	_, err := nb.DoRegister(parms, runtime.ClientAuthInfoWriterFunc(nb.authenticate))
	if err != nil {
		return fmt.Errorf("error calling netbox: %v", err)
	}
	return nil
}

// Allocate uses the given machines for the given tenant. On success it returns
// the CIDR which must be used in the new machine.
func (nb *APIProxy) Allocate(uuid string, tenant string, vrf uint, project, name, description, os string) (string, error) {
	parms := nbmachine.NewNetboxAPIProxyAPIMachineAllocateParams()
	parms.UUID = uuid
	vrfid := fmt.Sprintf("%d", vrf)
	parms.Request = &models.MachineAllocationRequest{
		Name:        &name,
		Tenant:      &tenant,
		Vrf:         &vrfid,
		Project:     &project,
		Description: description,
		Os:          os,
	}

	rsp, err := nb.DoAllocate(parms, runtime.ClientAuthInfoWriterFunc(nb.authenticate))
	if err != nil {
		return "", fmt.Errorf("error calling netbox: %v", err)
	}
	return *rsp.Payload.Cidr, nil
}

// Release releases the machine with the given uuid in the netbox.
func (nb *APIProxy) Release(uuid string) error {
	parms := nbmachine.NewNetboxAPIProxyAPIMachineReleaseParams()
	parms.UUID = uuid

	_, err := nb.DoRelease(parms, runtime.ClientAuthInfoWriterFunc(nb.authenticate))
	if err != nil {
		return fmt.Errorf("error calling netbox: %v", err)
	}
	return nil
}

// RegisterSwitch registers the switch on the netbox side.
func (nb *APIProxy) RegisterSwitch(partid, rackid, uuid, name string, hwnics []metal.Nic) error {
	parms := nbswitch.NewNetboxAPIProxyAPISwitchRegisterParams()
	parms.UUID = uuid
	nics := transformNicList(hwnics)
	parms.Request = &models.SwitchRegistrationRequest{
		Name:      &name,
		Rack:      &rackid,
		Partition: &partid,
		Nics:      nics,
	}

	_, err := nb.DoRegisterSwitch(parms, runtime.ClientAuthInfoWriterFunc(nb.authenticate))
	if err != nil {
		return fmt.Errorf("error calling netbox: %v", err)
	}
	return nil
}
