package service

import (
	"net/http"

	"git.f-i-ts.de/cloud-native/metal/metal-api/metal"
	"github.com/go-stack/stack"

	restful "github.com/emicklei/go-restful"
	"go.uber.org/zap"
)

func sendError(log *zap.Logger, rsp *restful.Response, service string, status int, err error) {
	s := stack.Caller(1)
	log.Error("service error", zap.String("service", service), zap.String("error", err.Error()), zap.Stringer("service-caller", s))
	rsp.WriteError(status, err)
}

func checkError(log *zap.Logger, rsp *restful.Response, service string, err error) bool {
	if err != nil {
		if metal.IsNotFound(err) {
			sendError(log, rsp, service, http.StatusNotFound, err)
			return true
		}
		sendError(log, rsp, service, http.StatusInternalServerError, err)
		return true
	}
	return false
}
