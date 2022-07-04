package service

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"go.uber.org/zap"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-lib/httperrors"
)

type sizeImageConstraintResource struct {
	webResource
}

// NewSize returns a webservice for size specific endpoints.
func NewSizeImageConstraint(log *zap.SugaredLogger, ds *datastore.RethinkStore) *restful.WebService {
	r := sizeImageConstraintResource{
		webResource: webResource{
			log: log,
			ds:  ds,
		},
	}
	return r.webService()
}

func (r *sizeImageConstraintResource) webService() *restful.WebService {
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

func (r *sizeImageConstraintResource) findSizeImageConstraint(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	s, err := r.ds.FindSizeImageConstraint(id)
	if err != nil {
		r.SendError(response, DefaultError(err))
		return
	}

	r.Send(response, http.StatusOK, v1.NewSizeImageConstraintResponse(s))
}

func (r *sizeImageConstraintResource) listSizeImageConstraints(request *restful.Request, response *restful.Response) {
	ss, err := r.ds.ListSizeImageConstraints()
	if err != nil {
		r.SendError(response, DefaultError(err))
		return
	}

	result := []*v1.SizeImageConstraintResponse{}
	for i := range ss {
		result = append(result, v1.NewSizeImageConstraintResponse(&ss[i]))
	}

	r.Send(response, http.StatusOK, result)
}

func (r *sizeImageConstraintResource) createSizeImageConstraint(request *restful.Request, response *restful.Response) {
	var requestPayload v1.SizeImageConstraintCreateRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.SendError(response, httperrors.BadRequest(err))
		return
	}

	if requestPayload.ID == "" {
		r.SendError(response, httperrors.BadRequest(errors.New("id should not be empty")))
		return
	}

	s := v1.NewSizeImageConstraint(requestPayload)

	err = s.Validate()
	if err != nil {
		r.SendError(response, httperrors.BadRequest(err))
		return
	}

	err = r.ds.CreateSizeImageConstraint(s)
	if err != nil {
		r.SendError(response, DefaultError(err))
		return
	}

	r.Send(response, http.StatusCreated, v1.NewSizeImageConstraintResponse(s))
}

func (r *sizeImageConstraintResource) deleteSizeImageConstraint(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	s, err := r.ds.FindSizeImageConstraint(id)
	if err != nil {
		r.SendError(response, DefaultError(err))
		return
	}

	err = r.ds.DeleteSizeImageConstraint(s)
	if err != nil {
		r.SendError(response, DefaultError(err))
		return
	}

	r.Send(response, http.StatusOK, v1.NewSizeImageConstraintResponse(s))
}

func (r *sizeImageConstraintResource) updateSizeImageConstraint(request *restful.Request, response *restful.Response) {
	var requestPayload v1.SizeImageConstraintUpdateRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.SendError(response, httperrors.BadRequest(err))
		return
	}

	old, err := r.ds.FindSizeImageConstraint(requestPayload.ID)
	if err != nil {
		r.SendError(response, DefaultError(err))
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
	if err != nil {
		r.SendError(response, httperrors.BadRequest(err))
		return
	}

	err = r.ds.UpdateSizeImageConstraint(old, &newSizeImageConstraint)
	if err != nil {
		r.SendError(response, DefaultError(err))
		return
	}

	r.Send(response, http.StatusOK, v1.NewSizeImageConstraintResponse(&newSizeImageConstraint))
}

func (r *sizeImageConstraintResource) trySizeImageConstraint(request *restful.Request, response *restful.Response) {
	var requestPayload v1.SizeImageConstraintTryRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.SendError(response, httperrors.BadRequest(err))
		return
	}

	if requestPayload.SizeID == "" || requestPayload.ImageID == "" {
		r.SendError(response, httperrors.BadRequest(fmt.Errorf("size and image must be given")))
		return
	}

	size, err := r.ds.FindSize(requestPayload.SizeID)
	if err != nil {
		r.SendError(response, DefaultError(err))
		return
	}

	image, err := r.ds.FindImage(requestPayload.ImageID)
	if err != nil {
		r.SendError(response, DefaultError(err))
		return
	}

	err = isSizeAndImageCompatible(r.ds, *size, *image)
	if err != nil {
		r.SendError(response, httperrors.NewHTTPError(http.StatusUnprocessableEntity, err))
		return
	}

	r.Send(response, http.StatusOK, v1.EmptyBody{})
}

func isSizeAndImageCompatible(ds *datastore.RethinkStore, size metal.Size, image metal.Image) error {
	sic, err := ds.FindSizeImageConstraint(size.ID)
	if err != nil && !metal.IsNotFound(err) {
		return err
	}
	if sic == nil {
		return nil
	}

	return sic.Matches(size, image)
}
