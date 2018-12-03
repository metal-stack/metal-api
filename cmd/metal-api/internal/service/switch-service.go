package service

import (
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/metal"
	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
)

type switchResource struct {
	*zap.SugaredLogger
	log *zap.Logger
	ds  *datastore.RethinkStore
}

func NewSwitch(log *zap.Logger, ds *datastore.RethinkStore) *restful.WebService {
	sr := switchResource{
		SugaredLogger: log.Sugar(),
		log:           log,
		ds:            ds,
	}
	return sr.webService()
}

func (sr switchResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path("/switch").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"switch"}

	ws.Route(ws.GET("/").To(sr.listSwitches).
		Doc("get all switches").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]metal.Switch{}).
		Returns(http.StatusOK, "OK", []metal.Switch{}))

	ws.Route(ws.DELETE("/{id}").To(sr.deleteSwitch).
		Doc("deletes an switch and returns the deleted entity").
		Param(ws.PathParameter("id", "identifier of the switch").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(metal.Switch{}).
		Returns(http.StatusOK, "OK", metal.Switch{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	ws.Route(ws.POST("/register").To(sr.registerSwitch).
		Doc("register a switch").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(metal.Switch{}).
		Returns(http.StatusOK, "OK", metal.Switch{}).
		Returns(http.StatusCreated, "Created", metal.Switch{}).
		Returns(http.StatusConflict, "Conflict", nil))

	return ws
}

func (sr switchResource) listSwitches(request *restful.Request, response *restful.Response) {
	res, err := sr.ds.ListSwitches()
	if err != nil {
		sendError(sr.log, response, "listSwitches", http.StatusInternalServerError, err)
		return
	}
	response.WriteEntity(res)
}

func (sr switchResource) deleteSwitch(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	sw, err := sr.ds.DeleteSwitch(id)
	if err != nil {
		sendError(sr.log, response, "deleteSwitch", http.StatusNotFound, err)
	} else {
		response.WriteEntity(sw)
	}
}

func (sr switchResource) registerSwitch(request *restful.Request, response *restful.Response) {
	var newSwitch metal.Switch
	err := request.ReadEntity(&newSwitch)
	if err != nil {
		sendError(sr.log, response, "registerSwitch", http.StatusInternalServerError, fmt.Errorf("cannot read switch from request: %v", err))
		return
	}
	_, err = sr.ds.FindSite(newSwitch.SiteID)
	if err != nil {
		sendError(sr.log, response, "registerSwitch", http.StatusInternalServerError, fmt.Errorf("Cannot find site %q: %v", newSwitch.SiteID, err))
		return
	}

	oldSwitch, err := sr.ds.FindSwitch(newSwitch.ID)
	if err != nil {
		newSwitch.Created = time.Now()
		newSwitch.Changed = newSwitch.Created
		sw, err := sr.ds.CreateSwitch(&newSwitch)
		if err != nil {
			sendError(sr.log, response, "registerSwitch", http.StatusInternalServerError, err)
			return
		} else {
			response.WriteHeaderAndEntity(http.StatusCreated, sw)
			return
		}
	}

	err = sr.ds.UpdateSwitch(oldSwitch, &newSwitch)

	if err != nil {
		sendError(sr.log, response, "registerSwitch", http.StatusConflict, err)
		return
	}
	response.WriteHeaderAndEntity(http.StatusOK, newSwitch)
}
