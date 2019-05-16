package service

import (
	"net/http"
	"time"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/utils"
	"git.f-i-ts.de/cloud-native/metallib/httperrors"
	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
)

type switchResource struct {
	webResource
}

// NewSwitch returns a webservice for switch specific endpoints.
func NewSwitch(ds *datastore.RethinkStore) *restful.WebService {
	sr := switchResource{
		webResource: webResource{
			ds: ds,
		},
	}
	return sr.webService()
}

func (sr switchResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path("/v1/switch").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"switch"}

	ws.Route(ws.GET("/{id}").
		To(sr.restEntityGet(sr.ds.FindSwitch)).
		Operation("findSwitch").
		Doc("get switch by id").
		Param(ws.PathParameter("id", "identifier of the switch").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(metal.Switch{}).
		Returns(http.StatusOK, "OK", metal.Switch{}).
		Returns(http.StatusNotFound, "Not Found", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/").To(sr.restListGet(sr.ds.ListSwitches)).
		Operation("listSwitches").
		Doc("get all switches").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]metal.Switch{}).
		Returns(http.StatusOK, "OK", []metal.Switch{}))

	ws.Route(ws.DELETE("/{id}").To(editor(sr.restEntityGet(sr.ds.DeleteSwitch))).
		Operation("deleteSwitch").
		Doc("deletes an switch and returns the deleted entity").
		Param(ws.PathParameter("id", "identifier of the switch").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(metal.Switch{}).
		Returns(http.StatusOK, "OK", metal.Switch{}).
		Returns(http.StatusNotFound, "Not Found", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/register").To(editor(sr.registerSwitch)).
		Doc("register a switch").
		Operation("registerSwitch").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(metal.RegisterSwitch{}).
		Returns(http.StatusOK, "OK", metal.Switch{}).
		Returns(http.StatusCreated, "Created", metal.Switch{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}))

	return ws
}

func (sr switchResource) registerSwitch(request *restful.Request, response *restful.Response) {
	op := utils.CurrentFuncName()
	var newSwitch metal.RegisterSwitch
	err := request.ReadEntity(&newSwitch)
	if checkError(request, response, op, err) {
		return
	}
	part, err := sr.ds.FindPartition(newSwitch.PartitionID)
	if checkError(request, response, op, err) {
		return
	}

	oldSwitch, err := sr.ds.FindSwitch(newSwitch.ID)
	sw := metal.NewSwitch(newSwitch.ID, newSwitch.RackID, newSwitch.Nics, part)
	if err != nil {
		if metal.IsNotFound(err) {
			sw.Created = time.Now()
			sw.Changed = sw.Created
			sw, err = sr.ds.CreateSwitch(sw)
			if checkError(request, response, op, err) {
				return
			}
			response.WriteHeaderAndEntity(http.StatusCreated, sw)
			return
		}
		sendError(utils.Logger(request), response, op, httperrors.InternalServerError(err))
		return
	}
	// Make sure we do not delete current connections
	sw.FromSwitch(oldSwitch)
	sw.Nics = oldSwitch.Nics
	sw.Changed = time.Now()

	err = sr.ds.UpdateSwitch(oldSwitch, sw)

	if checkError(request, response, op, err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, sw)
}
