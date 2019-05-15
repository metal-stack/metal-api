package service

import (
	"fmt"
	"net/http"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	v1 "git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/service/v1"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/utils"

	"git.f-i-ts.de/cloud-native/metallib/httperrors"
	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
)

type imageResource struct {
	webResource
}

// NewImage returns a webservice for image specific endpoints.
func NewImage(ds *datastore.RethinkStore) *restful.WebService {
	ir := imageResource{
		webResource: webResource{
			ds: ds,
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
		To(ir.findImage).
		Operation("findImage").
		Doc("get image by id").
		Param(ws.PathParameter("id", "identifier of the image").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.ImageResponse{}).
		Returns(http.StatusOK, "OK", v1.ImageResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/").
		To(ir.listImages).
		Operation("listImages").
		Doc("get all images").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]v1.ImageResponse{}).
		Returns(http.StatusOK, "OK", []v1.ImageResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.DELETE("/{id}").
		To(admin(ir.deleteImage)).
		Operation("deleteImage").
		Doc("deletes an image and returns the deleted entity").
		Param(ws.PathParameter("id", "identifier of the image").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.ImageResponse{}).
		Returns(http.StatusOK, "OK", v1.ImageResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.PUT("/").
		To(admin(ir.createImage)).
		Doc("create an image. if the given ID already exists a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.ImageCreateRequest{}).
		Returns(http.StatusCreated, "Created", v1.ImageResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/").
		To(admin(ir.updateImage)).
		Doc("updates an image. if the image was changed since this one was read, a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.ImageUpdateRequest{}).
		Returns(http.StatusOK, "OK", v1.ImageResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (ir imageResource) findImage(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	img, err := ir.ds.FindImage(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, v1.NewImageResponse(img))
}

func (ir imageResource) listImages(request *restful.Request, response *restful.Response) {
	imgs, err := ir.ds.ListImages()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	result := []*v1.ImageResponse{}
	for i := range imgs {
		result = append(result, v1.NewImageResponse(&imgs[i]))
	}

	response.WriteHeaderAndEntity(http.StatusOK, result)
}

func (ir imageResource) createImage(request *restful.Request, response *restful.Response) {
	var requestPayload v1.ImageCreateRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	if requestPayload.ID == "" {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("id should not be empty")) {
			return
		}
	}

	if requestPayload.URL == "" {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("url should not be empty")) {
			return
		}
	}

	var name string
	if requestPayload.Name != nil {
		name = *requestPayload.Name
	}
	var description string
	if requestPayload.Description != nil {
		description = *requestPayload.Description
	}

	features := make(map[metal.ImageFeatureType]bool)
	for _, f := range requestPayload.Features {
		ft, err := metal.ImageFeatureTypeFrom(f)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
		features[ft] = true
	}

	img := &metal.Image{
		Base: metal.Base{
			ID:          requestPayload.ID,
			Name:        name,
			Description: description,
		},
		URL:      requestPayload.URL,
		Features: features,
	}

	err = ir.ds.CreateImage(img)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusCreated, v1.NewImageResponse(img))
}

func (ir imageResource) deleteImage(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	img, err := ir.ds.FindImage(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = ir.ds.DeleteImage(img)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, v1.NewImageResponse(img))
}

func (ir imageResource) updateImage(request *restful.Request, response *restful.Response) {
	var requestPayload v1.ImageUpdateRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	oldImage, err := ir.ds.FindImage(requestPayload.ID)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	newImage := *oldImage

	if requestPayload.Name != nil {
		newImage.Name = *requestPayload.Name
	}
	if requestPayload.Description != nil {
		newImage.Description = *requestPayload.Description
	}
	if requestPayload.URL != nil {
		newImage.URL = *requestPayload.URL
	}
	features := make(map[metal.ImageFeatureType]bool)
	for _, f := range requestPayload.Features {
		ft, err := metal.ImageFeatureTypeFrom(f)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
		features[ft] = true
	}
	if len(features) > 0 {
		newImage.Features = features
	}

	err = ir.ds.UpdateImage(oldImage, &newImage)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, v1.NewImageResponse(&newImage))
}
