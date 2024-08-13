package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net"
	"os"
	"runtime/debug"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel/trace"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metrics"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"github.com/metal-stack/metal-lib/auditing"
	"github.com/metal-stack/metal-lib/bus"
)

const (
	defaultResponseInterval = 5 * time.Second
	defaultCheckInterval    = 60 * time.Second
)

type ServerConfig struct {
	Context                  context.Context
	Publisher                bus.Publisher
	Consumer                 *bus.Consumer
	Store                    *datastore.RethinkStore
	Logger                   *slog.Logger
	Listener                 net.Listener
	TlsEnabled               bool
	CaCertFile               string
	ServerCertFile           string
	ServerKeyFile            string
	ResponseInterval         time.Duration
	CheckInterval            time.Duration
	BMCSuperUserPasswordFile string
	Auditing                 auditing.Auditing
	IPMISuperUser            metal.MachineIPMISuperUser
}

func Run(cfg *ServerConfig) error {
	if cfg.ResponseInterval <= 0 {
		cfg.ResponseInterval = defaultResponseInterval
	}
	if cfg.CheckInterval <= 0 {
		cfg.CheckInterval = defaultCheckInterval
	}
	if cfg.Publisher == nil || cfg.Consumer == nil {
		return fmt.Errorf("nsq publisher and consumer must be specified")
	}

	kaep := keepalive.EnforcementPolicy{
		MinTime:             5 * time.Second, // If a client pings more than once every 5 seconds, terminate the connection
		PermitWithoutStream: true,            // Allow pings even when there are no active streams
	}
	kasp := keepalive.ServerParameters{
		Time:    5 * time.Second, // Ping the client if it is idle for 5 seconds to ensure the connection is still active
		Timeout: 1 * time.Second, // Wait 1 second for the ping ack before assuming the connection is dead
	}

	log := cfg.Logger.WithGroup("grpc")

	recoveryOpt := recovery.WithRecoveryHandlerContext(
		func(ctx context.Context, p any) error {
			log.Error("[PANIC] %s stack:%s", p, string(debug.Stack()))
			return status.Errorf(codes.Internal, "%s", p)
		},
	)

	srvMetrics := grpcprom.NewServerMetrics(
		grpcprom.WithServerHandlingTimeHistogram(
			grpcprom.WithHistogramBuckets([]float64{0.001, 0.01, 0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120}),
		),
	)
	exemplarFromContext := func(ctx context.Context) prometheus.Labels {
		if span := trace.SpanContextFromContext(ctx); span.IsSampled() {
			return prometheus.Labels{"traceID": span.TraceID().String()}
		}
		return nil
	}
	// Setup metric for panic recoveries.
	reg := prometheus.NewRegistry()
	reg.MustRegister(srvMetrics)
	reg.MustRegister(collectors.NewGoCollector())
	panicsTotal := promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Name: "grpc_req_panics_recovered_total",
		Help: "Total number of gRPC requests recovered from internal panic.",
	})
	grpcPanicRecoveryHandler := func(p any) (err error) {
		panicsTotal.Inc()
		log.Error("msg", "recovered from panic", "panic", p, "stack", debug.Stack())
		return status.Errorf(codes.Internal, "%s", p)
	}

	streamInterceptors := []grpc.StreamServerInterceptor{
		srvMetrics.StreamServerInterceptor(grpcprom.WithExemplarFromContext(exemplarFromContext)),
		logging.StreamServerInterceptor(interceptorLogger(log)),
		recovery.StreamServerInterceptor(recovery.WithRecoveryHandler(grpcPanicRecoveryHandler)),
	}
	unaryInterceptors := []grpc.UnaryServerInterceptor{
		// Order matters e.g. tracing interceptor have to create span first for the later exemplars to work.
		srvMetrics.UnaryServerInterceptor(),
		logging.UnaryServerInterceptor(interceptorLogger(log)),
		recovery.UnaryServerInterceptor(recovery.WithRecoveryHandler(grpcPanicRecoveryHandler)),
	}
	if cfg.Auditing != nil {
		shouldAudit := func(fullMethod string) bool {
			switch fullMethod {
			case "/api.v1.BootService/Register":
				return true
			default:
				return false
			}
		}
		auditStreamInterceptor, err := auditing.StreamServerInterceptor(cfg.Auditing, log.WithGroup("auditing-grpc"), shouldAudit)
		if err != nil {
			return err
		}
		auditUnaryInterceptor, err := auditing.UnaryServerInterceptor(cfg.Auditing, log.WithGroup("auditing-grpc"), shouldAudit)
		if err != nil {
			return err
		}
		streamInterceptors = append(streamInterceptors, auditStreamInterceptor)
		unaryInterceptors = append(unaryInterceptors, auditUnaryInterceptor)
	}

	unaryInterceptors = append(unaryInterceptors, metrics.GrpcMetrics, recovery.UnaryServerInterceptor(recoveryOpt))

	opts := []grpc.ServerOption{
		grpc.KeepaliveEnforcementPolicy(kaep),
		grpc.KeepaliveParams(kasp),
		grpc.ChainUnaryInterceptor(unaryInterceptors...),
		grpc.ChainStreamInterceptor(streamInterceptors...),
	}

	grpcServer := grpc.NewServer(opts...)
	srvMetrics.InitializeMetrics(grpcServer)

	eventService := NewEventService(cfg)
	bootService := NewBootService(cfg, eventService)

	err := bootService.initWaitEndpoint()
	if err != nil {
		return err
	}

	v1.RegisterEventServiceServer(grpcServer, eventService)
	v1.RegisterBootServiceServer(grpcServer, bootService)

	listener := cfg.Listener

	if cfg.TlsEnabled {
		cert, err := os.ReadFile(cfg.ServerCertFile)
		if err != nil {
			return fmt.Errorf("failed to serve gRPC: %w", err)
		}
		key, err := os.ReadFile(cfg.ServerKeyFile)
		if err != nil {
			return fmt.Errorf("failed to serve gRPC: %w", err)
		}
		serverCert, err := tls.X509KeyPair(cert, key)
		if err != nil {
			return err
		}

		caCert, err := os.ReadFile(cfg.CaCertFile)
		if err != nil {
			return err
		}
		caCertPool := x509.NewCertPool()
		ok := caCertPool.AppendCertsFromPEM(caCert)
		if !ok {
			return err
		}

		listener = tls.NewListener(listener, &tls.Config{
			NextProtos:   []string{"h2"},
			Certificates: []tls.Certificate{serverCert},
			ClientCAs:    caCertPool,
			ClientAuth:   tls.RequireAndVerifyClientCert,
			MinVersion:   tls.VersionTLS12,
		})
	}

	go func() {
		log.Info("serve gRPC", "address", listener.Addr())
		err = grpcServer.Serve(listener)
	}()

	<-cfg.Context.Done()
	grpcServer.Stop()

	return err
}

// interceptorLogger adapts slog logger to interceptor logger.
// This code is simple enough to be copied and not imported.
func interceptorLogger(l *slog.Logger) logging.Logger {
	return logging.LoggerFunc(func(_ context.Context, lvl logging.Level, msg string, fields ...any) {
		switch lvl {
		case logging.LevelDebug:
			l.Debug(msg, fields...)
		case logging.LevelInfo:
			l.Info(msg, fields...)
		case logging.LevelWarn:
			l.Warn(msg, fields...)
		case logging.LevelError:
			l.Error(msg, fields...)
		default:
			panic(fmt.Sprintf("unknown level %v", lvl))
		}
	})
}
