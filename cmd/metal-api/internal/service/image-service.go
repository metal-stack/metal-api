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

type imageResource struct {
	log15.Logger
	ds datastore.Datastore
}

func NewImage(log log15.Logger, ds datastore.Datastore) *restful.WebService {
	ir := imageResource{
		Logger: log,
		ds:     ds,
	}
	return ir.webService()
}

func (ir imageResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path("/image").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"image"}

	ws.Route(ws.GET("/{id}").To(ir.findImage).
		Doc("get image by id").
		Param(ws.PathParameter("id", "identifier of the image").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(metal.Image{}).
		Returns(http.StatusOK, "OK", metal.Image{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	ws.Route(ws.GET("/").To(ir.listImages).
		Doc("get all images").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]metal.Image{}).
		Returns(http.StatusOK, "OK", []metal.Image{}))

	ws.Route(ws.DELETE("/{id}").To(ir.deleteImage).
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

func (ir imageResource) findImage(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	image, err := ir.ds.FindImage(id)
	if err != nil {
		sendError(ir, response, "findImage", http.StatusNotFound, err)
		return
	}
	response.WriteEntity(image)
}

func (ir imageResource) listImages(request *restful.Request, response *restful.Response) {
	res, err := ir.ds.ListImages()
	if err != nil {
		sendError(ir, response, "listImages", http.StatusInternalServerError, err)
		return
	}
	response.WriteEntity(res)
}

func (ir imageResource) deleteImage(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	image, err := ir.ds.DeleteImage(id)
	if err != nil {
		sendError(ir, response, "deleteImage", http.StatusNotFound, err)
	} else {
		response.WriteEntity(image)
	}
}

func (ir imageResource) createImage(request *restful.Request, response *restful.Response) {
	var s metal.Image
	err := request.ReadEntity(&s)
	if err != nil {
		sendError(ir, response, "createImage", http.StatusInternalServerError, fmt.Errorf("cannot read image from request: %v", err))
		return
	}
	s.Created = time.Now()
	s.Changed = s.Created
	img, err := ir.ds.CreateImage(&s)
	if err != nil {
		sendError(ir, response, "createImage", http.StatusInternalServerError, err)
	} else {
		response.WriteHeaderAndEntity(http.StatusCreated, img)
	}
}

func (ir imageResource) updateImage(request *restful.Request, response *restful.Response) {
	var newImage metal.Image
	err := request.ReadEntity(&newImage)
	if err != nil {
		sendError(ir, response, "updateImage", http.StatusInternalServerError, fmt.Errorf("cannot read image from request: %v", err))
		return
	}

	oldImage, err := ir.ds.FindImage(newImage.ID)
	if err != nil {
		sendError(ir, response, "updateImage", http.StatusNotFound, err)
		return
	}

	err = ir.ds.UpdateImage(oldImage, &newImage)

	if err != nil {
		sendError(ir, response, "updateImage", http.StatusConflict, err)
		return
	}
	response.WriteHeaderAndEntity(http.StatusOK, newImage)
}
