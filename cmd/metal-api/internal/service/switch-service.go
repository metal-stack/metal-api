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

type switchRegistration struct {
	ID     string     `json:"id" description:"a unique ID" unique:"true"`
	Nics   metal.Nics `json:"nics" description:"the list of network interfaces on the switch"`
	SiteID string     `json:"site_id" description:"the id of the site in which this switch is located"`
	RackID string     `json:"rack_id" description:"the id of the rack in which this switch is located"`
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
		Reads(switchRegistration{}).
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
	var newSwitch switchRegistration
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
	sw := metal.NewSwitch(newSwitch.ID, newSwitch.SiteID, newSwitch.RackID, newSwitch.Nics)
	if err != nil {
		if metal.IsNotFound(err) {
			sw.Created = time.Now()
			sw.Changed = sw.Created
			sw, err = sr.ds.CreateSwitch(sw)
			if err != nil {
				sendError(sr.log, response, "registerSwitch", http.StatusInternalServerError, err)
				return
			}
			response.WriteHeaderAndEntity(http.StatusCreated, sw)
			return
		}
		sendError(sr.log, response, "registerSwitch", http.StatusInternalServerError, err)
		return
	}

	err = sr.ds.UpdateSwitch(oldSwitch, sw)

	if err != nil {
		sendError(sr.log, response, "registerSwitch", http.StatusConflict, err)
		return
	}
	response.WriteHeaderAndEntity(http.StatusOK, sw)
}
