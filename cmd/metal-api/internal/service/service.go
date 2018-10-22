package service

import (
	restful "github.com/emicklei/go-restful"
	"github.com/inconshreveable/log15"
)

func sendError(log log15.Logger, rsp *restful.Response, service string, status int, err error) {
	log.Error("service error", "service", service, "error", err)
	rsp.WriteError(status, err)
}
