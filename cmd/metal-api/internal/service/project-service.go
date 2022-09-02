package service

import (
	"context"
	"errors"
	"net/http"

	"github.com/metal-stack/masterdata-api/api/rest/mapper"
	v1 "github.com/metal-stack/masterdata-api/api/rest/v1"
	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	mdm "github.com/metal-stack/masterdata-api/pkg/client"
	"go.uber.org/zap"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-lib/httperrors"
)

type projectResource struct {
	webResource
	mdc mdm.Client
}

// NewProject returns a webservice for project specific endpoints.
func NewProject(log *zap.SugaredLogger, ds *datastore.RethinkStore, mdc mdm.Client) *restful.WebService {
	r := projectResource{
		webResource: webResource{
			log: log,
			ds:  ds,
		},
		mdc: mdc,
	}
	return r.webService()
}

func (r *projectResource) webService() *restful.WebService {
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
		To(admin(r.deleteProject)).
		Operation("deleteProject").
		Doc("deletes a project and returns the deleted entity").
		Param(ws.PathParameter("id", "identifier of the project").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.ProjectResponse{}).
		Returns(http.StatusOK, "OK", v1.ProjectResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.PUT("/").
		To(admin(r.createProject)).
		Operation("createProject").
		Doc("create a project. if the given ID already exists a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.ProjectCreateRequest{}).
		Returns(http.StatusCreated, "Created", v1.ProjectResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/").
		To(admin(r.updateProject)).
		Operation("updateProject").
		Doc("update a project. optimistic lock error can occur.").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.ProjectUpdateRequest{}).
		Returns(http.StatusOK, "Updated", v1.ProjectResponse{}).
		Returns(http.StatusPreconditionFailed, "OptimisticLock", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r *projectResource) findProject(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	p, err := r.mdc.Project().Get(context.Background(), &mdmv1.ProjectGetRequest{Id: id})
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	v1p, err := r.setProjectQuota(p.Project)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, v1p)
}

func (r *projectResource) listProjects(request *restful.Request, response *restful.Response) {
	res, err := r.mdc.Project().Find(context.Background(), &mdmv1.ProjectFindRequest{})
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	var ps []*v1.Project
	for i := range res.Projects {
		v1p := mapper.ToV1Project(res.Projects[i])
		ps = append(ps, v1p)
	}

	r.send(request, response, http.StatusOK, ps)
}

func (r *projectResource) findProjects(request *restful.Request, response *restful.Response) {
	var requestPayload v1.ProjectFindRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	res, err := r.mdc.Project().Find(context.Background(), mapper.ToMdmV1ProjectFindRequest(&requestPayload))
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	var ps []*v1.Project
	for i := range res.Projects {
		v1p := mapper.ToV1Project(res.Projects[i])
		ps = append(ps, v1p)
	}

	r.send(request, response, http.StatusOK, ps)
}

func (r *projectResource) createProject(request *restful.Request, response *restful.Response) {
	var pcr v1.ProjectCreateRequest
	err := request.ReadEntity(&pcr)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	project := mapper.ToMdmV1Project(&pcr.Project)

	if project.TenantId == "" {
		r.sendError(request, response, httperrors.BadRequest(errors.New("no tenant given")))
		return
	}

	mdmv1pcr := &mdmv1.ProjectCreateRequest{
		Project: project,
	}

	p, err := r.mdc.Project().Create(context.Background(), mdmv1pcr)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	v1p := mapper.ToV1Project(p.Project)
	pcres := &v1.ProjectResponse{
		Project: *v1p,
	}

	r.send(request, response, http.StatusCreated, pcres)
}

func (r *projectResource) deleteProject(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	pgr := &mdmv1.ProjectGetRequest{
		Id: id,
	}
	p, err := r.mdc.Project().Get(context.Background(), pgr)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	var ms metal.Machines
	err = r.ds.SearchMachines(&datastore.MachineSearchQuery{AllocationProject: &id}, &ms)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}
	if len(ms) > 0 {
		r.sendError(request, response, httperrors.BadRequest(errors.New("there are still machines allocated by this project")))
		return
	}

	var ns metal.Networks
	err = r.ds.SearchNetworks(&datastore.NetworkSearchQuery{ProjectID: &id}, &ns)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}
	if len(ns) > 0 {
		r.sendError(request, response, httperrors.BadRequest(errors.New("there are still networks allocated by this project")))
		return
	}

	var ips metal.IPs
	err = r.ds.SearchIPs(&datastore.IPSearchQuery{ProjectID: &id}, &ips)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	if len(ips) > 0 {
		r.sendError(request, response, httperrors.BadRequest(errors.New("there are still ips allocated by this project")))
		return
	}

	pdr := &mdmv1.ProjectDeleteRequest{
		Id: p.Project.Meta.Id,
	}
	pdresponse, err := r.mdc.Project().Delete(context.Background(), pdr)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	v1p := mapper.ToV1Project(pdresponse.Project)
	pcres := &v1.ProjectResponse{
		Project: *v1p,
	}

	r.send(request, response, http.StatusOK, pcres)
}

func (r *projectResource) updateProject(request *restful.Request, response *restful.Response) {
	var requestPayload v1.ProjectUpdateRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	if requestPayload.Project.Meta == nil {
		r.sendError(request, response, httperrors.BadRequest(errors.New("project and project.meta must be specified")))
		return
	}

	existingProject, err := r.mdc.Project().Get(context.Background(), &mdmv1.ProjectGetRequest{Id: requestPayload.Project.Meta.Id})
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	// new data
	projectUpdateData := mapper.ToMdmV1Project(&requestPayload.Project)
	// created date is not updateable
	projectUpdateData.Meta.CreatedTime = existingProject.Project.Meta.CreatedTime

	pur, err := r.mdc.Project().Update(context.TODO(), &mdmv1.ProjectUpdateRequest{
		Project: projectUpdateData,
	})
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	v1p := mapper.ToV1Project(pur.Project)

	r.send(request, response, http.StatusOK, &v1.ProjectResponse{
		Project: *v1p,
	})
}

func (r *projectResource) setProjectQuota(project *mdmv1.Project) (*v1.Project, error) {
	if project.Meta == nil {
		return nil, errors.New("project does not have a projectID")
	}
	projectID := project.Meta.Id

	var ips metal.IPs
	err := r.ds.SearchIPs(&datastore.IPSearchQuery{ProjectID: &projectID}, &ips)
	if err != nil {
		return nil, err
	}

	var ms metal.Machines
	err = r.ds.SearchMachines(&datastore.MachineSearchQuery{AllocationProject: &projectID}, &ms)
	if err != nil {
		return nil, err
	}

	p := mapper.ToV1Project(project)
	if p.Quotas == nil {
		p.Quotas = &v1.QuotaSet{}
	}
	qs := p.Quotas
	if qs.Machine == nil {
		qs.Machine = &v1.Quota{}
	}
	if qs.Ip == nil {
		qs.Ip = &v1.Quota{}
	}
	machineUsage := int32(len(ms))
	ipUsage := int32(len(ips))
	qs.Machine.Used = &machineUsage
	qs.Ip.Used = &ipUsage

	return p, nil
}
