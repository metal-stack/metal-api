package utils

import (
	"net/http/httputil"
	"strings"
	"time"

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
			"route", req.SelectedRoutePath(),
		}

		if debug {
			body, _ := httputil.DumpRequest(req.Request, true)
			info = append(info, "body", string(body))
		}
		t := time.Now()
		chain.ProcessFilter(req, resp)
		info = append(info, "status", resp.StatusCode(), "content-length", resp.ContentLength(), "duration", time.Since(t))
		logger.Info("Rest Call", info...)
	}
}
