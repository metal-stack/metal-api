package service

import (
	"net/http"
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/auditing"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	libauditing "github.com/metal-stack/metal-lib/auditing"
	"github.com/metal-stack/security"
	"go.uber.org/zap"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-lib/httperrors"
)

type auditResource struct {
	webResource
	a libauditing.Auditing
}

func NewAudit(log *zap.SugaredLogger, a libauditing.Auditing) *restful.WebService {
	ir := auditResource{
		webResource: webResource{
			log: log,
		},
		a: a,
	}

	return ir.webService()
}

func (r auditResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path(BasePath + "v1/audit").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"audit"}

	ws.Route(ws.POST("/find").
		To(viewer(r.find)).
		Operation("findAuditTraces").
		Doc("find all audit traces that match given properties").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Metadata(auditing.Exclude, true).
		Reads(v1.AuditFindRequest{}).
		Writes([]v1.AuditResponse{}).
		Returns(http.StatusOK, "OK", []v1.AuditResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r *auditResource) find(request *restful.Request, response *restful.Response) {
	// TODO: enable as soon as real backend can be enabled
	// if r.a == nil {
	// 	r.sendError(request, response, httperrors.InternalServerError(featureDisabledErr))
	// 	return
	// }

	var requestPayload v1.AuditFindRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	// FIXME: just dummy response until backend is implemented
	usr := security.GetUser(request.Request)
	backendResult := []*libauditing.Entry{
		{
			Id:           "db8e312b-491c-4282-b4cb-f95f77db49ef",
			Component:    "metal-api",
			RequestId:    "1410672d-ba63-4286-9b8f-3b9c61b4e564",
			Type:         libauditing.EntryTypeHTTP,
			Timestamp:    time.Now(),
			User:         usr.EMail,
			Tenant:       tenant(request),
			Phase:        libauditing.EntryPhaseResponse,
			Path:         BasePath + "v1/audit/find",
			ForwardedFor: "192.168.2.1",
			RemoteAddr:   "192.168.2.2",
			Body:         `{"changes": "foo"}`,
			StatusCode:   http.StatusOK,
		},
	}

	result := []*v1.AuditResponse{}
	for _, e := range backendResult {
		e := e
		result = append(result, v1.NewAuditResponse(e))
	}

	r.send(request, response, http.StatusOK, result)
}
