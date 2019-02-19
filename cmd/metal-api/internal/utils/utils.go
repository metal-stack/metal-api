package utils

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"git.f-i-ts.de/cloud-native/metallib/zapup"
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

type key int

var logkey key

// Logger returns the request logger from the request.
func Logger(rq *restful.Request) *zap.Logger {
	l, ok := rq.Request.Context().Value(logkey).(*zap.Logger)
	if ok {
		return l
	}
	return zapup.MustRootLogger()
}

// RestfulLogger is a middleware around every rest call and logs some information
// abount the request. If the 'debug' paramter is true, the body of the request
// and the body of the response will also be logged.
func RestfulLogger(logger *zap.Logger, debug bool) restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		// search a better way for a unique callid
		// perhaps a reverseproxy in front generates a unique header for som sort
		// of opentracing support?
		ts := time.Now().UnixNano()
		rqid := zap.Int64("rqid", ts)
		sg := logger.With(rqid)

		fields := []zap.Field{
			zap.String("remoteaddr", strings.Split(req.Request.RemoteAddr, ":")[0]),
			zap.String("method", req.Request.Method),
			zap.String("uri", req.Request.URL.RequestURI()),
			zap.String("protocol", req.Request.Proto),
			zap.String("route", req.SelectedRoutePath()),
			rqid,
		}

		if debug {
			body, _ := httputil.DumpRequest(req.Request, true)
			fields = append(fields, zap.String("body", string(body)))
			resp.ResponseWriter = &loggingResponseWriter{w: resp.ResponseWriter}
		}

		t := time.Now()
		ctx := req.Request.Context()
		ctx = context.WithValue(ctx, logkey, sg)
		rq := req.Request.WithContext(ctx)
		req.Request = rq
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
