package service

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/inconshreveable/log15"

	"git.f-i-ts.de/cloud-native/maas/maas-service/pkg/maas"
	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/mitchellh/mapstructure"
)

// A lshwInformation contains the required fields from the discovered information data. We only
// declare the fields which are needed, not a full LSHW model because we are not sure if the
// transported data is always identical over all hardware types.
type lshwInformation struct {
	Configuration struct {
		UUID string `json:"uuid"`
	} `json:"configuration"`
}

type lshwElement map[string]interface{}

type devicePool struct {
	all       map[string]*maas.Device
	free      map[string]*maas.Device
	allocated map[string]*maas.Device
}

// NewDevice returns a new Device endpoint
func NewDevice(log log15.Logger) *restful.WebService {
	dr := deviceResource{
		Logger: log,
		pool: devicePool{
			all:       make(map[string]*maas.Device),
			free:      make(map[string]*maas.Device),
			allocated: make(map[string]*maas.Device),
		},
	}
	return dr.webService()
}

// The deviceResource is the entrypoint for the whole device endpoints
type deviceResource struct {
	log15.Logger
	// dummy as long we do not have a database
	pool devicePool
}

// webService creates the webservice endpoint
func (dr deviceResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path("/device").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"device"}

	ws.Route(ws.GET("/{id}").To(dr.getDevice).
		Doc("get device by id").
		Param(ws.PathParameter("id", "identifier of the device").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(maas.Device{}).
		Returns(http.StatusOK, "OK", maas.Device{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	ws.Route(ws.GET("/").To(dr.getDevices).
		Doc("get all known devices").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]maas.Device{}).
		Returns(http.StatusOK, "OK", []maas.Device{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	ws.Route(ws.GET("/find").To(dr.findDevice).
		Doc("search devices").
		Param(ws.QueryParameter("mac", "one of the MAC address of the device").DataType("string")).
		Param(ws.QueryParameter("projectid", "search for devices with the givne projectid").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]maas.Device{}).
		Returns(http.StatusOK, "OK", []maas.Device{}))

	ws.Route(ws.POST("/register").To(dr.registerDevice).
		Doc("register a device").
		Param(ws.BodyParameter("rawdata", "raw json data").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(maas.Device{}).
		Returns(http.StatusOK, "OK", maas.Device{}).
		Returns(http.StatusCreated, "Created", maas.Device{}))

	return ws
}

func (dr deviceResource) getDevice(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	if d, ok := dr.pool.all[id]; ok {
		response.WriteEntity(d)
		return
	}
	response.WriteErrorString(http.StatusNotFound, fmt.Sprintf("the device-id %q was not found", id))
}

func (dr deviceResource) getDevices(request *restful.Request, response *restful.Response) {
	res := make([]*maas.Device, 0)
	for _, v := range dr.pool.all {
		res = append(res, v)
	}
	response.WriteEntity(res)
}

func (dr deviceResource) findDevice(request *restful.Request, response *restful.Response) {
	mac := strings.TrimSpace(request.QueryParameter("mac"))
	prjid := strings.TrimSpace(request.QueryParameter("projectid"))

	if mac == "" {
		msg := "empty MAC in findDevice"
		dr.Logger.Info(msg)
		http.Error(response, msg, http.StatusNotFound)
		return
	}
	result := make([]*maas.Device, 0)
	for _, d := range dr.pool.all {
		if prjid != "" && d.Project != prjid {
			continue
		}
		if d.HasMAC(mac) {
			result = append(result, d)
		}
	}
	response.WriteEntity(result)
}

func (dr deviceResource) registerDevice(request *restful.Request, response *restful.Response) {
	data := make(map[string]interface{})
	err := request.ReadEntity(&data)
	if err != nil {
		dr.Error("cannot read json from request", "error", err)
		http.Error(response, "Cannot read raw data from request", http.StatusInternalServerError)
		return
	}
	var info lshwInformation
	err = mapstructure.Decode(data, &info)
	if err != nil {
		dr.Error("cannot decode required lshw information", "error", err)
		http.Error(response, "Cannot decode required lshw information", http.StatusInternalServerError)
		return
	}
	result, has := dr.pool.all[info.Configuration.UUID]
	resultStatus := http.StatusOK
	if !has {
		result = new(maas.Device)
		resultStatus = http.StatusCreated
	}
	var macs []lshwElement
	result.ID = info.Configuration.UUID
	searchNetworkEntries(data, &macs)
	for _, m := range macs {
		result.MACAddresses = append(result.MACAddresses, m["serial"].(string))
	}
	dr.pool.all[info.Configuration.UUID] = result
	response.WriteHeaderAndEntity(resultStatus, result)
}

func searchNetworkEntries(data map[string]interface{}, result *[]lshwElement) {
	clzz, has := data["class"]
	if !has {
		return
	}
	if clzz == "network" {
		*result = append(*result, data)
	}
	child, has := data["children"]
	if has {
		childs := child.([]interface{})
		for i := range childs {
			cc := childs[i]
			searchNetworkEntries(cc.(map[string]interface{}), result)
		}
	}
}
