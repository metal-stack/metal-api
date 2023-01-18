package auditing

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type auditingContextKey string

var auditingCorrelationIDKey auditingContextKey = "auditing-correlation-id"

func UnaryServerInterceptor(a Auditing, logger *zap.SugaredLogger, shouldAudit func(fullMethod string) bool) grpc.UnaryServerInterceptor {
	if a == nil {
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
			return handler(ctx, req)
		}
	}
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		if !shouldAudit(info.FullMethod) {
			return handler(ctx, req)
		}
		correlationID := uuid.New().String()
		childCtx := context.WithValue(ctx, auditingCorrelationIDKey, correlationID)
		err = a.Index("correlation-id", correlationID, "method", info.FullMethod, "kind", "grpc", "req", req)
		if err != nil {
			return nil, err
		}
		resp, err = handler(childCtx, req)
		if err != nil {
			err2 := a.Index("correlation-id", correlationID, "method", info.FullMethod, "kind", "grpc", "err", err)
			if err2 != nil {
				logger.Errorf("unable to index error: %v", err2)
			}
			return nil, err
		}
		err = a.Index("correlation-id", correlationID, "method", info.FullMethod, "kind", "grpc", "resp", resp)
		return resp, err
	}
}

func StreamServerInterceptor(a Auditing, logger *zap.SugaredLogger, shouldAudit func(fullMethod string) bool) grpc.StreamServerInterceptor {
	if a == nil {
		return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			return handler(srv, ss)
		}
	}
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !shouldAudit(info.FullMethod) {
			return handler(srv, ss)
		}
		correlationID := uuid.New().String()
		err := a.Index("kind", "grpc-stream", "stream", srv)
		if err != nil {
			return err
		}
		err = handler(srv, ss)
		if err != nil {
			err2 := a.Index("correlation-id", correlationID, "method", info.FullMethod, "kind", "grpc-stream", "err", err)
			if err2 != nil {
				logger.Errorf("unable to index error: %v", err2)
			}
			return err
		}
		err = a.Index("correlation-id", correlationID, "method", info.FullMethod, "kind", "grpc-stream")
		return err
	}
}

func HttpMiddleware(a Auditing, logger *zap.SugaredLogger, shouldAudit func(*url.URL) bool) func(h http.Handler) http.Handler {
	if a == nil {
		return func(h http.Handler) http.Handler {
			return h
		}
	}
	return func(h http.Handler) http.Handler {
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
			correlationID := uuid.NewString()
			r = r.WithContext(context.WithValue(r.Context(), auditingCorrelationIDKey, correlationID))

			auditReqContext := []any{
				"correlation-id", correlationID,
				"method", r.Method,
				"path", r.URL.Path,
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
	}
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
