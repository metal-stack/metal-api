package service

import (
	"context"
	"net/http"

	"github.com/metal-stack/masterdata-api/api/rest/mapper"
	v1 "github.com/metal-stack/masterdata-api/api/rest/v1"
	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	mdm "github.com/metal-stack/masterdata-api/pkg/client"
	"go.uber.org/zap"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
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

	return ws
}

func (r *tenantResource) getTenant(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	tres, err := r.mdc.Tenant().Get(context.Background(), &mdmv1.TenantGetRequest{Id: id})
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	v1t := mapper.ToV1Tenant(tres.Tenant)

	r.send(request, response, http.StatusOK, &v1t)
}

func (r *tenantResource) listTenants(request *restful.Request, response *restful.Response) {
	tres, err := r.mdc.Tenant().Find(context.Background(), &mdmv1.TenantFindRequest{})
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
