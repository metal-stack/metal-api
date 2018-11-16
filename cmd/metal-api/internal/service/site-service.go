package service

import (
	"fmt"
	"net/http"
	"time"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/metal"
	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/inconshreveable/log15"
)

type SiteResource struct {
	log15.Logger
	ds *datastore.RethinkStore
}

func NewSite(log log15.Logger, ds *datastore.RethinkStore) *restful.WebService {
	fr := SiteResource{
		Logger: log,
		ds:     ds,
	}
	return fr.webService()
}

func (fr SiteResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path("/site").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"Site"}

	ws.Route(ws.GET("/{id}").To(fr.findSite).
		Doc("get Site by id").
		Param(ws.PathParameter("id", "identifier of the Site").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(metal.Site{}).
		Returns(http.StatusOK, "OK", metal.Site{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	ws.Route(ws.GET("/").To(fr.listSites).
		Doc("get all Sites").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]metal.Site{}).
		Returns(http.StatusOK, "OK", []metal.Site{}))

	ws.Route(ws.DELETE("/{id}").To(fr.deleteSite).
		Doc("deletes a Site and returns the deleted entity").
		Param(ws.PathParameter("id", "identifier of the Site").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(metal.Site{}).
		Returns(http.StatusOK, "OK", metal.Site{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	ws.Route(ws.PUT("/").To(fr.createSite).
		Doc("create a Site. if the given ID already exists a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(metal.Site{}).
		Returns(http.StatusCreated, "Created", metal.Site{}).
		Returns(http.StatusConflict, "Conflict", nil))

	ws.Route(ws.POST("/").To(fr.updateSite).
		Doc("updates a Site. if the Site was changed since this one was read, a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(metal.Site{}).
		Returns(http.StatusOK, "OK", metal.Site{}).
		Returns(http.StatusNotFound, "Not Found", nil).
		Returns(http.StatusConflict, "Conflict", nil))

	return ws
}

func (fr SiteResource) findSite(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	Site, err := fr.ds.FindSite(id)
	if err != nil {
		sendError(fr, response, "findSite", http.StatusNotFound, err)
		return
	}
	response.WriteEntity(Site)
}

func (fr SiteResource) listSites(request *restful.Request, response *restful.Response) {
	res, err := fr.ds.ListSites()
	if err != nil {
		sendError(fr, response, "listSites", http.StatusInternalServerError, err)
		return
	}
	response.WriteEntity(res)
}

func (fr SiteResource) deleteSite(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	Site, err := fr.ds.DeleteSite(id)
	if err != nil {
		sendError(fr, response, "deleteFaility", http.StatusNotFound, err)
	} else {
		response.WriteEntity(Site)
	}
}

func (fr SiteResource) createSite(request *restful.Request, response *restful.Response) {
	var s metal.Site
	err := request.ReadEntity(&s)
	if err != nil {
		sendError(fr, response, "createSite", http.StatusInternalServerError, fmt.Errorf("cannot read Site from request: %v", err))
		return
	}
	s.Created = time.Now()
	s.Changed = s.Created
	err = fr.ds.CreateSite(&s)
	if err != nil {
		sendError(fr, response, "createSite", http.StatusInternalServerError, fmt.Errorf("cannot create Site: %v", err))
	} else {
		response.WriteHeaderAndEntity(http.StatusCreated, s)
	}
}

func (fr SiteResource) updateSite(request *restful.Request, response *restful.Response) {
	var newSite metal.Site
	err := request.ReadEntity(&newSite)
	if err != nil {
		sendError(fr, response, "updateSite", http.StatusInternalServerError, fmt.Errorf("cannot read Site from request: %v", err))
		return
	}

	oldSite, err := fr.ds.FindSite(newSite.ID)
	if err != nil {
		sendError(fr, response, "updateSite", http.StatusNotFound, err)
		return
	}

	err = fr.ds.UpdateSite(oldSite, &newSite)

	if err != nil {
		sendError(fr, response, "updateSite", http.StatusConflict, err)
		return
	}
	response.WriteHeaderAndEntity(http.StatusOK, newSite)
}
