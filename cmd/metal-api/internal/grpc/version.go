package grpc

import (
	"context"

	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"github.com/metal-stack/v"
)

type versionService struct {
}

func NewVersionService() *versionService {
	return &versionService{}
}
func (vs *versionService) Get(ctx context.Context, request *v1.GetVersionRequest) (*v1.GetVersionResponse, error) {
	res := v1.GetVersionResponse{Version: v.Version, Revision: v.Revision, BuildDate: v.BuildDate, GitSha1: v.GitSHA1}
	return &res, nil
}
