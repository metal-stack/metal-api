package utils

import (
	"bytes"
	"net/http"
	"net/http/httputil"
	"runtime"
	"strings"
	"time"

	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"

	"github.com/emicklei/go-restful"
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

var (
	logkey = key(0)
)

// Logger returns the request logger from the request.
func Logger(rq *restful.Request) *zap.Logger {
	l, ok := rq.Request.Context().Value(logkey).(*zap.Logger)
	if ok {
		return l
	}
	return zapup.MustRootLogger()
}

// MetalAPI is a middleware around every rest call and logs some information
// abount the request. If the 'debug' paramter is true, the body of the request
// and the body of the response will also be logged.
func MetalAPI(debug bool) restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		log := zapup.RequestLogger(req.Request)
		rq := req.Request

		fields := []zap.Field{
			zap.String("remoteaddr", strings.Split(rq.RemoteAddr, ":")[0]),
			zap.String("method", rq.Method),
			zap.String("uri", rq.URL.RequestURI()),
			zap.String("protocol", rq.Proto),
			zap.String("route", req.SelectedRoutePath()),
		}

		if debug {
			body, _ := httputil.DumpRequest(rq, true)
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
			log.Info("Rest Call", fields...)
		} else {
			log.Error("Rest Call", fields...)
		}
	}
}

// CurrentFuncName returns the name of the caller of this function.
func CurrentFuncName() string {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return "unknown"
	}
	ffpc := runtime.FuncForPC(pc)
	if ffpc == nil {
		return "unknown"
	}
	pp := strings.Split(ffpc.Name(), ".")
	return pp[len(pp)-1]
}
