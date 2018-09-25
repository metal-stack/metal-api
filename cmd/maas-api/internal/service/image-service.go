package service

import (
	"fmt"
	"net/http"
	"time"

	"git.f-i-ts.de/cloud-native/maas/maas-service/cmd/maas-api/internal/datastore"

	"git.f-i-ts.de/cloud-native/maas/maas-service/pkg/maas"
	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
)

type imageResource struct {
	ds datastore.Datastore
}

func NewImage(ds datastore.Datastore) *restful.WebService {
	ir := imageResource{
		ds: ds,
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
		Writes(maas.Image{}).
		Returns(http.StatusOK, "OK", maas.Image{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	ws.Route(ws.GET("/").To(ir.listImages).
		Doc("get all images").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]maas.Image{}).
		Returns(http.StatusOK, "OK", []maas.Image{}))

	ws.Route(ws.DELETE("/{id}").To(ir.deleteImage).
		Doc("deletes an image and returns the deleted entity").
		Param(ws.PathParameter("id", "identifier of the image").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(maas.Image{}).
		Returns(http.StatusOK, "OK", maas.Image{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	ws.Route(ws.PUT("/").To(ir.createImage).
		Doc("create an image. if the given ID already exists a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(maas.Image{}).
		Returns(http.StatusCreated, "Created", maas.Image{}).
		Returns(http.StatusConflict, "Conflict", nil))

	ws.Route(ws.POST("/").To(ir.updateImage).
		Doc("updates an image. if the image was changed since this one was read, a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(maas.Image{}).
		Returns(http.StatusOK, "OK", maas.Image{}).
		Returns(http.StatusNotFound, "Not Found", nil).
		Returns(http.StatusConflict, "Conflict", nil))

	return ws
}

func (ir imageResource) findImage(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	image, err := ir.ds.FindImage(id)
	if err != nil {
		response.WriteError(http.StatusNotFound, err)
	}
	response.WriteEntity(image)
}

func (ir imageResource) listImages(request *restful.Request, response *restful.Response) {
	res := ir.ds.ListImages()
	response.WriteEntity(res)
}

func (ir imageResource) deleteImage(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	image, err := ir.ds.DeleteImage(id)
	if err != nil {
		response.WriteError(http.StatusNotFound, err)
	} else {
		response.WriteEntity(image)
	}
}

func (ir imageResource) createImage(request *restful.Request, response *restful.Response) {
	var s maas.Image
	err := request.ReadEntity(&s)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, fmt.Errorf("cannot read image from request: %v", err))
		return
	}
	s.Created = time.Now()
	s.Changed = s.Created
	err = ir.ds.CreateImage(&s)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, fmt.Errorf("cannot create image: %v", err))
	} else {
		response.WriteHeaderAndEntity(http.StatusCreated, s)
	}
}

func (ir imageResource) updateImage(request *restful.Request, response *restful.Response) {
	var newImage maas.Image
	err := request.ReadEntity(&newImage)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, fmt.Errorf("cannot read image from request: %v", err))
		return
	}

	oldImage, err := ir.ds.FindImage(newImage.ID)
	if err != nil {
		response.WriteError(http.StatusNotFound, err)
		return
	}

	err = ir.ds.UpdateImage(oldImage, &newImage)

	if err != nil {
		response.WriteError(http.StatusConflict, err)
	}
	response.WriteHeaderAndEntity(http.StatusOK, newImage)
}
