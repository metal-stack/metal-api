package service

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"git.f-i-ts.de/cloud-native/maas/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/maas/metal-api/netbox-api/client"
	nbdevice "git.f-i-ts.de/cloud-native/maas/metal-api/netbox-api/client/device"
	"git.f-i-ts.de/cloud-native/maas/metal-api/netbox-api/models"
	"git.f-i-ts.de/cloud-native/maas/metal-api/pkg/metal"
	"git.f-i-ts.de/cloud-native/metallib/bus"
	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/inconshreveable/log15"
)

const (
	waitForServerTimeout = 30 * time.Second
)

type deviceResource struct {
	log15.Logger
	*bus.Publisher
	netbox *client.NetboxAPIProxy
	ds     datastore.Datastore
}

type allocateRequest struct {
	Name        string `json:"name" description:"the new name for the allocated device" optional:"true"`
	Tenant      string `json:"tenant" description:"the name of the owning tenant"`
	TenantGroup string `json:"tenant_group" description:"the name of the owning tenant group"`
	Hostname    string `json:"hostname" description:"the hostname for the allocated device"`
	Description string `json:"description" description:"the description for the allocated device" optional:"true"`
	ProjectID   string `json:"projectid" description:"the project id to assign this device to"`
	SiteID      string `json:"siteid" description:"the site id to assign this device to"`
	SizeID      string `json:"sizeid" description:"the size id to assign this device to"`
	ImageID     string `json:"imageid" description:"the image id to assign this device to"`
	SSHPubKey   string `json:"ssh_pub_key" description:"the public ssh key to access the device with"`
}

type registerRequest struct {
	UUID     string               `json:"uuid" description:"the product uuid of the device to register"`
	SiteID   string               `json:"siteid" description:"the site id to register this device with"`
	RackID   string               `json:"rackid" description:"the rack id where this device is connected to"`
	Hardware metal.DeviceHardware `json:"hardware" description:"the hardware of this device"`
}

func NewDevice(
	log log15.Logger,
	ds datastore.Datastore,
	pub *bus.Publisher,
	netbox *client.NetboxAPIProxy) *restful.WebService {
	dr := deviceResource{
		Logger:    log,
		ds:        ds,
		Publisher: pub,
		netbox:    netbox,
	}
	return dr.webService()
}

