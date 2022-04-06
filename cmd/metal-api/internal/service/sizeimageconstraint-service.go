package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/utils"
	"go.uber.org/zap"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/metal-lib/zapup"
)

type sizeImageConstraintResource struct {
	webResource
}

// NewSize returns a webservice for size specific endpoints.
func NewSizeImageConstraint(ds *datastore.RethinkStore) *restful.WebService {
	r := sizeImageConstraintResource{
		webResource: webResource{
			ds: ds,
		},
	}
	return r.webService()
}

func (r sizeImageConstraintResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path(BasePath + "v1/size-image-constraint").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"sizeimageconstraint"}

	ws.Route(ws.GET("/{id}").
		To(r.findSizeImageConstraint).
		Operation("findSizeImageConstraint").
		Doc("get sizeimageconstraint by id").
		Param(ws.PathParameter("id", "identifier of the sizeimageconstraint").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.SizeImageConstraintResponse{}).
		Returns(http.StatusOK, "OK", v1.SizeImageConstraintResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/").
		To(r.listSizeImageConstraints).
		Operation("listSizeImageConstraints").
		Doc("get all sizeimageconstraints").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]v1.SizeImageConstraintResponse{}).
		Returns(http.StatusOK, "OK", []v1.SizeImageConstraintResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.DELETE("/{id}").
		To(admin(r.deleteSizeImageConstraint)).
		Operation("deleteSizeImageConstraint").
		Doc("deletes an sizeimageconstraint and returns the deleted entity").
		Param(ws.PathParameter("id", "identifier of the size").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.SizeImageConstraintResponse{}).
		Returns(http.StatusOK, "OK", v1.SizeImageConstraintResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.PUT("/").
		To(admin(r.createSizeImageConstraint)).
		Operation("createSizeImageConstraint").
		Doc("create a sizeimageconstraint. if the given ID already exists a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.SizeImageConstraintCreateRequest{}).
		Returns(http.StatusCreated, "Created", v1.SizeImageConstraintResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/").
		To(admin(r.updateSizeImageConstraint)).
		Operation("updateSizeImageConstraint").
		Doc("updates a sizeimageconstraint. if the sizeimageconstraint was changed since this one was read, a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.SizeImageConstraintUpdateRequest{}).
		Returns(http.StatusOK, "OK", v1.SizeImageConstraintResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/try").
		To(admin(r.trySizeImageConstraint)).
		Operation("trySizeImageConstraint").
		Doc("try if the given combination of image and size is supported and possible to allocate").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.SizeImageConstraintTryRequest{}).
		Returns(http.StatusOK, "OK", v1.EmptyBody{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r sizeImageConstraintResource) findSizeImageConstraint(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	s, err := r.ds.FindSizeImageConstraint(request.Request.Context(), id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, v1.NewSizeImageConstraintResponse(s))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r sizeImageConstraintResource) listSizeImageConstraints(request *restful.Request, response *restful.Response) {
	ss, err := r.ds.ListSizeImageConstraints(request.Request.Context())
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	result := []*v1.SizeImageConstraintResponse{}
	for i := range ss {
		result = append(result, v1.NewSizeImageConstraintResponse(&ss[i]))
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, result)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r sizeImageConstraintResource) createSizeImageConstraint(request *restful.Request, response *restful.Response) {
	var requestPayload v1.SizeImageConstraintCreateRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	if requestPayload.ID == "" {
		if checkError(request, response, utils.CurrentFuncName(), errors.New("id should not be empty")) {
			return
		}
	}

	s := v1.NewSizeImageConstraint(requestPayload)

	err = s.Validate()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = r.ds.CreateSizeImageConstraint(request.Request.Context(), s)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	err = response.WriteHeaderAndEntity(http.StatusCreated, v1.NewSizeImageConstraintResponse(s))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r sizeImageConstraintResource) deleteSizeImageConstraint(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	s, err := r.ds.FindSizeImageConstraint(request.Request.Context(), id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = r.ds.DeleteSizeImageConstraint(request.Request.Context(), s)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, v1.NewSizeImageConstraintResponse(s))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r sizeImageConstraintResource) updateSizeImageConstraint(request *restful.Request, response *restful.Response) {
	var requestPayload v1.SizeImageConstraintUpdateRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	old, err := r.ds.FindSizeImageConstraint(request.Request.Context(), requestPayload.ID)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	newSizeImageConstraint := *old

	if requestPayload.Name != nil {
		newSizeImageConstraint.Name = *requestPayload.Name
	}
	if requestPayload.Description != nil {
		newSizeImageConstraint.Description = *requestPayload.Description
	}

	newSizeImageConstraint.Images = requestPayload.Images

	err = newSizeImageConstraint.Validate()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = r.ds.UpdateSizeImageConstraint(request.Request.Context(), old, &newSizeImageConstraint)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, v1.NewSizeImageConstraintResponse(&newSizeImageConstraint))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}
func (r sizeImageConstraintResource) trySizeImageConstraint(request *restful.Request, response *restful.Response) {
	var requestPayload v1.SizeImageConstraintTryRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	if requestPayload.SizeID == "" || requestPayload.ImageID == "" {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("size and image must be given")) {
			return
		}
	}

	size, err := r.ds.FindSize(request.Request.Context(), requestPayload.SizeID)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	image, err := r.ds.FindImage(request.Request.Context(), requestPayload.ImageID)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = isSizeAndImageCompatible(request.Request.Context(), r.ds, *size, *image)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = response.WriteHeaderAndEntity(http.StatusOK, v1.EmptyBody{})
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func isSizeAndImageCompatible(ctx context.Context, ds *datastore.RethinkStore, size metal.Size, image metal.Image) error {
	sic, err := ds.FindSizeImageConstraint(ctx, size.ID)
	if err != nil && !metal.IsNotFound(err) {
		return err
	}
	if sic == nil {
		return nil
	}

	return sic.Matches(size, image)
}
