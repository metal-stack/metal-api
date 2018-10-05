package service

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"git.f-i-ts.de/cloud-native/maas/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/maas/metal-api/pkg/metal"
	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/inconshreveable/log15"
)

const (
	waitForServerTimeout = 30 * time.Second
)

type deviceResource struct {
	log15.Logger
	ds datastore.Datastore
}

type allocateRequest struct {
	Name        string `json:"name" description:"the new name for the allocated device" optional:"true"`
	Hostname    string `json:"hostname" description:"the hostname for the allocated device"`
	Description string `json:"description" description:"the description for the allocated device" optional:"true"`
	ProjectID   string `json:"projectid" description:"the project id to assign this device to"`
	FacilityID  string `json:"facilityid" description:"the facility id to assign this device to"`
	SizeID      string `json:"sizeid" description:"the size id to assign this device to"`
	ImageID     string `json:"imageid" description:"the image id to assign this device to"`
	SSHPubKey   string `json:"ssh_pub_key" description:"the public ssh key to access the device with"`
}

type registerRequest struct {
	UUID       string               `json:"uuid" description:"the product uuid of the device to register"`
	FacilityID string               `json:"facilityid" description:"the facility id to register this device with"`
	Hardware   metal.DeviceHardware `json:"hardware" description:"the hardware of this device"`
}

func NewDevice(log log15.Logger, ds datastore.Datastore) *restful.WebService {
	dr := deviceResource{
		Logger: log,
		ds:     ds,
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
	dr.ds.Wait(id, func(alloc datastore.Allocation) error {
		select {
		case <-time.After(waitForServerTimeout):
			response.WriteErrorString(http.StatusGatewayTimeout, "server timeout")
			return fmt.Errorf("server timeout")
		case a := <-alloc:
			response.WriteEntity(a)
		case <-ctx.Done():
			return fmt.Errorf("client timeout")
		}
		return nil
	})
}

func (dr deviceResource) findDevice(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	device, err := dr.ds.FindDevice(id)
	if err != nil {
		dr.Error("error finding device", "id", id, "error", err)
		response.WriteError(http.StatusNotFound, err)
		return
	}
	response.WriteEntity(device)
}

func (dr deviceResource) listDevices(request *restful.Request, response *restful.Response) {
	res, err := dr.ds.ListDevices()
	if err != nil {
		dr.Error("cannot list devices", "error", err)
		response.WriteError(http.StatusInternalServerError, err)
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
		dr.Error("error searching devices", "mac", mac, "projectid", prjid, "free", free, "error", err)
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	response.WriteEntity(result)
}

func (dr deviceResource) registerDevice(request *restful.Request, response *restful.Response) {
	var data registerRequest
	err := request.ReadEntity(&data)
	if err != nil {
		dr.Error("cannot read entity from input", "error", err)
		response.WriteError(http.StatusInternalServerError, fmt.Errorf("Cannot read data from request: %v", err))
		return
	}
	if data.UUID == "" {
		dr.Error("no UUID given in entity", "entity", data)
		response.WriteErrorString(http.StatusInternalServerError, "No UUID given")
		return
	}

	device, err := dr.ds.RegisterDevice(data.UUID, data.FacilityID, data.Hardware)

	if err != nil {
		dr.Error("error registering device", "error", err, "entity", data)
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	response.WriteEntity(device)
}

func (dr deviceResource) allocateDevice(request *restful.Request, response *restful.Response) {
	var allocate allocateRequest
	err := request.ReadEntity(&allocate)
	if err != nil {
		dr.Error("cannot read entity from input", "error", err)
		response.WriteError(http.StatusInternalServerError, fmt.Errorf("Cannot read request: %v", err))
		return
	}
	d, err := dr.ds.AllocateDevice(allocate.Name, allocate.Description, allocate.Hostname, allocate.ProjectID, allocate.FacilityID, allocate.SizeID, allocate.ImageID, allocate.SSHPubKey)
	if err != nil {
		if err == datastore.ErrNoDeviceAvailable {
			dr.Warn("no device found for allocation", "request", allocate, "error", err)
			response.WriteError(http.StatusNotFound, err)
		} else {
			dr.Error("cannot allocate a device", "error", err, "request", allocate)
			response.WriteError(http.StatusInternalServerError, err)
		}
		return
	}
	response.WriteEntity(d)
}

func (dr deviceResource) freeDevice(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	device, err := dr.ds.FreeDevice(id)
	if err != nil {
		dr.Error("cannot free device", "id", id, "error", err)
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	response.WriteEntity(device)
}
