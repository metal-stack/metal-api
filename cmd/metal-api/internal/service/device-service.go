package service

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/netbox"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/utils/jwt"
	"git.f-i-ts.de/cloud-native/metallib/bus"
	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"go.uber.org/zap"
)

const (
	waitForServerTimeout = 30 * time.Second
)

type deviceResource struct {
	webResource
	bus.Publisher
	netbox *netbox.APIProxy
}

// NewDevice returns a webservice for device specific endpoints.
func NewDevice(
	log *zap.Logger,
	ds *datastore.RethinkStore,
	pub bus.Publisher,
	netbox *netbox.APIProxy) *restful.WebService {
	dr := deviceResource{
		webResource: webResource{
			log:           log,
			SugaredLogger: log.Sugar(),
			ds:            ds,
		},
		Publisher: pub,
		netbox:    netbox,
	}
	return dr.webService()
}

// webService creates the webservice endpoint
func (dr deviceResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path("/v1/device").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"device"}

	ws.Route(ws.GET("/{id}").
		To(dr.restEntityGet(dr.ds.FindDevice)).
		Operation("findDevice").
		Doc("get device by id").
		Param(ws.PathParameter("id", "identifier of the device").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(metal.Device{}).
		Returns(http.StatusOK, "OK", metal.Device{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	ws.Route(ws.GET("/").
		To(dr.restListGet(dr.ds.ListDevices)).
		Operation("listDevices").
		Doc("get all known devices").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]metal.Device{}).
		Returns(http.StatusOK, "OK", []metal.Device{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	ws.Route(ws.GET("/find").To(dr.searchDevice).
		Doc("search devices").
		Param(ws.QueryParameter("mac", "one of the MAC address of the device").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]metal.Device{}).
		Returns(http.StatusOK, "OK", []metal.Device{}))

	ws.Route(ws.POST("/register").To(dr.registerDevice).
		Doc("register a device").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(metal.RegisterDevice{}).
		Writes(metal.Device{}).
		Returns(http.StatusOK, "OK", metal.Device{}).
		Returns(http.StatusCreated, "Created", metal.Device{}).
		Returns(http.StatusNotFound, "one of the given key values was not found", nil))

	ws.Route(ws.POST("/allocate").To(dr.allocateDevice).
		Doc("allocate a device").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(metal.AllocateDevice{}).
		Returns(http.StatusOK, "OK", metal.Device{}).
		Returns(http.StatusNotFound, "No free device for allocation found", nil).
		Returns(http.StatusBadRequest, "Bad Request", metal.ErrorResponse{}))

	ws.Route(ws.DELETE("/{id}/free").To(dr.freeDevice).
		Doc("free a device").
		Param(ws.PathParameter("id", "identifier of the device").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(http.StatusOK, "OK", metal.Device{}).
		Returns(http.StatusBadRequest, "Bad Request", metal.ErrorResponse{}))

	ws.Route(ws.POST("/{id}/ipmi").To(dr.ipmiData).
		Doc("returns the IPMI connection data for a device").
		Param(ws.PathParameter("id", "identifier of the device").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(http.StatusOK, "OK", metal.IPMI{}).
		Returns(http.StatusNotFound, "Not Found", nil).
		Returns(http.StatusBadRequest, "Bad Request", metal.ErrorResponse{}))

	ws.Route(ws.GET("/{id}/wait").To(dr.waitForAllocation).
		Doc("wait for an allocation of this device").
		Param(ws.PathParameter("id", "identifier of the device").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(http.StatusOK, "OK", metal.DeviceWithPhoneHomeToken{}).
		Returns(http.StatusGatewayTimeout, "Timeout", nil).
		Returns(http.StatusBadRequest, "Bad Request", metal.ErrorResponse{}))

	ws.Route(ws.POST("/{id}/report").To(dr.allocationReport).
		Doc("send the allocation report of a given device").
		Param(ws.PathParameter("id", "identifier of the device").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(metal.ReportAllocation{}).
		Returns(http.StatusOK, "OK", metal.DeviceAllocation{}).
		Returns(http.StatusNotFound, "Not Found", nil).
		Returns(http.StatusBadRequest, "Bad Request", metal.ErrorResponse{}))

	ws.Route(ws.POST("/phoneHome").To(dr.phoneHome).
		Doc("phone back home from the device").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(metal.PhoneHomeRequest{}).
		Returns(http.StatusOK, "OK", nil).
		Returns(http.StatusNotFound, "Device could not be found by id", nil).
		Returns(http.StatusBadRequest, "Bad Request", metal.ErrorResponse{}))

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
	var data metal.PhoneHomeRequest
	err := request.ReadEntity(&data)
	if err != nil {
		sendError(dr.log, response, "phoneHome", http.StatusBadRequest, fmt.Errorf("Cannot read data from request: %v", err))
		return
	}
	c, err := jwt.FromJWT(data.PhoneHomeToken)
	if err != nil {
		sendError(dr.log, response, "phoneHome", http.StatusBadRequest, fmt.Errorf("Token is invalid: %v", err))
		return
	}
	if c.Device == nil || c.Device.ID == "" {
		sendError(dr.log, response, "phoneHome", http.StatusBadRequest, fmt.Errorf("Token contains malformed data"))
		return
	}
	oldDevice, err := dr.ds.FindDevice(c.Device.ID)
	if err != nil {
		sendError(dr.log, response, "phoneHome", http.StatusNotFound, err)
		return
	}
	if oldDevice.Allocation == nil {
		dr.Error("unallocated devices sends phoneHome", "device", *oldDevice)
		sendError(dr.log, response, "phoneHome", http.StatusInternalServerError, fmt.Errorf("this device is not allocated"))
	}
	newDevice := *oldDevice
	newDevice.Allocation.LastPing = time.Now()
	err = dr.ds.UpdateDevice(oldDevice, &newDevice)
	if checkError(dr.log, response, "phoneHome", err) {
		return
	}
	response.WriteEntity(nil)
}

func (dr deviceResource) searchDevice(request *restful.Request, response *restful.Response) {
	mac := strings.TrimSpace(request.QueryParameter("mac"))

	result, err := dr.ds.SearchDevice(mac)
	if checkError(dr.log, response, "searchDevice", err) {
		return
	}

	response.WriteEntity(result)
}

func (dr deviceResource) registerDevice(request *restful.Request, response *restful.Response) {
	var data metal.RegisterDevice
	err := request.ReadEntity(&data)
	if checkError(dr.log, response, "registerDevice", err) {
		return
	}
	if data.UUID == "" {
		sendError(dr.log, response, "registerDevice", http.StatusBadRequest, fmt.Errorf("No UUID given"))
		return
	}
	site, err := dr.ds.FindSite(data.SiteID)
	if checkError(dr.log, response, "registerDevice", err) {
		return
	}

	size, err := dr.ds.FromHardware(data.Hardware)
	if err != nil {
		size = metal.UnknownSize
		dr.Error("no size found for hardware", "hardware", data.Hardware, "error", err)
	}

	err = dr.netbox.Register(site.ID, data.RackID, size.ID, data.UUID, data.Hardware.Nics)
	if checkError(dr.log, response, "registerDevice", err) {
		return
	}

	device, err := dr.ds.RegisterDevice(data.UUID, *site, data.RackID, *size, data.Hardware, data.IPMI)

	if checkError(dr.log, response, "registerDevice", err) {
		return
	}

	err = dr.ds.UpdateSwitchConnections(device)
	if checkError(dr.log, response, "registerDevice", err) {
		return
	}

	response.WriteEntity(device)
}

func (dr deviceResource) ipmiData(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	ipmi, err := dr.ds.FindIPMI(id)

	if checkError(dr.log, response, "ipmiData", err) {
		return
	}
	response.WriteEntity(ipmi)
}

func (dr deviceResource) allocateDevice(request *restful.Request, response *restful.Response) {
	var allocate metal.AllocateDevice
	err := request.ReadEntity(&allocate)
	if checkError(dr.log, response, "allocateDevice", err) {
		return
	}
	if allocate.Tenant == "" {
		if checkError(dr.log, response, "allocateDevice", fmt.Errorf("no tenant given")) {
			dr.log.Error("allocate", zap.String("tenant", "missing"))
			return
		}
	}
	image, err := dr.ds.FindImage(allocate.ImageID)
	if checkError(dr.log, response, "allocateDevice", err) {
		return
	}
	size, err := dr.ds.FindSize(allocate.SizeID)
	if checkError(dr.log, response, "allocateDevice", err) {
		return
	}
	site, err := dr.ds.FindSite(allocate.SiteID)
	if checkError(dr.log, response, "allocateDevice", err) {
		return
	}

	d, err := dr.ds.AllocateDevice(allocate.Name, allocate.Description, allocate.Hostname,
		allocate.ProjectID, site, size,
		image, allocate.SSHPubKeys,
		allocate.Tenant,
		dr.netbox)
	if err != nil {
		if err == datastore.ErrNoDeviceAvailable {
			sendError(dr.log, response, "allocateDevice", http.StatusNotFound, err)
		} else {
			sendError(dr.log, response, "allocateDevice", http.StatusBadRequest, err)
		}
		return
	}
	response.WriteEntity(d)
}

func (dr deviceResource) freeDevice(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	device, err := dr.ds.FreeDevice(id)
	if checkError(dr.log, response, "freeDevice", err) {
		return
	}
	err = dr.netbox.Release(id)
	if checkError(dr.log, response, "freeDevice", err) {
		return
	}

	evt := metal.DeviceEvent{Type: metal.DELETE, Old: device}
	dr.Publish("device", evt)
	dr.Info("publish delete event", "event", evt)
	response.WriteEntity(device)
}

func (dr deviceResource) allocationReport(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	var report metal.ReportAllocation
	err := request.ReadEntity(&report)
	if checkError(dr.log, response, "allocationReport", err) {
		return
	}

	dev, err := dr.ds.FindDevice(id)

	if checkError(dr.log, response, "allocationReport", err) {
		return
	}
	if !report.Success {
		dr.Errorw("failed allocation", "id", id, "error-message", report.ErrorMessage)
		response.WriteEntity(dev.Allocation)
		return
	}
	if dev.Allocation == nil {
		sendError(dr.log, response, "allocationReport", http.StatusBadRequest, fmt.Errorf("the device %q is not allocated", id))
		return
	}
	old := *dev
	dev.Allocation.ConsolePassword = report.ConsolePassword
	dr.ds.UpdateDevice(&old, dev)
	response.WriteEntity(dev.Allocation)
}
