package service

import (
	"net/http"

	"github.com/inconshreveable/log15"

	"git.f-i-ts.de/ize0h88/maas-service/pkg/maas"
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

	ws.Route(ws.GET("/").To(dr.getDevice).
		Doc("get devices").
		Param(ws.QueryParameter("id", "identifier of the device").DataType("string").AllowMultiple(true)).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]maas.Facility{}).
		Returns(http.StatusOK, "OK", maas.Facility{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	ws.Route(ws.POST("/register").To(dr.registerDevice).
		Doc("register a device").
		Param(ws.BodyParameter("rawdata", "raw json data").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(maas.Device{}).
		Returns(http.StatusOK, "OK", maas.Device{}).
		Returns(http.StatusCreated, "Created", maas.Device{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	return ws
}

func (dr deviceResource) getDevice(request *restful.Request, response *restful.Response) {
	request.Request.ParseForm()
	ids := request.Request.Form["id"]
	res := []*maas.Device{}
	if len(ids) == 0 {
		for _, d := range dr.pool.all {
			res = append(res, d)
		}
	} else {
		for _, id := range ids {
			dev, has := dr.pool.all[id]
			if has {
				res = append(res, dev)
			}
		}
	}

	response.WriteEntity(res)
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
	var result maas.Device
	result.ID = info.Configuration.UUID

	response.WriteEntity(result)
}
