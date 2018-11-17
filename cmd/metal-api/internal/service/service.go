package service

import (
	"github.com/go-stack/stack"

	restful "github.com/emicklei/go-restful"
	"go.uber.org/zap"
)

func sendError(log *zap.Logger, rsp *restful.Response, service string, status int, err error) {
	s := stack.Caller(1)
	log.Error("service error", zap.String("service", service), zap.String("error", err.Error()), zap.Stringer("service-caller", s))
	rsp.WriteError(status, err)
}
