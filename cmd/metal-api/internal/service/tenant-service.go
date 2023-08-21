package service

import (
	"errors"
	"net/http"

	"github.com/metal-stack/masterdata-api/api/rest/mapper"
	v1 "github.com/metal-stack/masterdata-api/api/rest/v1"
	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	mdm "github.com/metal-stack/masterdata-api/pkg/client"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/wrapperspb"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-lib/auditing"
	"github.com/metal-stack/metal-lib/httperrors"
)

type tenantResource struct {
	webResource
	mdc mdm.Client
}

// NewTenant returns a webservice for tenant specific endpoints.
func NewTenant(log *zap.SugaredLogger, mdc mdm.Client) *restful.WebService {
	r := tenantResource{
		webResource: webResource{
			log: log,
		},
		mdc: mdc,
	}
	return r.webService()
}

func (r *tenantResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path(BasePath + "v1/tenant").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"tenant"}

	ws.Route(ws.GET("/{id}").
		To(viewer(r.getTenant)).
		Operation("getTenant").
		Doc("get tenant by id").
		Param(ws.PathParameter("id", "identifier of the tenant").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.TenantResponse{}).
		Returns(http.StatusOK, "OK", v1.TenantResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/").
		To(viewer(r.listTenants)).
		Operation("listTenants").
		Doc("get all tenants").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]v1.TenantResponse{}).
		Returns(http.StatusOK, "OK", []v1.TenantResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/find").
		To(viewer(r.findTenants)).
		Operation("findTenants").
		Doc("get all tenants that match given properties").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Metadata(auditing.Exclude, true).
		Reads(v1.TenantFindRequest{}).
		Writes([]v1.TenantResponse{}).
		Returns(http.StatusOK, "OK", []v1.TenantResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.DELETE("/{id}").
		To(admin(r.deleteTenant)).
		Operation("deleteTenant").
		Doc("deletes a tenant and returns the deleted entity").
		Param(ws.PathParameter("id", "identifier of the tenant").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.TenantResponse{}).
		Returns(http.StatusOK, "OK", v1.TenantResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.PUT("/").
		To(admin(r.createTenant)).
		Operation("createTenant").
		Doc("create a tenant. if the given ID already exists a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.TenantCreateRequest{}).
		Returns(http.StatusCreated, "Created", v1.TenantResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/").
		To(admin(r.updateTenant)).
		Operation("updateTenant").
		Doc("update a tenant. optimistic lock error can occur.").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.TenantUpdateRequest{}).
		Returns(http.StatusOK, "Updated", v1.TenantResponse{}).
		Returns(http.StatusPreconditionFailed, "OptimisticLock", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r *tenantResource) getTenant(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	tres, err := r.mdc.Tenant().Get(request.Request.Context(), &mdmv1.TenantGetRequest{Id: id})
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	v1t := mapper.ToV1Tenant(tres.Tenant)

	r.send(request, response, http.StatusOK, &v1t)
}

func (r *tenantResource) listTenants(request *restful.Request, response *restful.Response) {
	tres, err := r.mdc.Tenant().Find(request.Request.Context(), &mdmv1.TenantFindRequest{})
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	var v1ts []*v1.Tenant
	for _, t := range tres.Tenants {
		v1t := mapper.ToV1Tenant(t)
		v1ts = append(v1ts, v1t)
	}

	r.send(request, response, http.StatusOK, v1ts)
}

func (r *tenantResource) findTenants(request *restful.Request, response *restful.Response) {
	var requestPayload v1.TenantFindRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	res, err := r.mdc.Tenant().Find(request.Request.Context(), mapper.ToMdmV1TenantFindRequest(&requestPayload))
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	var ps []*v1.Tenant
	for i := range res.Tenants {
		v1p := mapper.ToV1Tenant(res.Tenants[i])
		ps = append(ps, v1p)
	}

	r.send(request, response, http.StatusOK, ps)
}

func (r *tenantResource) createTenant(request *restful.Request, response *restful.Response) {
	var pcr v1.TenantCreateRequest
	err := request.ReadEntity(&pcr)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	tenant := mapper.ToMdmV1Tenant(&pcr.Tenant)

	mdmv1pcr := &mdmv1.TenantCreateRequest{
		Tenant: tenant,
	}

	p, err := r.mdc.Tenant().Create(request.Request.Context(), mdmv1pcr)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	v1p := mapper.ToV1Tenant(p.Tenant)
	pcres := &v1.TenantResponse{
		Tenant: *v1p,
	}

	r.send(request, response, http.StatusCreated, pcres)
}

func (r *tenantResource) deleteTenant(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	pgr := &mdmv1.TenantGetRequest{
		Id: id,
	}
	p, err := r.mdc.Tenant().Get(request.Request.Context(), pgr)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	plr, err := r.mdc.Project().Find(request.Request.Context(), &mdmv1.ProjectFindRequest{TenantId: wrapperspb.String(id)})
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}
	if len(plr.Projects) > 0 {
		r.sendError(request, response, httperrors.UnprocessableEntity(errors.New("there are still projects allocated by this tenant")))
		return
	}

	pdr := &mdmv1.TenantDeleteRequest{
		Id: p.Tenant.Meta.Id,
	}
	pdresponse, err := r.mdc.Tenant().Delete(request.Request.Context(), pdr)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	v1p := mapper.ToV1Tenant(pdresponse.Tenant)
	pcres := &v1.TenantResponse{
		Tenant: *v1p,
	}

	r.send(request, response, http.StatusOK, pcres)
}

func (r *tenantResource) updateTenant(request *restful.Request, response *restful.Response) {
	var requestPayload v1.TenantUpdateRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	if requestPayload.Tenant.Meta == nil {
		r.sendError(request, response, httperrors.BadRequest(errors.New("tenant and tenant.meta must be specified")))
		return
	}

	existingTenant, err := r.mdc.Tenant().Get(request.Request.Context(), &mdmv1.TenantGetRequest{Id: requestPayload.Tenant.Meta.Id})
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	// new data
	tenantUpdateData := mapper.ToMdmV1Tenant(&requestPayload.Tenant)
	// created date is not updateable
	tenantUpdateData.Meta.CreatedTime = existingTenant.Tenant.Meta.CreatedTime

	pur, err := r.mdc.Tenant().Update(request.Request.Context(), &mdmv1.TenantUpdateRequest{
		Tenant: tenantUpdateData,
	})
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	v1p := mapper.ToV1Tenant(pur.Tenant)

	r.send(request, response, http.StatusOK, &v1.TenantResponse{
		Tenant: *v1p,
	})
}
