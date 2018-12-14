package utils

import (
	"bytes"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"go.uber.org/zap"

	restful "github.com/emicklei/go-restful"
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

// RestfulLogger is a middleware around every rest call and logs some information
// abount the request. If the 'debug' paramter is true, the body of the request
// and the body of the response will also be logged.
func RestfulLogger(logger *zap.Logger, debug bool) restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		fields := []zap.Field{
			zap.String("remoteaddr", strings.Split(req.Request.RemoteAddr, ":")[0]),
			zap.String("method", req.Request.Method),
			zap.String("uri", req.Request.URL.RequestURI()),
			zap.String("protocol", req.Request.Proto),
			zap.String("route", req.SelectedRoutePath()),
		}

		if debug {
			body, _ := httputil.DumpRequest(req.Request, true)
			fields = append(fields, zap.String("body", string(body)))
			resp.ResponseWriter = &loggingResponseWriter{w: resp.ResponseWriter}
		}

		t := time.Now()
		chain.ProcessFilter(req, resp)

		fields = append(fields, zap.Int("status", resp.StatusCode()), zap.Int("content-length", resp.ContentLength()), zap.Duration("duration", time.Since(t)))

		if debug {
			fields = append(fields, zap.String("response", resp.ResponseWriter.(*loggingResponseWriter).Content()))
		}
		if resp.StatusCode() < 400 {
			logger.Info("Rest Call", fields...)
		} else {
			logger.Error("Rest Call", fields...)
		}
	}
}
