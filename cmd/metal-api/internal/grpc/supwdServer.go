package grpc

import (
	"context"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"io/ioutil"
	"strings"
)

func (s *Server) FetchSuperUserPassword(ctx context.Context, req *v1.SuperUserPasswordRequest) (*v1.SuperUserPasswordResponse, error) {
	defer ctx.Done()

	resp := &v1.SuperUserPasswordResponse{
		FeatureDisabled: false,
	}
	bb, err := ioutil.ReadFile("/bmc/superUser.pwd")
	if err != nil {
		resp.FeatureDisabled = true // having no superUser password in place is not an error but indicates that we disable updating bmc admin user
		return resp, nil
	}
	resp.SuperUserPassword = strings.TrimSpace(string(bb))
	return resp, nil
}
