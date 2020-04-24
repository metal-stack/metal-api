package service

import (
	"fmt"
	"net/http"
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/metal-lib/zapup"
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
	iuc := imageUsageCollector{ir: &ir}
	err := prometheus.Register(iuc)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to register prometheus", zap.Error(err))
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

	ws.Route(ws.GET("/migrate").
		To(admin(ir.migrateImages)).
		Operation("migrateImages").
		Doc("migrate existing machine allocation images to semver equivalents").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]v1.ImageResponse{}).
		Returns(http.StatusOK, "OK", []v1.ImageResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

// Migrate existing Images of allocations to semver images
// FIXME remove this after all machines are migrated.
func (ir imageResource) migrateImages(request *restful.Request, response *restful.Response) {
	_, err := ir.ds.MigrateMachineImages(nil)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
}

func (ir imageResource) findImage(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	img, err := ir.ds.GetImage(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, v1.NewImageResponse(img))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
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
	err = response.WriteHeaderAndEntity(http.StatusOK, result)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
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

	os, v, err := datastore.GetOsAndSemver(requestPayload.ID)
	if checkError(request, response, utils.CurrentFuncName(), err) {
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
			if checkError(request, response, utils.CurrentFuncName(), err) {
				return
			}
		}
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

	err = ir.ds.CreateImage(img)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	err = response.WriteHeaderAndEntity(http.StatusCreated, v1.NewImageResponse(img))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (ir imageResource) deleteImage(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	img, err := ir.ds.GetImage(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	machines, err := ir.ds.ListMachines()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	for _, m := range machines {
		if m.Allocation == nil {
			continue
		}
		if m.Allocation.ImageID == img.ID {
			if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("image %s is in use by machine:%s", img.ID, m.ID)) {
				return
			}
		}
	}

	err = ir.ds.DeleteImage(img)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, v1.NewImageResponse(img))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (ir imageResource) updateImage(request *restful.Request, response *restful.Response) {
	var requestPayload v1.ImageUpdateRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	oldImage, err := ir.ds.GetImage(requestPayload.ID)
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

	if requestPayload.Classification != nil {
		vc, err := metal.VersionClassificationFrom(*requestPayload.Classification)
		if err != nil {
			if checkError(request, response, utils.CurrentFuncName(), err) {
				return
			}
		}
		newImage.Classification = vc
	}

	if requestPayload.ExpirationDate != nil {
		newImage.ExpirationDate = *requestPayload.ExpirationDate
	}

	err = ir.ds.UpdateImage(oldImage, &newImage)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, v1.NewImageResponse(&newImage))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

// networkUsageCollector implements the prometheus collector interface.
type imageUsageCollector struct {
	ir *imageResource
}

var (
	usedImageDesc = prometheus.NewDesc(
		"metal_image_used_total",
		"The total number of machines using a image",
		[]string{"imageID"}, nil,
	)
)

func (iuc imageUsageCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(iuc, ch)
}

func (iuc imageUsageCollector) Collect(ch chan<- prometheus.Metric) {
	// FIXME bad workaround to be able to run make spec
	if iuc.ir == nil || iuc.ir.ds == nil {
		return
	}
	imgs, err := iuc.ir.ds.ListImages()
	if err != nil {
		return
	}
	// init with 0
	usage := make(map[string]int)
	for _, i := range imgs {
		usage[i.ID] = 0
	}
	// loop over machines and count
	machines, err := iuc.ir.ds.ListMachines()
	if err != nil {
		return
	}
	for _, m := range machines {
		if m.Allocation == nil {
			continue
		}
		usage[m.Allocation.ImageID]++
	}

	for image, count := range usage {

		metric, err := prometheus.NewConstMetric(
			usedImageDesc,
			prometheus.CounterValue,
			float64(count),
			image,
		)
		if err != nil {
			zapup.MustRootLogger().Error("Failed create metric for UsedImages", zap.Error(err))
			return
		}
		ch <- metric
	}
}
