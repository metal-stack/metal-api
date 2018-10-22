package service

import (
	"github.com/go-stack/stack"
)
import (
	restful "github.com/emicklei/go-restful"
	"github.com/inconshreveable/log15"
)

func sendError(log log15.Logger, rsp *restful.Response, service string, status int, err error) {
	s := stack.Caller(1)
	log.Error("service error", "service", service, "error", err, "service-caller", s)
	rsp.WriteError(status, err)
}
