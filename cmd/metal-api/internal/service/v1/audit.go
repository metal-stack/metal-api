package v1

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/metal-stack/metal-lib/auditing"
)

type AuditFindRequest struct {
	Limit int64 `json:"limit" optional:"true"`

	From time.Time `json:"from" optional:"true"`
	To   time.Time `json:"to" optional:"true"`

	Component string `json:"component" optional:"true"`
	RequestId string `json:"rqid" optional:"true"`
	Type      string `json:"type" optional:"true"`

	User    string `json:"user" optional:"true"`
	Tenant  string `json:"tenant" optional:"true"`
	Project string `json:"project" optional:"true"`

	Detail string `json:"detail" optional:"true"`
	Phase  string `json:"phase" optional:"true"`

	Path         string `json:"path" optional:"true"`
	ForwardedFor string `json:"forwarded_for" optional:"true"`
	RemoteAddr   string `json:"remote_addr" optional:"true"`

	Body       string `json:"body" optional:"true"`
	StatusCode int    `json:"status_code" optional:"true"`

	Error string `json:"error" optional:"true"`
}

type AuditResponse struct {
	Component string    `json:"component" optional:"true"`
	RequestId string    `json:"rqid" optional:"true"`
	Type      string    `json:"type" optional:"true"`
	Timestamp time.Time `json:"timestamp" optional:"true"`

	User   string `json:"user" optional:"true"`
	Tenant string `json:"tenant" optional:"true"`

	// HTTP method get, post, put, delete, ...
	// or for grpc unary, stream
	Detail string `json:"detail" optional:"true"`
	// e.g. Request, Response, Error, Opened, Close
	Phase string `json:"phase" optional:"true"`
	// /api/v1/... or the method name
	Path         string `json:"path" optional:"true"`
	ForwardedFor string `json:"forwarded_for" optional:"true"`
	RemoteAddr   string `json:"remote_addr" optional:"true"`

	Body       string `json:"body" optional:"true"`
	StatusCode *int   `json:"status_code" optional:"true"`

	// Internal errors
	Error string `json:"error" optional:"true"`
}

func NewAuditResponse(e auditing.Entry) *AuditResponse {
	body := ""
	switch v := e.Body.(type) {
	case string:
		body = v
	case []byte:
		body = string(v)
	default:
		b, err := json.Marshal(v)
		if err == nil {
			body = string(b)
		} else {
			body = fmt.Sprintf("unknown body: %v", v)
		}
	}
	errStr := ""
	if e.Error != nil {
		b, err := json.Marshal(e.Error)
		if err == nil {
			errStr = string(b)
		} else {
			errStr = fmt.Sprintf("unknown error: %v", e.Error)
		}
	}

	return &AuditResponse{
		Component:    e.Component,
		RequestId:    e.RequestId,
		Type:         string(e.Type),
		Timestamp:    e.Timestamp,
		User:         e.User,
		Tenant:       e.Tenant,
		Detail:       string(e.Detail),
		Phase:        string(e.Phase),
		Path:         e.Path,
		ForwardedFor: e.ForwardedFor,
		RemoteAddr:   e.RemoteAddr,
		Body:         body,
		StatusCode:   e.StatusCode,
		Error:        errStr,
	}
}
