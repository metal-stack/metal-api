package utils

import (
	"strings"

	restful "github.com/emicklei/go-restful"
	"github.com/inconshreveable/log15"
)

func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func RestfulLogger(logger log15.Logger) restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		chain.ProcessFilter(req, resp)
		logger.Info("Rest Call",
			"remoteaddr", strings.Split(req.Request.RemoteAddr, ":")[0],
			"method", req.Request.Method,
			"uri", req.Request.URL.RequestURI(),
			"protocol", req.Request.Proto,
			"status", resp.StatusCode(),
			"content-length", resp.ContentLength())
	}
}