// webService creates the webservice endpoint
func (dr deviceResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path("/device").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"device"}

	ws.Route(ws.GET("/{id}").To(dr.findDevice).
		Doc("get device by id").
		Param(ws.PathParameter("id", "identifier of the device").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(metal.Device{}).
		Returns(http.StatusOK, "OK", metal.Device{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	ws.Route(ws.GET("/").To(dr.listDevices).
		Doc("get all known devices").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]metal.Device{}).
		Returns(http.StatusOK, "OK", []metal.Device{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	ws.Route(ws.GET("/find").To(dr.searchDevice).
		Doc("search devices").
		Param(ws.QueryParameter("mac", "one of the MAC address of the device").DataType("string")).
		Param(ws.QueryParameter("projectid", "search for devices with the givne projectid").DataType("string")).
		Param(ws.QueryParameter("allocated", "returns allocated machines if set to true, free machines when set to false, all machines when not provided").DataType("boolean")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]metal.Device{}).
		Returns(http.StatusOK, "OK", []metal.Device{}))

	ws.Route(ws.POST("/register").To(dr.registerDevice).
		Doc("register a device").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(registerRequest{}).
		Writes(metal.Device{}).
		Returns(http.StatusOK, "OK", metal.Device{}).
		Returns(http.StatusCreated, "Created", metal.Device{}))

	ws.Route(ws.POST("/allocate").To(dr.allocateDevice).
		Doc("allocate a device").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(allocateRequest{}).
		Returns(http.StatusOK, "OK", metal.Device{}).
		Returns(http.StatusNotFound, "No free device for allocation found", nil).
		Returns(http.StatusInternalServerError, "Internal Server Error", nil))

	ws.Route(ws.DELETE("/{id}/free").To(dr.freeDevice).
		Doc("free a device").
		Param(ws.PathParameter("id", "identifier of the device").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(http.StatusOK, "OK", metal.Device{}).
		Returns(http.StatusInternalServerError, "Internal Server Error", metal.Device{}))

	ws.Route(ws.GET("/{id}/wait").To(dr.waitForAllocation).
		Doc("wait for an allocation of this device").
		Param(ws.PathParameter("id", "identifier of the device").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(http.StatusOK, "OK", metal.Device{}).
		Returns(http.StatusGatewayTimeout, "Timeout", nil).
		Returns(http.StatusInternalServerError, "Internal Server Error", nil))

	return ws
}

func (dr deviceResource) waitForAllocation(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	ctx := request.Request.Context()
	err := dr.ds.Wait(id, func(alloc datastore.Allocation) error {
		select {
		case <-time.After(waitForServerTimeout):
			response.WriteErrorString(http.StatusGatewayTimeout, "server timeout")
			return fmt.Errorf("server timeout")
		case a := <-alloc:
			dr.Info("return allocated device", "device", a)
			response.WriteEntity(a)
		case <-ctx.Done():
			return fmt.Errorf("client timeout")
		}
		return nil
	})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
	}
}

func (dr deviceResource) findDevice(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	device, err := dr.ds.FindDevice(id)
	if err != nil {
		sendError(dr, response, "findDevice", http.StatusNotFound, err)
		return
	}
	response.WriteEntity(device)
}

func (dr deviceResource) listDevices(request *restful.Request, response *restful.Response) {
	res, err := dr.ds.ListDevices()
	if err != nil {
		sendError(dr, response, "listDevices", http.StatusInternalServerError, err)
		return
	}
	response.WriteEntity(res)
}

func (dr deviceResource) searchDevice(request *restful.Request, response *restful.Response) {
	mac := strings.TrimSpace(request.QueryParameter("mac"))
	prjid := strings.TrimSpace(request.QueryParameter("projectid"))
	var free *bool
	salloc := request.QueryParameter("allocated")
	if salloc != "" {
		allocated, _ := strconv.ParseBool(salloc)
		allocated = !allocated
		free = &allocated
	}

	result, err := dr.ds.SearchDevice(prjid, mac, free)
	if err != nil {
		sendError(dr, response, "searchDevice", http.StatusInternalServerError, err)
		return
	}

	response.WriteEntity(result)
}

func (dr deviceResource) registerDevice(request *restful.Request, response *restful.Response) {
	var data registerRequest
	err := request.ReadEntity(&data)
	if err != nil {
		sendError(dr, response, "registerDevice", http.StatusInternalServerError, fmt.Errorf("Cannot read data from request: %v", err))
		return
	}
	if data.UUID == "" {
		sendError(dr, response, "registerDevice", http.StatusInternalServerError, fmt.Errorf("No UUID given"))
		return
	}

	err = dr.netboxRegister(data)
	if err != nil {
		sendError(dr, response, "registerDevice", http.StatusInternalServerError, err)
		return
	}

	device, err := dr.ds.RegisterDevice(data.UUID, data.SiteID, data.Hardware)

	if err != nil {
		sendError(dr, response, "registerDevice", http.StatusInternalServerError, err)
		return
	}

	response.WriteEntity(device)
}

func (dr deviceResource) allocateDevice(request *restful.Request, response *restful.Response) {
	var allocate allocateRequest
	err := request.ReadEntity(&allocate)
	if err != nil {
		sendError(dr, response, "allocateDevice", http.StatusInternalServerError, fmt.Errorf("Cannot read request: %v", err))
		return
	}
	d, err := dr.ds.AllocateDevice(allocate.Name, allocate.Description, allocate.Hostname, allocate.ProjectID, allocate.SiteID, allocate.SizeID, allocate.ImageID, allocate.SSHPubKey)
	if err != nil {
		if err == datastore.ErrNoDeviceAvailable {
			sendError(dr, response, "allocateDevice", http.StatusNotFound, err)
		} else {
			sendError(dr, response, "allocateDevice", http.StatusInternalServerError, err)
		}
		return
	}
	cidr, err := dr.netboxAllocate(allocate.Tenant, allocate.TenantGroup, d)
	if err != nil {
		sendError(dr, response, "cannot allocate at netbox", http.StatusInternalServerError, err)
	}
	d.Cidr = cidr

	response.WriteEntity(d)
}

func (dr deviceResource) freeDevice(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	device, err := dr.ds.FreeDevice(id)
	if err != nil {
		sendError(dr, response, "freeDevice", http.StatusInternalServerError, err)
		return
	}
	evt := metal.DeviceEvent{Type: metal.DELETE, Old: device}
	dr.Publish("device", evt)
	dr.Info("publish delete event", "event", evt)
	response.WriteEntity(device)
}

func (dr deviceResource) netboxRegister(data registerRequest) error {
	parms := nbdevice.NewLibServerRegisterDeviceParams()
	parms.UUID = data.UUID
	size := calculateSize(data)
	var nics []*models.Nic
	for i := range data.Hardware.Nics {
		nic := data.Hardware.Nics[i]
		newnic := new(models.Nic)
		newnic.Mac = &nic.MacAddress
		newnic.Name = &nic.Name
		nics = append(nics, newnic)
	}
	parms.Request = &models.DeviceRegistrationRequest{
		Rack: &data.RackID,
		Site: &data.SiteID,
		Size: &size,
		Nics: nics,
	}

	_, err := dr.netbox.Device.LibServerRegisterDevice(parms)
	if err != nil {
		return fmt.Errorf("error calling netbox: %v", err)
	}
	return nil
}

func (dr deviceResource) netboxAllocate(tenant, tenantgroup string, d *metal.Device) (string, error) {
	parms := nbdevice.NewLibServerAllocateDeviceParams()
	parms.UUID = d.ID
	parms.Request = &models.DeviceAllocationRequest{
		Name:        &d.Name,
		Tenant:      &tenant,
		TenantGroup: &tenantgroup,
	}

	rsp, err := dr.netbox.Device.LibServerAllocateDevice(parms)
	if err != nil {
		return "", fmt.Errorf("error calling netbox: %v", err)
	}
	return rsp.Payload.Cidr, nil
}

func calculateSize(rq registerRequest) string {
	return "t1.small.x86"
}
