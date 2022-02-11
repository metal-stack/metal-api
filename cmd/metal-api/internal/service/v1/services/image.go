package services

import (
	"net/http"
	"strconv"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/service"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/utils"
	"go.uber.org/zap"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/metal-lib/zapup"
)

type imageResource struct {
	images *service.ImageService
	ds     *datastore.RethinkStore // TODO: to be removed when migrated all services
}

// NewImage returns a webservice for image specific endpoints.
func NewImage(ds *datastore.RethinkStore) *restful.WebService {
	ir := imageResource{
		images: service.NewImageService(ds),
		ds:     ds,
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

func (ir imageResource) findImage(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	img, err := ir.images.Get(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, v1.NewImageResponse(img))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (ir imageResource) queryImages(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	img, err := ir.images.Find(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	result := []*v1.ImageResponse{}

	for i := range img {
		result = append(result, v1.NewImageResponse(&img[i]))
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, result)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (ir imageResource) findLatestImage(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	img, err := ir.images.FindLatest(id)
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
	imgs, err := ir.images.List()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	ms := metal.Machines{}
	showUsage := false
	if request.QueryParameter("show-usage") != "" {
		showUsage, err = strconv.ParseBool(request.QueryParameter("show-usage"))
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
		if showUsage {
			ms, err = ir.ds.ListMachines()
			if checkError(request, response, utils.CurrentFuncName(), err) {
				return
			}
		}
	}

	result := []*v1.ImageResponse{}
	for i := range imgs {
		img := v1.NewImageResponse(&imgs[i])
		if showUsage {
			machines := ir.machinesByImage(ms, imgs[i].ID)
			if len(machines) > 0 {
				img.UsedBy = machines
			}
		}
		result = append(result, img)
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
			ID: requestPayload.ID,
		},
		URL:      requestPayload.URL,
		Features: features,
	}

	if requestPayload.Name != nil {
		img.Name = *requestPayload.Name
	}
	if requestPayload.Description != nil {
		img.Description = *requestPayload.Description
	}
	if requestPayload.ExpirationDate != nil && !requestPayload.ExpirationDate.IsZero() {
		img.ExpirationDate = *requestPayload.ExpirationDate
	}
	if requestPayload.Classification != nil {
		vc, err := metal.VersionClassificationFrom(*requestPayload.Classification)
		if err != nil {
			if checkError(request, response, utils.CurrentFuncName(), err) {
				return
			}
		}
		img.Classification = vc
	}

	err = ir.images.Create(img)
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

	img, err := ir.images.Delete(id)
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

	img := &metal.Image{
		Base: metal.Base{
			ID: requestPayload.ID,
		},
	}

	if requestPayload.Name != nil {
		img.Name = *requestPayload.Name
	}
	if requestPayload.Description != nil {
		img.Description = *requestPayload.Description
	}
	if requestPayload.URL != nil {
		img.URL = *requestPayload.URL
	}
	if requestPayload.ExpirationDate != nil && !requestPayload.ExpirationDate.IsZero() {
		img.ExpirationDate = *requestPayload.ExpirationDate
	}
	if requestPayload.Classification != nil {
		vc, err := metal.VersionClassificationFrom(*requestPayload.Classification)
		if err != nil {
			if checkError(request, response, utils.CurrentFuncName(), err) {
				return
			}
		}
		img.Classification = vc
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
		img.Features = features
	}

	if requestPayload.Classification != nil {
		vc, err := metal.VersionClassificationFrom(*requestPayload.Classification)
		if err != nil {
			if checkError(request, response, utils.CurrentFuncName(), err) {
				return
			}
		}
		img.Classification = vc
	}

	img, err = ir.images.Update(img)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, v1.NewImageResponse(img))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (ir imageResource) machinesByImage(machines metal.Machines, imageID string) []string {
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
