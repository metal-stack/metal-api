package grpc

import (
	"context"
	"fmt"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"io/ioutil"
	"strings"
)

func (s *Server) FetchSupermetalPassword(ctx context.Context, req *v1.SupermetalPasswordRequest) (*v1.SupermetalPasswordResponse, error) {
	defer ctx.Done()

	bb, err := ioutil.ReadFile(fmt.Sprintf("/supermetal/%s.pwd", req.PartitionID))
	if err != nil {
		return nil, err
	}
	supwd := strings.TrimSpace(string(bb))
	return &v1.SupermetalPasswordResponse{
		SupermetalPassword: supwd,
	}, nil
}
