package utils

import (
	"net/http/httputil"
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

func RestfulLogger(logger log15.Logger, debug bool) restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		info := []interface{}{
			"remoteaddr", strings.Split(req.Request.RemoteAddr, ":")[0],
			"method", req.Request.Method,
			"uri", req.Request.URL.RequestURI(),
			"protocol", req.Request.Proto,
		}

		if debug {
			body, _ := httputil.DumpRequest(req.Request, true)
			info = append(info, "body")
			info = append(info, string(body))
		}

		chain.ProcessFilter(req, resp)

		info = append(info, "status")
		info = append(info, resp.StatusCode())
		info = append(info, "content-length")
		info = append(info, resp.ContentLength())

		if resp.StatusCode() < 400 {
			logger.Info("Rest Call", info...)
		} else {
			logger.Error("Rest Call", info...)
		}
	}
}
