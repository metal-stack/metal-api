package service

import (
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	v1 "git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/service/v1"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/utils"

	"git.f-i-ts.de/cloud-native/metallib/httperrors"
	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
)

type projectResource struct {
	webResource
}

// NewProject returns a webservice for project specific endpoints.
func NewProject(ds *datastore.RethinkStore) *restful.WebService {
	r := projectResource{
		webResource: webResource{
			ds: ds,
		},
	}
	return r.webService()
}

func (r projectResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path(BasePath + "v1/project").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"project"}

	ws.Route(ws.GET("/{id}").
		To(viewer(r.findProject)).
		Operation("findProject").
		Doc("get project by id").
		Param(ws.PathParameter("id", "identifier of the project").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.ProjectResponse{}).
		Returns(http.StatusOK, "OK", v1.ProjectResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/").
		To(viewer(r.listProjects)).
		Operation("listProjects").
		Doc("get all projects").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]v1.ProjectResponse{}).
		Returns(http.StatusOK, "OK", []v1.ProjectResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/find").
		To(viewer(r.findProjects)).
		Operation("findProjects").
		Doc("get all projects that match given properties").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.ProjectFindRequest{}).
		Writes([]v1.ProjectResponse{}).
		Returns(http.StatusOK, "OK", []v1.ProjectResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.DELETE("/{id}").
		To(editor(r.deleteProject)).
		Operation("deleteProject").
		Doc("deletes a project and returns the deleted entity").
		Param(ws.PathParameter("id", "identifier of the project").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.ProjectResponse{}).
		Returns(http.StatusOK, "OK", v1.ProjectResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.PUT("/").
		To(editor(r.createProject)).
		Operation("createProject").
		Doc("create a project. if the given ID already exists a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.ProjectCreateRequest{}).
		Returns(http.StatusCreated, "Created", v1.ProjectResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/").
		To(editor(r.updateProject)).
		Operation("updateProject").
		Doc("updates a project. if the project was changed since this one was read, a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.ProjectUpdateRequest{}).
		Returns(http.StatusOK, "OK", v1.ProjectResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r projectResource) findProject(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	p, err := r.ds.FindProjectByID(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, makeProjectResponse(p, r.ds, utils.Logger(request).Sugar()))
}

func (r projectResource) listProjects(request *restful.Request, response *restful.Response) {
	ps, err := r.ds.ListProjects()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, makeProjectResponseList(ps, r.ds, utils.Logger(request).Sugar()))
}

func (r projectResource) findProjects(request *restful.Request, response *restful.Response) {
	var requestPayload datastore.ProjectSearchQuery
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	ps := metal.Projects{}
	err = r.ds.SearchProjects(&requestPayload, &ps)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, makeProjectResponseList(ps, r.ds, utils.Logger(request).Sugar()))
}

func (r projectResource) createProject(request *restful.Request, response *restful.Response) {
	var requestPayload v1.ProjectCreateRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
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

	if name == "" {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("name should not be empty")) {
			return
		}
	}
	if requestPayload.Tenant == "" {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("tenant should not be empty")) {
			return
		}
	}

	p := &metal.Project{
		Base: metal.Base{
			Name:        name,
			Description: description,
		},
		Tenant: requestPayload.Tenant,
	}

	err = r.ds.CreateProject(p)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusCreated, makeProjectResponse(p, r.ds, utils.Logger(request).Sugar()))
}

func (r projectResource) deleteProject(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	p, err := r.ds.FindProjectByID(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	ms, err := r.ds.ListMachines()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	_, ok := ms.ByProjectID()[p.ID]
	if ok {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("project still has machines contained")) {
			return
		}
	}

	ips, err := r.ds.ListIPs()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	_, ok = ips.ByProjectID()[p.ID]
	if ok {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("project still has ips assigned")) {
			return
		}
	}

	err = r.ds.DeleteProject(p)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, makeProjectResponse(p, r.ds, utils.Logger(request).Sugar()))
}

func (r projectResource) updateProject(request *restful.Request, response *restful.Response) {
	var requestPayload v1.ProjectUpdateRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	oldProject, err := r.ds.FindProjectByID(requestPayload.ID)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	newProject := *oldProject

	if requestPayload.Name != nil {
		newProject.Name = *requestPayload.Name
	}
	if requestPayload.Description != nil {
		newProject.Description = *requestPayload.Description
	}

	err = r.ds.UpdateProject(oldProject, &newProject)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, makeProjectResponse(&newProject, r.ds, utils.Logger(request).Sugar()))
}

func makeProjectResponse(p *metal.Project, ds *datastore.RethinkStore, logger *zap.SugaredLogger) *v1.ProjectResponse {
	return v1.NewProjectResponse(p)
}

func makeProjectResponseList(ps metal.Projects, ds *datastore.RethinkStore, logger *zap.SugaredLogger) []*v1.ProjectResponse {
	result := []*v1.ProjectResponse{}

	for index := range ps {
		result = append(result, v1.NewProjectResponse(&ps[index]))
	}

	return result
}
