package v1

import (
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"

)

type ProjectBase struct {
	Tenant string `json:"tenant" description:"the tenant of this project."`
}

type ProjectCreateRequest struct {
	Common
	ProjectBase
}

type ProjectFindRequest struct {
	datastore.ProjectSearchQuery
}

type ProjectUpdateRequest struct {
	Common
	ProjectBase
}

type ProjectResponse struct {
	Common
	ProjectBase
	Timestamps
}

func NewProjectResponse(p *metal.Project) *ProjectResponse {
	if p == nil {
		return nil
	}

	return &ProjectResponse{
		Common: Common{
			Identifiable: Identifiable{
				ID: p.ID,
			},
			Describable: Describable{
				Name:        &p.Name,
				Description: &p.Description,
			},
		},
		ProjectBase: ProjectBase{
			Tenant: p.Tenant,
		},
		Timestamps: Timestamps{
			Created: p.Created,
			Changed: p.Changed,
		},
	}
}
