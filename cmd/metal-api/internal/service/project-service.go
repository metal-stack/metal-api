package service

import (
	"context"
	"net/http"

	"github.com/metal-stack/masterdata-api/api/rest/mapper"
	v1 "github.com/metal-stack/masterdata-api/api/rest/v1"
	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	mdm "github.com/metal-stack/masterdata-api/pkg/client"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/utils"
	"go.uber.org/zap"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/metal-lib/zapup"
)

type projectResource struct {
	webResource
	mdc mdm.Client
}

// NewProject returns a webservice for project specific endpoints.
func NewProject(ds *datastore.RethinkStore, mdc mdm.Client) *restful.WebService {
	r := projectResource{
		webResource: webResource{
			ds: ds,
		},
		mdc: mdc,
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

	return ws
}

func (r projectResource) findProject(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	p, err := r.mdc.Project().Get(context.Background(), &mdmv1.ProjectGetRequest{Id: id})
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	v1p := mapper.ToV1Project(p.Project)

	// TODO: Retrieve current counts from backend
	// if v1p.Quotas == nil {
	// 	v1p.Quotas = &v1.QuotaSet{}
	// }
	// qs := v1p.Quotas
	// if qs.Cluster == nil {
	// 	qs.Cluster = &v1.Quota{}
	// }
	// if qs.Machine == nil {
	// 	qs.Machine = &v1.Quota{}
	// }
	// if qs.Ip == nil {
	// 	qs.Ip = &v1.Quota{}
	// }
	// qs.Machine.Used = utils.Int32Ptr(int32(machineUsage))
	// qs.Ip.Used = utils.Int32Ptr(int32(ipUsage))
	// qs.Project.Used

	err = response.WriteHeaderAndEntity(http.StatusOK, v1p)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r projectResource) listProjects(request *restful.Request, response *restful.Response) {
	res, err := r.mdc.Project().Find(context.Background(), &mdmv1.ProjectFindRequest{})
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	var ps []*v1.Project
	for _, p := range res.Projects {
		v1p := mapper.ToV1Project(p)

		// TODO: Retrieve current counts from backend
		// if v1p.Quotas == nil {
		// 	v1p.Quotas = &v1.QuotaSet{}
		// }
		// qs := v1p.Quotas
		// if qs.Cluster == nil {
		// 	qs.Cluster = &v1.Quota{}
		// }
		// if qs.Machine == nil {
		// 	qs.Machine = &v1.Quota{}
		// }
		// if qs.Ip == nil {
		// 	qs.Ip = &v1.Quota{}
		// }
		// qs.Machine.Used = utils.Int32Ptr(int32(machineUsage))
		// qs.Ip.Used = utils.Int32Ptr(int32(ipUsage))
		// qs.Project.Used

		ps = append(ps, v1p)
	}

	projectResponse := &v1.ProjectListResponse{
		Projects: ps,
	}

	err = response.WriteHeaderAndEntity(http.StatusOK, projectResponse)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r projectResource) findProjects(request *restful.Request, response *restful.Response) {
	var requestPayload v1.ProjectFindRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	res, err := r.mdc.Project().Find(context.Background(), mapper.ToMdmV1ProjectFindRequest(&requestPayload))
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	var ps []*v1.Project
	for _, p := range res.Projects {
		v1p := mapper.ToV1Project(p)

		// TODO: Retrieve current counts from backend
		// if v1p.Quotas == nil {
		// 	v1p.Quotas = &v1.QuotaSet{}
		// }
		// qs := v1p.Quotas
		// if qs.Cluster == nil {
		// 	qs.Cluster = &v1.Quota{}
		// }
		// if qs.Machine == nil {
		// 	qs.Machine = &v1.Quota{}
		// }
		// if qs.Ip == nil {
		// 	qs.Ip = &v1.Quota{}
		// }
		// qs.Machine.Used = utils.Int32Ptr(int32(machineUsage))
		// qs.Ip.Used = utils.Int32Ptr(int32(ipUsage))
		// qs.Project.Used

		ps = append(ps, v1p)
	}

	projectResponse := &v1.ProjectListResponse{
		Projects: ps,
	}

	err = response.WriteHeaderAndEntity(http.StatusOK, projectResponse)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}
