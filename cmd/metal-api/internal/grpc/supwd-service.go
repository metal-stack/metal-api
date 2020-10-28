package grpc

import (
	"context"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"go.uber.org/zap"
	"io/ioutil"
	"strings"
)

type SupwdService struct {
	logger  *zap.SugaredLogger
	pwdFile string
}

func NewSupwdService(bmcSuperuserPasswordFile string, logger *zap.SugaredLogger) *SupwdService {
	return &SupwdService{
		logger:  logger,
		pwdFile: bmcSuperuserPasswordFile,
	}
}

func (s *SupwdService) FetchSuperUserPassword(ctx context.Context, req *v1.SuperUserPasswordRequest) (*v1.SuperUserPasswordResponse, error) {
	defer ctx.Done()

	resp := &v1.SuperUserPasswordResponse{}
	if s.pwdFile == "" {
		resp.FeatureDisabled = true
		return resp, nil
	}

	bb, err := ioutil.ReadFile(s.pwdFile)
	if err != nil {
		s.logger.Errorw("failed to lookup BMC superuser password", "password file", s.pwdFile, "error", err)
		return nil, err
	}
	resp.FeatureDisabled = false
	resp.SuperUserPassword = strings.TrimSpace(string(bb))
	return resp, nil
}
