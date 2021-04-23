package audit

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/security"
	"go.uber.org/zap"
)

type AuditHandler struct {
	log *zap.SugaredLogger
	out AuditOutput
	c   *handlerPolicy
}

type handlerPolicy struct {
	withHTTPResponse bool
	withHTTPRequest  bool

	auditReads  bool
	readMethods map[string]bool

	auditWrites  bool
	writeMethods map[string]bool
}

func NewAuditHandler(log *zap.SugaredLogger, out AuditOutput) (*AuditHandler, error) {
	if out == nil {
		return nil, fmt.Errorf("audit outputter must be defined")
	}

	return &AuditHandler{
		log: log,
		out: out,
		c: &handlerPolicy{
			withHTTPResponse: false,
			withHTTPRequest:  false,
			auditReads:       false,
			readMethods: map[string]bool{
				http.MethodGet:     true,
				http.MethodHead:    true,
				http.MethodOptions: true,
				http.MethodTrace:   true,
			},
			auditWrites: true,
			writeMethods: map[string]bool{
				http.MethodDelete: true,
				http.MethodPost:   true,
				http.MethodPut:    true,
				http.MethodPatch:  true,
			},
		},
	}, nil
}

func (a *AuditHandler) WithWriteActions() {
	a.c.auditWrites = true
}

func (a *AuditHandler) WithWriteMethods(writeMethods map[string]bool) {
	a.c.writeMethods = writeMethods
}

func (a *AuditHandler) WithReadActions() {
	a.c.auditReads = true
}

func (a *AuditHandler) WithReadMethods(readMethods map[string]bool) {
	a.c.readMethods = readMethods
}

func (a *AuditHandler) WithResponse() {
	a.c.withHTTPResponse = true
}

func (a *AuditHandler) WithRequest() {
	a.c.withHTTPRequest = true
}

// Audit is a go-restful request filter method that will emit audit logs to the configured channels.
//
// Ensure that this is at the beginning of the filter chain.
func (a *AuditHandler) Audit(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	chain.ProcessFilter(req, resp)

	if !a.c.auditReads && a.c.readMethods[req.Request.Method] {
		return
	}

	if !a.c.auditWrites && a.c.writeMethods[req.Request.Method] {
		return
	}

	msg := AuditMessage{
		Path:   strings.Split(strings.TrimPrefix(req.Request.URL.Path, "/"), "/"),
		Method: req.Request.Method,
		Code:   resp.StatusCode(),
	}

	if a.c.withHTTPRequest {
		body, err := httputil.DumpRequest(req.Request, true)
		if err != nil {
			a.log.Errorw("request body could not be dumped", "error", err)
		} else {
			msg.Request = body
		}
	}

	if a.c.withHTTPResponse {
		payload, err := httputil.DumpResponse(req.Request.Response, true)
		if err != nil {
			a.log.Errorw("request response could not be dumped", "error", err)
		} else {
			msg.Response = payload
		}
	}

	u, ok := req.Attribute("user").(*security.User)
	if ok {
		msg.User = u.Name
		msg.EMail = u.EMail
		msg.Tenant = u.Tenant
	}

	scope, ok := req.Attribute("scope").(datastore.ResourceScope)
	if ok {
		msg.ScopeJSON = scope.String()
	}

	msg.constructSummary()

	err := a.out.Emit(msg)
	if err != nil {
		a.log.Errorw("unable to emit audit event", "message", msg, "error", err)
	}
}

func (msg *AuditMessage) constructSummary() {
	userName := "Anonymous user"
	if msg.User != "" {
		userName = msg.User
	}
	if msg.EMail != "" {
		userName = fmt.Sprintf("%s (%s)", userName, msg.EMail)
	}

	outcome := "succeeded to"
	if msg.Code >= 500 {
		outcome = "failed to"
	} else if msg.Code >= 400 {
		outcome = "attempted to"
	}

	action := strings.ToLower(msg.Method)
	resource := strings.Join(msg.Path, "/")

	msg.Summary = fmt.Sprintf("%s %s %s %s", userName, outcome, action, resource)
}
