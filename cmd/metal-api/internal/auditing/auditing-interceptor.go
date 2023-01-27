package auditing

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/google/uuid"
	"github.com/metal-stack/metal-lib/rest"
	"github.com/metal-stack/security"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	Exclude string = "exclude-from-auditing"
)

func UnaryServerInterceptor(a Auditing, logger *zap.SugaredLogger, shouldAudit func(fullMethod string) bool) grpc.UnaryServerInterceptor {
	if a == nil {
		logger.Fatal("cannot use nil auditing to create unary server interceptor")
	}
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		if !shouldAudit(info.FullMethod) {
			return handler(ctx, req)
		}
		requestID := uuid.New().String()
		childCtx := context.WithValue(ctx, rest.RequestIDKey, requestID)

		auditReqContext := []any{
			"rqid", requestID,
			"method", info.FullMethod,
			"kind", "grpc",
		}
		user := security.GetUserFromContext(ctx)
		if user != nil {
			auditReqContext = append(
				auditReqContext,
				"user", user.EMail,
				"tenant", user.Tenant,
			)
		}
		err = a.Index(auditReqContext...)
		if err != nil {
			return nil, err
		}
		resp, err = handler(childCtx, req)
		if err != nil {
			auditRespContext := append(auditReqContext, "err", err)
			err2 := a.Index(auditRespContext...)
			if err2 != nil {
				logger.Errorf("unable to index error: %v", err2)
			}
			return nil, err
		}
		auditRespContext := append(auditReqContext, "resp", resp)
		err = a.Index(auditRespContext...)
		return resp, err
	}
}

func StreamServerInterceptor(a Auditing, logger *zap.SugaredLogger, shouldAudit func(fullMethod string) bool) grpc.StreamServerInterceptor {
	if a == nil {
		logger.Fatal("cannot use nil auditing to create stream server interceptor")
	}
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !shouldAudit(info.FullMethod) {
			return handler(srv, ss)
		}
		requestID := uuid.New().String()
		auditReqContext := []any{
			"rqid", requestID,
			"method", info.FullMethod,
			"kind", "grpc-stream",
		}

		user := security.GetUserFromContext(ss.Context())
		if user != nil {
			auditReqContext = append(
				auditReqContext,
				"user", user.EMail,
				"tenant", user.Tenant,
			)
		}
		err := a.Index(auditReqContext...)
		if err != nil {
			return err
		}
		err = handler(srv, ss)
		if err != nil {
			auditRespContext := append(auditReqContext, "err", err)
			err2 := a.Index(auditRespContext...)
			if err2 != nil {
				logger.Errorf("unable to index error: %v", err2)
			}
			return err
		}
		auditRespContext := append(auditReqContext, "finished", true)
		err = a.Index(auditRespContext...)
		return err
	}
}

func HttpFilter(a Auditing, logger *zap.SugaredLogger) restful.FilterFunction {
	if a == nil {
		logger.Fatal("cannot use nil auditing to create http middleware")
	}
	return func(request *restful.Request, response *restful.Response, chain *restful.FilterChain) {
		r := request.Request

		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			break
		default:
			chain.ProcessFilter(request, response)
			return
		}

		excluded, ok := request.SelectedRoute().Metadata()[Exclude].(bool)
		if ok && excluded {
			logger.Debugw("excluded route from auditing through metadata annotation", "path", request.SelectedRoute().Path())
			chain.ProcessFilter(request, response)
			return
		}

		requestID := r.Context().Value(rest.RequestIDKey)
		if requestID == nil {
			requestID = uuid.New().String()
		}
		auditReqContext := []any{
			"rqid", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"forwarded-for", request.HeaderParameter("x-forwarded-for"),
			"remote-addr", r.RemoteAddr,
		}
		user := security.GetUserFromContext(r.Context())
		if user != nil {
			auditReqContext = append(
				auditReqContext,
				"user", user.EMail,
				"tenant", user.Tenant,
			)
		}

		if r.Method != http.MethodGet && r.Body != nil {
			bodyReader := r.Body
			body, err := io.ReadAll(bodyReader)
			r.Body = io.NopCloser(bytes.NewReader(body))
			if err != nil {
				logger.Errorf("unable to read request body: %v", err)
				response.WriteHeader(http.StatusInternalServerError)
				return
			}
			auditReqContext = append(auditReqContext, "body", string(body))
		}

		err := a.Index(auditReqContext...)
		if err != nil {
			logger.Errorf("unable to index error: %v", err)
			response.WriteHeader(http.StatusInternalServerError)
			return
		}

		bufferedResponseWriter := &bufferedHttpResponseWriter{
			w: response.ResponseWriter,
		}
		response.ResponseWriter = bufferedResponseWriter

		chain.ProcessFilter(request, response)

		auditRespContext := append(auditReqContext,
			"resp", bufferedResponseWriter.Content(),
			"status-code", response.StatusCode(),
		)
		err = a.Index(auditRespContext...)
		if err != nil {
			logger.Errorf("unable to index error: %v", err)
			response.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

type bufferedHttpResponseWriter struct {
	w http.ResponseWriter

	buf    bytes.Buffer
	header int
}

func (w *bufferedHttpResponseWriter) Header() http.Header {
	return w.w.Header()
}

func (w *bufferedHttpResponseWriter) Write(b []byte) (int, error) {
	(&w.buf).Write(b)
	return w.w.Write(b)
}

func (w *bufferedHttpResponseWriter) WriteHeader(h int) {
	w.header = h
	w.w.WriteHeader(h)
}

func (w *bufferedHttpResponseWriter) Content() string {
	return w.buf.String()
}
