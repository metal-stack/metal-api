package service

import (
	"net/http"
	"time"

	"go.uber.org/zap"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
)

type imageResource struct {
	webResource
}

func NewImage(log *zap.Logger, ds *datastore.RethinkStore) *restful.WebService {
	ir := imageResource{
		webResource: webResource{
			SugaredLogger: log.Sugar(),
			log:           log,
			ds:            ds,
		},
	}
	return ir.webService()
}

func (ir imageResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path("/v1/image").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"image"}

	ws.Route(ws.GET("/{id}").
		To(ir.restEntityGet(ir.ds.FindImage)).
		Operation("findImage").
		Doc("get image by id").
		Param(ws.PathParameter("id", "identifier of the image").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(metal.Image{}).
		Returns(http.StatusOK, "OK", metal.Image{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	ws.Route(ws.GET("/").
		To(ir.restListGet(ir.ds.ListImages)).
		Operation("listImages").
		Doc("get all images").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]metal.Image{}).
		Returns(http.StatusOK, "OK", []metal.Image{}))

	ws.Route(ws.DELETE("/{id}").
		To(ir.restEntityGet(ir.ds.DeleteImage)).
		Operation("deleteImage").
		Doc("deletes an image and returns the deleted entity").
		Param(ws.PathParameter("id", "identifier of the image").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(metal.Image{}).
		Returns(http.StatusOK, "OK", metal.Image{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	ws.Route(ws.PUT("/").To(ir.createImage).
		Doc("create an image. if the given ID already exists a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(metal.Image{}).
		Returns(http.StatusCreated, "Created", metal.Image{}).
		Returns(http.StatusConflict, "Conflict", nil))

	ws.Route(ws.POST("/").To(ir.updateImage).
		Doc("updates an image. if the image was changed since this one was read, a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(metal.Image{}).
		Returns(http.StatusOK, "OK", metal.Image{}).
		Returns(http.StatusNotFound, "Not Found", nil).
		Returns(http.StatusConflict, "Conflict", nil))

	return ws
}

func (ir imageResource) createImage(request *restful.Request, response *restful.Response) {
	var s metal.Image
	err := request.ReadEntity(&s)
	if checkError(ir.log, response, "createImage", err) {
		return
	}
	s.Created = time.Now()
	s.Changed = s.Created
	img, err := ir.ds.CreateImage(&s)
	if checkError(ir.log, response, "createImage", err) {
		return
	}
	response.WriteHeaderAndEntity(http.StatusCreated, img)
}

func (ir imageResource) updateImage(request *restful.Request, response *restful.Response) {
	var newImage metal.Image
	err := request.ReadEntity(&newImage)
	if checkError(ir.log, response, "updateImage", err) {
		return
	}

	oldImage, err := ir.ds.FindImage(newImage.ID)
	if checkError(ir.log, response, "updateImage", err) {
		return
	}

	err = ir.ds.UpdateImage(oldImage, &newImage)

	if checkError(ir.log, response, "updateImage", err) {
		return
	}
	response.WriteHeaderAndEntity(http.StatusOK, newImage)
}
