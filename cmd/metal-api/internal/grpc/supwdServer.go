package grpc

import (
	"context"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"io/ioutil"
	"strings"
)

func (s *Server) FetchSupermetalPassword(ctx context.Context, req *v1.SupermetalPasswordRequest) (*v1.SupermetalPasswordResponse, error) {
	defer ctx.Done()

	resp := &v1.SupermetalPasswordResponse{
		FeatureDisabled: false,
	}
	bb, err := ioutil.ReadFile("/bmc/supermetal.pwd")
	if err != nil {
		resp.FeatureDisabled = true // having no supermetal password in place is not an error but indicates that we disable updating bmc admin user
		return resp, nil
	}
	resp.SupermetalPassword = strings.TrimSpace(string(bb))
	return resp, nil
}
