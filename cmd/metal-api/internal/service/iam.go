package service

import (
	"net/http"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"go.uber.org/zap"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/metal-lib/zapup"
)

type iamResource struct {
	webResource
}

// NewIAM returns a webservice for iam specific endpoints.
func NewIAM(ds *datastore.RethinkStore) *restful.WebService {
	ir := iamResource{
		webResource: webResource{
			ds: ds,
		},
	}
	return ir.webService()
}

func (ir iamResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path(BasePath + "v1/iam").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"iam"}

	ws.Route(ws.GET("/permissions").
		To(ir.listPermissions).
		Operation("listPermissions").
		Doc("get all permissions").
		Param(ws.QueryParameter("visibility", "comma-separated list of visbilities (private, shared, public, admin)").DataType("string").DefaultValue("shared")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]string{}).
		Returns(http.StatusOK, "OK", []string{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/roles").
		To(ir.listRoles).
		Operation("listRoles").
		Doc("get all roles").
		Param(ws.QueryParameter("visibility", "comma-separated list of visbilities (private, shared, public, admin)").DataType("string").DefaultValue("shared")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]string{}).
		Returns(http.StatusOK, "OK", []string{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (ir iamResource) listPermissions(request *restful.Request, response *restful.Response) {
	err := response.WriteHeaderAndEntity(http.StatusNotImplemented, "not yet implemented")
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (ir iamResource) listRoles(request *restful.Request, response *restful.Response) {
	err := response.WriteHeaderAndEntity(http.StatusNotImplemented, "not yet implemented")
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}
