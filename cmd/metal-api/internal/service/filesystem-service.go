package service

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"go.uber.org/zap"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-lib/httperrors"
)

type filesystemResource struct {
	webResource
}

// NewFilesystemLayout returns a webservice for filesystem specific endpoints.
func NewFilesystemLayout(log *zap.SugaredLogger, ds *datastore.RethinkStore) *restful.WebService {
	r := filesystemResource{
		webResource: webResource{
			log: log,
			ds:  ds,
		},
	}
	return r.webService()
}

func (r *filesystemResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path(BasePath + "v1/filesystemlayout").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"filesystemlayout"}

	ws.Route(ws.GET("/{id}").
		To(viewer(r.findFilesystemLayout)).
		Operation("getFilesystemLayout").
		Doc("get filesystemlayout by id").
		Param(ws.PathParameter("id", "identifier of the filesystemlayout").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.FilesystemLayoutResponse{}).
		Returns(http.StatusOK, "OK", v1.FilesystemLayoutResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/").
		To(viewer(r.listFilesystemLayouts)).
		Operation("listFilesystemLayouts").
		Doc("get all filesystemlayouts").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]v1.FilesystemLayoutResponse{}).
		Returns(http.StatusOK, "OK", []v1.FilesystemLayoutResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.DELETE("/{id}").
		To(admin(r.deleteFilesystemLayout)).
		Operation("deleteFilesystemLayout").
		Doc("deletes an filesystemlayout and returns the deleted entity").
		Param(ws.PathParameter("id", "identifier of the filesystemlayout").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.FilesystemLayoutResponse{}).
		Returns(http.StatusOK, "OK", v1.FilesystemLayoutResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.PUT("/").
		To(admin(r.createFilesystemLayout)).
		Operation("createFilesystemLayout").
		Doc("create a filesystemlayout. if the given ID already exists a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.FilesystemLayoutCreateRequest{}).
		Returns(http.StatusCreated, "Created", v1.FilesystemLayoutResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/").
		To(admin(r.updateFilesystemLayout)).
		Operation("updateFilesystemLayout").
		Doc("updates a filesystemlayout. if the filesystemlayout was changed since this one was read, a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.FilesystemLayoutUpdateRequest{}).
		Returns(http.StatusOK, "OK", v1.FilesystemLayoutResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/try").
		To(admin(r.tryFilesystemLayout)).
		Operation("tryFilesystemLayout").
		Doc("try to detect a filesystemlayout based on given size and image.").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.FilesystemLayoutTryRequest{}).
		Returns(http.StatusOK, "OK", v1.FilesystemLayoutResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/matches").
		To(admin(r.matchFilesystemLayout)).
		Operation("matchFilesystemLayout").
		Doc("check if the given machine id satisfies the disk requirements of the filesystemlayout in question").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.FilesystemLayoutMatchRequest{}).
		Returns(http.StatusOK, "OK", v1.FilesystemLayoutResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r *filesystemResource) findFilesystemLayout(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	s, err := r.ds.FindFilesystemLayout(id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, v1.NewFilesystemLayoutResponse(s))
}

func (r *filesystemResource) listFilesystemLayouts(request *restful.Request, response *restful.Response) {
	ss, err := r.ds.ListFilesystemLayouts()
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	result := []*v1.FilesystemLayoutResponse{}
	for i := range ss {
		result = append(result, v1.NewFilesystemLayoutResponse(&ss[i]))
	}

	r.send(request, response, http.StatusOK, result)
}

func (r *filesystemResource) createFilesystemLayout(request *restful.Request, response *restful.Response) {
	var requestPayload v1.FilesystemLayoutCreateRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	if requestPayload.ID == "" {
		r.sendError(request, response, httperrors.BadRequest(errors.New("id should not be empty")))
		return
	}
	existing, _ := r.ds.FindFilesystemLayout(requestPayload.ID)
	if existing != nil {
		r.sendError(request, response, httperrors.Conflict(fmt.Errorf("filesystemlayout:%s already exists", existing.ID)))
		return
	}

	fsl, err := v1.NewFilesystemLayout(requestPayload)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	err = fsl.Validate()
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	fsls, err := r.ds.ListFilesystemLayouts()
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	fsls = append(fsls, *fsl)
	err = fsls.Validate()
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	err = r.ds.CreateFilesystemLayout(fsl)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusCreated, v1.NewFilesystemLayoutResponse(fsl))
}

func (r *filesystemResource) deleteFilesystemLayout(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	s, err := r.ds.FindFilesystemLayout(id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	err = r.ds.DeleteFilesystemLayout(s)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, v1.NewFilesystemLayoutResponse(s))
}

func (r *filesystemResource) updateFilesystemLayout(request *restful.Request, response *restful.Response) {
	var requestPayload v1.FilesystemLayoutUpdateRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	oldFilesystemLayout, err := r.ds.FindFilesystemLayout(requestPayload.ID)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	newFilesystemLayout, err := v1.NewFilesystemLayout(v1.FilesystemLayoutCreateRequest(requestPayload))
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	err = newFilesystemLayout.Validate()
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	fsls, err := r.ds.ListFilesystemLayouts()
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	fsls = append(fsls, *newFilesystemLayout)
	err = fsls.Validate()
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	err = r.ds.UpdateFilesystemLayout(oldFilesystemLayout, newFilesystemLayout)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, v1.NewFilesystemLayoutResponse(newFilesystemLayout))
}

func (r *filesystemResource) tryFilesystemLayout(request *restful.Request, response *restful.Response) {
	var try v1.FilesystemLayoutTryRequest
	err := request.ReadEntity(&try)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	ss, err := r.ds.ListFilesystemLayouts()
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	fsl, err := ss.From(try.Size, try.Image)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, v1.NewFilesystemLayoutResponse(fsl))
}

func (r *filesystemResource) matchFilesystemLayout(request *restful.Request, response *restful.Response) {
	var match v1.FilesystemLayoutMatchRequest
	err := request.ReadEntity(&match)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	fsl, err := r.ds.FindFilesystemLayout(match.FilesystemLayout)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	machine, err := r.ds.FindMachineByID(match.Machine)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	err = fsl.Matches(machine.Hardware)
	if err != nil {
		r.sendError(request, response, httperrors.UnprocessableEntity(err))
		return
	}

	r.send(request, response, http.StatusOK, v1.NewFilesystemLayoutResponse(fsl))
}
