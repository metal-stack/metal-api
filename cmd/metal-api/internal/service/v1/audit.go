package v1

import (
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

	User   string `json:"user" optional:"true"`
	Tenant string `json:"tenant" optional:"true"`

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
	RequestID string    `json:"rqid" optional:"true"`
	Phase     string    `json:"phase" optional:"true"`
	Timestamp time.Time `json:"timestamp" optional:"true"`

	User   string `json:"user" optional:"true"`
	Tenant string `json:"tenant" optional:"true"`

	Path string `json:"path" optional:"true"`

	ForwardedFor string `json:"forwarded_for" optional:"true"`
	RemoteAddr   string `json:"remote_address" optional:"true"`

	Body       string `json:"body" optional:"true"`
	StatusCode int    `json:"code" optional:"true"`
}

func NewAuditResponse(e auditing.Entry) *AuditResponse {
	body := ""
	switch v := e.Body.(type) {
	case string:
		body = v
	case []byte:
		body = string(v)
	}

	return &AuditResponse{
		RequestID:    e.RequestId,
		Phase:        string(e.Phase),
		Timestamp:    e.Timestamp,
		User:         e.User,
		Tenant:       e.Tenant,
		Path:         e.Path,
		ForwardedFor: e.ForwardedFor,
		RemoteAddr:   e.RemoteAddr,
		Body:         body,
		StatusCode:   e.StatusCode,
	}
}
