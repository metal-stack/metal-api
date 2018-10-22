package utils

import (
	"bytes"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	restful "github.com/emicklei/go-restful"
	"github.com/inconshreveable/log15"
)

type loggingResponseWriter struct {
	w      http.ResponseWriter
	buf    bytes.Buffer
	header int
}

func (w *loggingResponseWriter) Header() http.Header {
	return w.w.Header()
}

func (w *loggingResponseWriter) Write(b []byte) (int, error) {
	(&w.buf).Write(b)
	return w.w.Write(b)
}

func (w *loggingResponseWriter) WriteHeader(h int) {
	w.header = h
	w.w.WriteHeader(h)
}

func (w *loggingResponseWriter) Content() string {
	return w.buf.String()
}

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
			resp.ResponseWriter = &loggingResponseWriter{w: resp.ResponseWriter}
		}

		t := time.Now()
		chain.ProcessFilter(req, resp)

		info = append(info, "status", resp.StatusCode(), "content-length", resp.ContentLength(), "duration", time.Since(t))
		if debug {
			info = append(info, "response", resp.ResponseWriter.(*loggingResponseWriter).Content())
		}
		if resp.StatusCode() < 400 {
			logger.Info("Rest Call", info...)
		} else {
			logger.Error("Rest Call", info...)
		}
	}
}
