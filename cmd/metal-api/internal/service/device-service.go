package service

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/netbox"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/utils/jwt"
	"git.f-i-ts.de/cloud-native/metal/metal-api/metal"
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
	netbox *netbox.APIProxy
	ds     datastore.Datastore
}

type allocateRequest struct {
	Name        string   `json:"name" description:"the new name for the allocated device" optional:"true"`
	Tenant      string   `json:"tenant" description:"the name of the owning tenant"`
	Hostname    string   `json:"hostname" description:"the hostname for the allocated device"`
	Description string   `json:"description" description:"the description for the allocated device" optional:"true"`
	ProjectID   string   `json:"projectid" description:"the project id to assign this device to"`
	SiteID      string   `json:"siteid" description:"the site id to assign this device to"`
	SizeID      string   `json:"sizeid" description:"the size id to assign this device to"`
	ImageID     string   `json:"imageid" description:"the image id to assign this device to"`
	SSHPubKeys  []string `json:"ssh_pub_keys" description:"the public ssh keys to access the device with"`
}

type registerRequest struct {
	UUID     string               `json:"uuid" description:"the product uuid of the device to register"`
	SiteID   string               `json:"siteid" description:"the site id to register this device with"`
	RackID   string               `json:"rackid" description:"the rack id where this device is connected to"`
	Hardware metal.DeviceHardware `json:"hardware" description:"the hardware of this device"`
	IPMI     metal.IPMI           `json:"ipmi" description:"the ipmi access infos"`
}

type phoneHomeRequest struct {
	PhoneHomeToken string `json:"phone_home_token" description:"the jwt that was issued for the device"`
}

func NewDevice(
	log log15.Logger,
	ds datastore.Datastore,
	pub *bus.Publisher,
	netbox *netbox.APIProxy) *restful.WebService {
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
		Returns(http.StatusCreated, "Created", metal.Device{}).
		Returns(http.StatusNotFound, "one of the given key values was not found", nil))

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

	ws.Route(ws.POST("/{id}/ipmi").To(dr.ipmiData).
		Doc("returns the IPMI connection data for a device").
		Param(ws.PathParameter("id", "identifier of the device").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(http.StatusOK, "OK", metal.IPMI{}).
		Returns(http.StatusNotFound, "Not Found", nil).
		Returns(http.StatusInternalServerError, "Internal Server Error", metal.Device{}))

	ws.Route(ws.GET("/{id}/wait").To(dr.waitForAllocation).
		Doc("wait for an allocation of this device").
		Param(ws.PathParameter("id", "identifier of the device").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(http.StatusOK, "OK", metal.DeviceWithPhoneHomeToken{}).
		Returns(http.StatusGatewayTimeout, "Timeout", nil).
		Returns(http.StatusInternalServerError, "Internal Server Error", nil))

	ws.Route(ws.POST("/phoneHome").To(dr.phoneHome).
		Doc("phone back home from the device").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(phoneHomeRequest{}).
		Returns(http.StatusOK, "OK", nil).
		Returns(http.StatusNotFound, "Device could not be found by id", nil).
		Returns(http.StatusBadRequest, "Bad Request", nil).
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
			ka := jwt.NewPhoneHomeClaims(&a)
			token, err := ka.JWT()
			if err != nil {
				return fmt.Errorf("could not create jwt: %v", err)
			}
			response.WriteEntity(metal.DeviceWithPhoneHomeToken{Device: &a, PhoneHomeToken: token})
		case <-ctx.Done():
			return fmt.Errorf("client timeout")
		}
		return nil
	})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
	}
}

func (dr deviceResource) phoneHome(request *restful.Request, response *restful.Response) {
	var data phoneHomeRequest
	err := request.ReadEntity(&data)
	if err != nil {
		sendError(dr, response, "phoneHome", http.StatusBadRequest, fmt.Errorf("Cannot read data from request: %v", err))
		return
	}
	c, err := jwt.FromJWT(data.PhoneHomeToken)
	if err != nil {
		sendError(dr, response, "phoneHome", http.StatusBadRequest, fmt.Errorf("Token is invalid: %v", err))
		return
	}
	if c.Device == nil || c.Device.ID == "" {
		sendError(dr, response, "phoneHome", http.StatusBadRequest, fmt.Errorf("Token contains malformed data"))
		return
	}
	oldDevice, err := dr.ds.FindDevice(c.Device.ID)
	if err != nil {
		sendError(dr, response, "phoneHome", http.StatusNotFound, err)
		return
	}
	newDevice := *oldDevice
	newDevice.LastPing = time.Now()
	err = dr.ds.UpdateDevice(oldDevice, &newDevice)
	if err != nil {
		sendError(dr, response, "phoneHome", http.StatusInternalServerError, err)
		return
	}
	response.WriteEntity(nil)
}

