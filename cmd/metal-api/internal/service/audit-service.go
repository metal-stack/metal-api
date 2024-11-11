package service

import (
	"log/slog"
	"net/http"

	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-lib/auditing"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-lib/httperrors"
)

type auditResource struct {
	webResource
	a auditing.Auditing
}

func NewAudit(log *slog.Logger, a auditing.Auditing) *restful.WebService {
	ir := auditResource{
		webResource: webResource{
			log: log,
		},
		a: a,
	}

	return ir.webService()
}

func (r *auditResource) webService() *restful.WebService {
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
	if r.a == nil {
		r.sendError(request, response, httperrors.InternalServerError(featureDisabledErr))
		return
	}

	var requestPayload v1.AuditFindRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	backendResult, err := r.a.Search(auditing.EntryFilter{
		Limit:        requestPayload.Limit,
		From:         requestPayload.From,
		To:           requestPayload.To,
		Component:    requestPayload.Component,
		RequestId:    requestPayload.RequestId,
		Type:         auditing.EntryType(requestPayload.Type),
		User:         requestPayload.User,
		Tenant:       requestPayload.Tenant,
		Detail:       auditing.EntryDetail(requestPayload.Detail),
		Phase:        auditing.EntryPhase(requestPayload.Phase),
		Path:         requestPayload.Path,
		ForwardedFor: requestPayload.ForwardedFor,
		RemoteAddr:   requestPayload.RemoteAddr,
		Body:         requestPayload.Body,
		StatusCode:   requestPayload.StatusCode,
		Error:        requestPayload.Error,
	})
	if err != nil {
		r.sendError(request, response, httperrors.InternalServerError(err))
		return
	}

	result := []*v1.AuditResponse{}
	for _, e := range backendResult {
		e := e
		result = append(result, v1.NewAuditResponse(e))
	}

	r.send(request, response, http.StatusOK, result)
}
