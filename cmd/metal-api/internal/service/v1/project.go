package v1

import (
	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
)

type ProjectResponse struct {
	mdmv1.Project
}

type ProjectFindRequest struct {
	mdmv1.ProjectFindRequest
}