func (dr deviceResource) findDevice(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	device, err := dr.ds.FindDevice(id)
	if err != nil {
		sendError(dr, response, "findDevice", http.StatusNotFound, err)
		return
	}
	if device.SiteID != "" {
		site, err := dr.ds.FindSite(device.SiteID)
		if err != nil {
			sendError(dr, response, "findDevice", http.StatusInternalServerError, err)
			return
		}
		device.Site = *site
	}
	response.WriteEntity(device)
}

func (dr deviceResource) listDevices(request *restful.Request, response *restful.Response) {
	res, err := dr.ds.ListDevices()
	if err != nil {
		sendError(dr, response, "listDevices", http.StatusInternalServerError, err)
		return
	}
	res, err = dr.fillDeviceList(res...)
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
	result, err = dr.fillDeviceList(result...)
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
	site, err := dr.ds.FindSite(data.SiteID)
	if err != nil {
		sendError(dr, response, "registerDevice", http.StatusInternalServerError, err)
		return
	}

	size, err := dr.ds.FromHardware(data.Hardware)
	if err != nil {
		size = metal.UnknownSize
		dr.Error("no size found for hardware", "hardware", data.Hardware, "error", err)
	}

	err = dr.netbox.Register(site.ID, data.RackID, size.ID, data.UUID, data.Hardware.Nics)
	if err != nil {
		sendError(dr, response, "registerDevice", http.StatusInternalServerError, err)
		return
	}

	device, err := dr.ds.RegisterDevice(data.UUID, *site, *size, data.Hardware, data.IPMI)

	if err != nil {
		sendError(dr, response, "registerDevice", http.StatusInternalServerError, err)
		return
	}

	response.WriteEntity(device)
}

func (dr deviceResource) ipmiData(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	ipmi, err := dr.ds.FindIPMI(id)

	if err != nil {
		if err == datastore.ErrNotFound {
			sendError(dr, response, "ipmiData", http.StatusNotFound, err)
			return
		}
		sendError(dr, response, "ipmiData", http.StatusInternalServerError, err)
		return
	}
	response.WriteEntity(ipmi)
}

func (dr deviceResource) allocateDevice(request *restful.Request, response *restful.Response) {
	var allocate allocateRequest
	err := request.ReadEntity(&allocate)
	if err != nil {
		sendError(dr, response, "allocateDevice", http.StatusInternalServerError, fmt.Errorf("Cannot read request: %v", err))
		return
	}
	image, err := dr.ds.FindImage(allocate.ImageID)
	if err != nil {
		sendError(dr, response, "allocateDevice", http.StatusInternalServerError, fmt.Errorf("Cannot find image %q: %v", allocate.ImageID, err))
		return
	}
	size, err := dr.ds.FindSize(allocate.SizeID)
	if err != nil {
		sendError(dr, response, "allocateDevice", http.StatusInternalServerError, fmt.Errorf("Cannot find size %q: %v", allocate.SizeID, err))
		return
	}
	site, err := dr.ds.FindSite(allocate.SiteID)
	if err != nil {
		sendError(dr, response, "allocateDevice", http.StatusInternalServerError, err)
		return
	}

	d, err := dr.ds.AllocateDevice(allocate.Name, allocate.Description, allocate.Hostname,
		allocate.ProjectID, site, size,
		image, allocate.SSHPubKeys,
		allocate.Tenant,
		dr.netbox.Allocate)
	if err != nil {
		if err == datastore.ErrNoDeviceAvailable {
			sendError(dr, response, "allocateDevice", http.StatusNotFound, err)
		} else {
			sendError(dr, response, "allocateDevice", http.StatusInternalServerError, err)
		}
		return
	}
	response.WriteEntity(d)
}

func (dr deviceResource) freeDevice(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	device, err := dr.ds.FreeDevice(id)
	if err != nil {
		sendError(dr, response, "freeDevice", http.StatusInternalServerError, err)
		return
	}
	err = dr.netbox.Release(id)
	if err != nil {
		sendError(dr, response, "freeDevice", http.StatusInternalServerError, err)
		return
	}

	evt := metal.DeviceEvent{Type: metal.DELETE, Old: device}
	dr.Publish("device", evt)
	dr.Info("publish delete event", "event", evt)
	response.WriteEntity(device)
}

func (dr deviceResource) fillDeviceList(data ...metal.Device) ([]metal.Device, error) {
	all, err := dr.ds.ListSites()
	if err != nil {
		return nil, fmt.Errorf("cannot query all sites: %v", err)
	}
	sitemap := metal.Sites(all).ByID()

	res := make([]metal.Device, len(data), len(data))
	for i, d := range data {
		res[i] = d
		res[i].Site = sitemap[d.SiteID]
	}
	return res, nil
}
