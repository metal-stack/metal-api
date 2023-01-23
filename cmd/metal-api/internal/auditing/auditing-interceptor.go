package auditing

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/emicklei/go-restful/v3"
	"github.com/google/uuid"
	"github.com/metal-stack/metal-lib/rest"
	"github.com/metal-stack/security"
	"go.uber.org/zap"
	"google.golang.org/grpc"
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
				"subject", user.Subject,
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
				"subject", user.Subject,
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

func HttpFilter(a Auditing, logger *zap.SugaredLogger, shouldAudit func(*url.URL) bool) restful.FilterFunction {
	if a == nil {
		logger.Fatal("cannot use nil auditing to create http middleware")
	}
	return restful.HttpMiddlewareHandlerToFilter(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
				break
			default:
				h.ServeHTTP(w, r)
				return
			}
			if !shouldAudit(r.URL) {
				h.ServeHTTP(w, r)
				return
			}
			requestID := r.Context().Value(rest.RequestIDKey)

			auditReqContext := []any{
				"rqid", requestID,
				"method", r.Method,
				"path", r.URL.Path,
			}
			user := security.GetUserFromContext(r.Context())
			if user != nil {
				auditReqContext = append(
					auditReqContext,
					"user", user.EMail,
					"subject", user.Subject,
					"tenant", user.Tenant,
				)
			}
			if r.Method != http.MethodGet && r != nil && r.Body != nil {
				bodyReader := r.Body
				body, err := io.ReadAll(bodyReader)
				r.Body = io.NopCloser(bytes.NewReader(body))
				if err != nil {
					logger.Errorf("unable to read request body: %v", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				auditReqContext = append(auditReqContext, "body", string(body))
			}

			err := a.Index(auditReqContext...)
			if err != nil {
				logger.Errorf("unable to index error: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			bufferedResponseWriter := &bufferedHttpResponseWriter{
				ResponseWriter: w,
			}
			h.ServeHTTP(bufferedResponseWriter, r)

			respBody := string(bufferedResponseWriter.body)

			auditRespContext := append(auditReqContext,
				"resp", respBody,
				"status-code", bufferedResponseWriter.statusCode,
			)
			err = a.Index(auditRespContext...)
			if err != nil {
				logger.Errorf("unable to index error: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		})
	})
}

type bufferedHttpResponseWriter struct {
	http.ResponseWriter

	statusCode int
	body       []byte
}

func (w *bufferedHttpResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

func (w *bufferedHttpResponseWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return w.ResponseWriter.Write(b)
}

func (w *bufferedHttpResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
