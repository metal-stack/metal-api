package service

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/utils"
	"go.uber.org/zap"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-lib/httperrors"
)

type imageResource struct {
	webResource
}

// NewImage returns a webservice for image specific endpoints.
func NewImage(log *zap.SugaredLogger, ds *datastore.RethinkStore) *restful.WebService {
	ir := imageResource{
		webResource: webResource{
			log: log,
			ds:  ds,
		},
	}

	return ir.webService()
}

func (ir imageResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path(BasePath + "v1/image").
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

	ws.Route(ws.GET("/{id}/query").
		To(ir.queryImages).
		Operation("queryImages by id").
		Doc("query all images which match at least id").
		Param(ws.PathParameter("id", "identifier of the image").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]v1.ImageResponse{}).
		Returns(http.StatusOK, "OK", []v1.ImageResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/{id}/latest").
		To(ir.findLatestImage).
		Operation("findLatestImage").
		Doc("find latest image by id").
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
		Param(ws.QueryParameter("show-usage", "include image usage into response").DataType("bool").DefaultValue("false")).
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
		Operation("createImage").
		Doc("create an image. if the given ID already exists a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.ImageCreateRequest{}).
		Returns(http.StatusCreated, "Created", v1.ImageResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/").
		To(admin(ir.updateImage)).
		Operation("updateImage").
		Doc("updates an image. if the image was changed since this one was read, a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.ImageUpdateRequest{}).
		Returns(http.StatusOK, "OK", v1.ImageResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r *imageResource) findImage(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	img, err := r.ds.GetImage(id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, v1.NewImageResponse(img))
}

func (r *imageResource) queryImages(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	img, err := r.ds.FindImages(id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	result := []*v1.ImageResponse{}

	for i := range img {
		result = append(result, v1.NewImageResponse(&img[i]))
	}

	r.send(request, response, http.StatusOK, result)
}

func (r *imageResource) findLatestImage(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	img, err := r.ds.FindImage(id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, v1.NewImageResponse(img))
}

func (r *imageResource) listImages(request *restful.Request, response *restful.Response) {
	imgs, err := r.ds.ListImages()
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	ms := metal.Machines{}
	showUsage := false
	if request.QueryParameter("show-usage") != "" {
		showUsage, err = strconv.ParseBool(request.QueryParameter("show-usage"))
		if err != nil {
			r.sendError(request, response, httperrors.BadRequest(err))
			return
		}

		if showUsage {
			ms, err = r.ds.ListMachines()
			if err != nil {
				r.sendError(request, response, defaultError(err))
				return
			}
		}
	}

	result := []*v1.ImageResponse{}
	for i := range imgs {
		img := v1.NewImageResponse(&imgs[i])
		if showUsage {
			machines := r.machinesByImage(ms, imgs[i].ID)
			if len(machines) > 0 {
				img.UsedBy = machines
			}
		}
		result = append(result, img)
	}

	r.send(request, response, http.StatusOK, result)
}

func (r *imageResource) createImage(request *restful.Request, response *restful.Response) {
	var requestPayload v1.ImageCreateRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	if requestPayload.ID == "" {
		r.sendError(request, response, httperrors.BadRequest(errors.New("id should not be empty")))
		return
	}

	if requestPayload.URL == "" {
		r.sendError(request, response, httperrors.BadRequest(errors.New("url should not be empty")))
		return
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
		if err != nil {
			r.sendError(request, response, httperrors.BadRequest(err))
			return
		}

		features[ft] = true
	}

	os, v, err := utils.GetOsAndSemverFromImage(requestPayload.ID)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	expirationDate := time.Now().Add(metal.DefaultImageExpiration)
	if requestPayload.ExpirationDate != nil && !requestPayload.ExpirationDate.IsZero() {
		expirationDate = *requestPayload.ExpirationDate
	}

	vc := metal.ClassificationPreview
	if requestPayload.Classification != nil {
		vc, err = metal.VersionClassificationFrom(*requestPayload.Classification)
		if err != nil {
			r.sendError(request, response, httperrors.BadRequest(err))
			return
		}
	}

	err = checkImageURL(requestPayload.ID, requestPayload.URL)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	img := &metal.Image{
		Base: metal.Base{
			ID:          requestPayload.ID,
			Name:        name,
			Description: description,
		},
		URL:            requestPayload.URL,
		Features:       features,
		OS:             os,
		Version:        v.String(),
		ExpirationDate: expirationDate,
		Classification: vc,
	}

	err = r.ds.CreateImage(img)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusCreated, v1.NewImageResponse(img))
}

func checkImageURL(id, url string) error {
	// nolint
	res, err := http.Head(url)
	if err != nil {
		return fmt.Errorf("image:%s is not accessible under:%s error:%w", id, url, err)
	}
	if res.StatusCode >= 400 {
		return fmt.Errorf("image:%s is not accessible under:%s status:%s", id, url, res.Status)
	}
	return nil
}

func (r *imageResource) deleteImage(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	img, err := r.ds.GetImage(id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	ms, err := r.ds.ListMachines()
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	machines := r.machinesByImage(ms, img.ID)
	if len(machines) > 0 {
		if err != nil {
			r.sendError(request, response, httperrors.UnprocessableEntity(fmt.Errorf("image %s is in use by machines:%v", img.ID, machines)))
			return
		}
	}

	err = r.ds.DeleteImage(img)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, v1.NewImageResponse(img))
}

func (r *imageResource) updateImage(request *restful.Request, response *restful.Response) {
	var requestPayload v1.ImageUpdateRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	oldImage, err := r.ds.GetImage(requestPayload.ID)
	if err != nil {
		r.sendError(request, response, defaultError(err))
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
		err = checkImageURL(requestPayload.ID, *requestPayload.URL)
		if err != nil {
			r.sendError(request, response, httperrors.BadRequest(err))
			return
		}

		newImage.URL = *requestPayload.URL
	}
	features := make(map[metal.ImageFeatureType]bool)
	for _, f := range requestPayload.Features {
		ft, err := metal.ImageFeatureTypeFrom(f)
		if err != nil {
			r.sendError(request, response, httperrors.BadRequest(err))
			return
		}

		features[ft] = true
	}
	if len(features) > 0 {
		newImage.Features = features
	}

	if requestPayload.Classification != nil {
		vc, err := metal.VersionClassificationFrom(*requestPayload.Classification)
		if err != nil {
			r.sendError(request, response, httperrors.BadRequest(err))
			return
		}

		newImage.Classification = vc
	}

	if requestPayload.ExpirationDate != nil {
		newImage.ExpirationDate = *requestPayload.ExpirationDate
	}

	err = r.ds.UpdateImage(oldImage, &newImage)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, v1.NewImageResponse(&newImage))
}

func (r *imageResource) machinesByImage(machines metal.Machines, imageID string) []string {
	var machinesByImage []string
	for _, m := range machines {
		if m.Allocation == nil {
			continue
		}
		if m.Allocation.ImageID == imageID {
			machinesByImage = append(machinesByImage, m.ID)
		}
	}
	return machinesByImage
}
