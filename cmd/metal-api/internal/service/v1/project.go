package v1

import (
	"github.com/golang/protobuf/ptypes/timestamp"
	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
)

var p mdmv1.Project

type Timestamp struct {
	Nanos   int32 `json:"nanos,omitempty"`
	Seconds int64 `json:"seconds,omitempty"`
}

type Int32Value struct {
	Value int32 `json:"value,omitempty"`
}
type Quota struct {
	Quota *Int32Value `json:"quota,omitempty"`
}

type QuotaSet struct {
	Cluster *Quota `json:"cluster,omitempty"`
	Machine *Quota `json:"machine,omitempty"`
	IP      *Quota `json:"ip,omitempty"`
	Project *Quota `json:"project,omitempty"`
}
type Meta struct {
	ID          string            `json:"id,omitempty"`
	Kind        string            `json:"kind,omitempty"`
	Apiversion  string            `json:"apiversion,omitempty"`
	Version     int64             `json:"version,omitempty"`
	CreatedTime *Timestamp        `json:"created_time,omitempty"`
	UpdatedTime *Timestamp        `json:"updated_time,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Labels      []string          `json:"labels,omitempty"`
}

type ProjectResponse struct {
	Meta        *Meta     `json:"meta,omitempty"`
	Name        string    `json:"name,omitempty"`
	Description string    `json:"description,omitempty"`
	TenantID    string    `json:"tenant_id,omitempty"`
	Quotas      *QuotaSet `json:"quotas,omitempty"`
}

type ProjectFindRequest struct {
	ID          *string `json:"id,omitempty"`
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	TenantID    *string `json:"tenant_id,omitempty"`
}

func FromProjects(ps []*mdmv1.Project) []*ProjectResponse {
	var prs []*ProjectResponse
	for _, p := range ps {
		prs = append(prs, FromProject(p))
	}
	return prs
}

func FromProject(p *mdmv1.Project) *ProjectResponse {
	return &ProjectResponse{
		Meta:        fromMeta(p.Meta),
		Name:        p.Name,
		Description: p.Description,
		TenantID:    p.TenantId,
		Quotas:      fromQuotaSet(p.Quotas),
	}
}
func fromQuota(q *mdmv1.Quota) *Quota {
	if q == nil {
		return nil
	}
	if q.Quota == nil {
		return &Quota{}
	}
	return &Quota{
		Quota: &Int32Value{
			Value: q.Quota.Value,
		},
	}
}

func fromQuotaSet(q *mdmv1.QuotaSet) *QuotaSet {
	if q == nil {
		return nil
	}
	return &QuotaSet{
		Cluster: fromQuota(q.Cluster),
		IP:      fromQuota(q.Ip),
		Machine: fromQuota(q.Machine),
		Project: fromQuota(q.Project),
	}
}

func fromMeta(m *mdmv1.Meta) *Meta {
	if m == nil {
		return nil
	}
	return &Meta{
		ID:          p.Meta.Id,
		Kind:        p.Meta.Kind,
		Apiversion:  p.Meta.Apiversion,
		Version:     p.Meta.Version,
		Annotations: p.Meta.Annotations,
		Labels:      p.Meta.Labels,
		CreatedTime: fromTimestamp(p.Meta.CreatedTime),
		UpdatedTime: fromTimestamp(p.Meta.UpdatedTime),
	}
}

func fromTimestamp(t *timestamp.Timestamp) *Timestamp {
	if t == nil {
		return nil
	}
	return &Timestamp{
		Nanos:   t.Nanos,
		Seconds: t.Seconds,
	}
}
