package service

import (
	"fmt"
	"net/http"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/utils"
	"go.uber.org/zap"

	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/metal-lib/zapup"
)

type sizeResource struct {
	webResource
}

// NewSize returns a webservice for size specific endpoints.
func NewSize(ds *datastore.RethinkStore) *restful.WebService {
	r := sizeResource{
		webResource: webResource{
			ds: ds,
		},
	}
	return r.webService()
}

func (r sizeResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path(BasePath + "v1/size").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"size"}

	ws.Route(ws.GET("/{id}").
		To(r.findSize).
		Operation("findSize").
		Doc("get size by id").
		Param(ws.PathParameter("id", "identifier of the size").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.SizeResponse{}).
		Returns(http.StatusOK, "OK", v1.SizeResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/").
		To(r.listSizes).
		Operation("listSizes").
		Doc("get all sizes").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]v1.SizeResponse{}).
		Returns(http.StatusOK, "OK", []v1.SizeResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.DELETE("/{id}").
		To(admin(r.deleteSize)).
		Operation("deleteSize").
		Doc("deletes an size and returns the deleted entity").
		Param(ws.PathParameter("id", "identifier of the size").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.SizeResponse{}).
		Returns(http.StatusOK, "OK", v1.SizeResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.PUT("/").
		To(admin(r.createSize)).
		Operation("createSize").
		Doc("create a size. if the given ID already exists a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.SizeCreateRequest{}).
		Returns(http.StatusCreated, "Created", v1.SizeResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/").
		To(admin(r.updateSize)).
		Operation("updateSize").
		Doc("updates a size. if the size was changed since this one was read, a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.SizeUpdateRequest{}).
		Returns(http.StatusOK, "OK", v1.SizeResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/from-hardware").
		To(r.fromHardware).
		Operation("fromHardware").
		Doc("Searches all sizes for one to match the given hardwarespecs. If nothing is found, a list of entries is returned which describe the constraint which did not match").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.MachineHardwareExtended{}).
		Returns(http.StatusOK, "OK", v1.SizeMatchingLog{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r sizeResource) findSize(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	s, err := r.ds.FindSize(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, v1.NewSizeResponse(s))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r sizeResource) listSizes(request *restful.Request, response *restful.Response) {
	ss, err := r.ds.ListSizes()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	var result []*v1.SizeResponse
	for i := range ss {
		result = append(result, v1.NewSizeResponse(&ss[i]))
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, result)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r sizeResource) createSize(request *restful.Request, response *restful.Response) {
	var requestPayload v1.SizeCreateRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	if requestPayload.ID == "" {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("id should not be empty")) {
			return
		}
	}

	if requestPayload.ID == metal.UnknownSize.GetID() {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("id cannot be %q", metal.UnknownSize.GetID())) {
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
	var constraints []metal.Constraint
	for _, c := range requestPayload.SizeConstraints {
		constraint := metal.Constraint{
			Type: c.Type,
			Min:  c.Min,
			Max:  c.Max,
		}
		constraints = append(constraints, constraint)
	}

	s := &metal.Size{
		Base: metal.Base{
			ID:          requestPayload.ID,
			Name:        name,
			Description: description,
		},
		Constraints: constraints,
	}

	err = r.ds.CreateSize(s)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	err = response.WriteHeaderAndEntity(http.StatusCreated, v1.NewSizeResponse(s))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r sizeResource) deleteSize(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	s, err := r.ds.FindSize(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = r.ds.DeleteSize(s)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, v1.NewSizeResponse(s))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r sizeResource) updateSize(request *restful.Request, response *restful.Response) {
	var requestPayload v1.SizeUpdateRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	oldSize, err := r.ds.FindSize(requestPayload.ID)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	newSize := *oldSize

	if requestPayload.Name != nil {
		newSize.Name = *requestPayload.Name
	}
	if requestPayload.Description != nil {
		newSize.Description = *requestPayload.Description
	}
	var constraints []metal.Constraint
	if requestPayload.SizeConstraints != nil {
		sizeConstraints := *requestPayload.SizeConstraints
		for i := range sizeConstraints {
			constraint := metal.Constraint{
				Type: sizeConstraints[i].Type,
				Min:  sizeConstraints[i].Min,
				Max:  sizeConstraints[i].Max,
			}
			constraints = append(constraints, constraint)
		}
		newSize.Constraints = constraints
	}

	err = r.ds.UpdateSize(oldSize, &newSize)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, v1.NewSizeResponse(&newSize))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r sizeResource) fromHardware(request *restful.Request, response *restful.Response) {
	var requestPayload v1.MachineHardwareExtended
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	hw := v1.NewMetalMachineHardware(&requestPayload)
	_, lg, err := r.ds.FromHardware(hw)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	if len(lg) < 1 {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("size matching log is empty")) {
			return
		}
	}

	err = response.WriteHeaderAndEntity(http.StatusOK, v1.NewSizeMatchingLog(lg[0]))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}
