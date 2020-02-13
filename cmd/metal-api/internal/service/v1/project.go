package v1

import (
	mdmv1 "git.f-i-ts.de/cloud-native/masterdata-api/api/v1"
)

type ProjectResponse struct {
	mdmv1.Project
}

type ProjectFindRequest struct {
	mdmv1.ProjectFindRequest
}
