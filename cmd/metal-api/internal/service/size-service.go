package service

import (
	"net/http"
	"time"

	"go.uber.org/zap"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/metal"
	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
)

type sizeResource struct {
	*zap.SugaredLogger
	log *zap.Logger
	ds  *datastore.RethinkStore
}

func NewSize(log *zap.Logger, ds *datastore.RethinkStore) *restful.WebService {
	sr := sizeResource{
		SugaredLogger: log.Sugar(),
		log:           log,
		ds:            ds,
	}
	return sr.webService()
}

func (sr sizeResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path("/size").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"size"}

	ws.Route(ws.GET("/{id}").To(sr.findSize).
		Doc("get size by id").
		Param(ws.PathParameter("id", "identifier of the size").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(metal.Size{}).
		Returns(http.StatusOK, "OK", metal.Image{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	ws.Route(ws.GET("/").To(sr.listSizes).
		Doc("get all sizes").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]metal.Size{}).
		Returns(http.StatusOK, "OK", []metal.Size{}))

	ws.Route(ws.DELETE("/{id}").To(sr.deleteSize).
		Doc("deletes an size and returns the deleted entity").
		Param(ws.PathParameter("id", "identifier of the size").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(metal.Size{}).
		Returns(http.StatusOK, "OK", metal.Size{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	ws.Route(ws.PUT("/").To(sr.createSize).
		Doc("create a size. if the given ID already exists a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(metal.Size{}).
		Returns(http.StatusCreated, "Created", metal.Size{}).
		Returns(http.StatusConflict, "Conflict", nil))

	ws.Route(ws.POST("/").To(sr.updateSize).
		Doc("updates a size. if the size was changed since this one was read, a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(metal.Size{}).
		Returns(http.StatusOK, "OK", metal.Size{}).
		Returns(http.StatusNotFound, "Not Found", nil).
		Returns(http.StatusConflict, "Conflict", nil))

	return ws
}

func (sr sizeResource) findSize(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	size, err := sr.ds.FindSize(id)
	if checkError(sr.log, response, "findSize", err) {
		return
	}
	response.WriteEntity(size)
}

func (sr sizeResource) listSizes(request *restful.Request, response *restful.Response) {
	res, err := sr.ds.ListSizes()
	if checkError(sr.log, response, "listSizes", err) {
		return
	}
	response.WriteEntity(res)
}

func (sr sizeResource) deleteSize(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	size, err := sr.ds.DeleteSize(id)
	if checkError(sr.log, response, "deleteSize", err) {
		return
	}
	response.WriteEntity(size)
}

func (sr sizeResource) createSize(request *restful.Request, response *restful.Response) {
	var s metal.Size
	err := request.ReadEntity(&s)
	if checkError(sr.log, response, "createSize", err) {
		return
	}
	s.Created = time.Now()
	s.Changed = s.Created
	err = sr.ds.CreateSize(&s)
	if checkError(sr.log, response, "createSize", err) {
		return
	}
	response.WriteHeaderAndEntity(http.StatusCreated, s)
}

func (sr sizeResource) updateSize(request *restful.Request, response *restful.Response) {
	var newSize metal.Size
	err := request.ReadEntity(&newSize)
	if checkError(sr.log, response, "updateSize", err) {
		return
	}

	oldSize, err := sr.ds.FindSize(newSize.ID)
	if checkError(sr.log, response, "updateSize", err) {
		return
	}

	err = sr.ds.UpdateSize(oldSize, &newSize)

	if checkError(sr.log, response, "updateSize", err) {
		return
	}
	response.WriteHeaderAndEntity(http.StatusOK, newSize)
}
