package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"runtime/debug"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/metal-stack/masterdata-api/pkg/interceptors/grpc_internalerror"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"github.com/metal-stack/metal-lib/bus"
	"go.uber.org/zap"
)

const (
	defaultResponseInterval = 5 * time.Second
	defaultCheckInterval    = 60 * time.Second
)

type ServerConfig struct {
	Publisher                bus.Publisher
	Store                    *datastore.RethinkStore
	Logger                   *zap.SugaredLogger
	NsqTlsConfig             *bus.TLSConfig
	NsqlookupdHttpAddress    string
	GrpcPort                 int
	TlsEnabled               bool
	CaCertFile               string
	ServerCertFile           string
	ServerKeyFile            string
	ResponseInterval         time.Duration
	CheckInterval            time.Duration
	BMCSuperUserPasswordFile string
}

type Server struct {
	grpcServer  *grpc.Server
	ds          *datastore.RethinkStore
	logger      *zap.SugaredLogger
	cfg         *ServerConfig
	listener    net.Listener
	waitService *WaitService
}

func NewServer(cfg *ServerConfig) (*Server, error) {
	addr := fmt.Sprintf(":%d", cfg.GrpcPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	if cfg.ResponseInterval <= 0 {
		cfg.ResponseInterval = defaultResponseInterval
	}
	if cfg.CheckInterval <= 0 {
		cfg.CheckInterval = defaultCheckInterval
	}

	kaep := keepalive.EnforcementPolicy{
		MinTime:             5 * time.Second, // If a client pings more than once every 5 seconds, terminate the connection
		PermitWithoutStream: true,            // Allow pings even when there are no active streams
	}
	kasp := keepalive.ServerParameters{
		Time:    5 * time.Second, // Ping the client if it is idle for 5 seconds to ensure the connection is still active
		Timeout: 1 * time.Second, // Wait 1 second for the ping ack before assuming the connection is dead
	}

	grpcLogger := cfg.Logger.Named("grpc").Desugar()
	grpc_zap.ReplaceGrpcLoggerV2(grpcLogger)

	recoveryOpt := grpc_recovery.WithRecoveryHandlerContext(
		func(ctx context.Context, p any) error {
			grpcLogger.Sugar().Errorf("[PANIC] %s stack:%s", p, string(debug.Stack()))
			return status.Errorf(codes.Internal, "%s", p)
		},
	)

	server := grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(kaep),
		grpc.KeepaliveParams(kasp),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_ctxtags.StreamServerInterceptor(),
			grpc_prometheus.StreamServerInterceptor,
			grpc_zap.StreamServerInterceptor(grpcLogger),
			grpc_internalerror.StreamServerInterceptor(),
			grpc_recovery.StreamServerInterceptor(recoveryOpt),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_prometheus.UnaryServerInterceptor,
			grpc_zap.UnaryServerInterceptor(grpcLogger),
			grpc_internalerror.UnaryServerInterceptor(),
			grpc_recovery.UnaryServerInterceptor(recoveryOpt),
		)),
	)
	grpc_prometheus.Register(server)

	waitService, err := NewWaitService(cfg)
	if err != nil {
		return nil, err
	}
	eventService := NewEventService(cfg)
	bootService := NewBootService(cfg, eventService)

	v1.RegisterWaitServer(server, waitService)
	v1.RegisterEventServiceServer(server, eventService)
	v1.RegisterBootServiceServer(server, bootService)

	if cfg.TlsEnabled {
		cert, err := os.ReadFile(cfg.ServerCertFile)
		if err != nil {
			return nil, fmt.Errorf("failed to serve gRPC: %w", err)
		}
		key, err := os.ReadFile(cfg.ServerKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to serve gRPC: %w", err)
		}
		serverCert, err := tls.X509KeyPair(cert, key)
		if err != nil {
			return nil, err
		}

		caCert, err := os.ReadFile(cfg.CaCertFile)
		if err != nil {
			return nil, err
		}
		caCertPool := x509.NewCertPool()
		ok := caCertPool.AppendCertsFromPEM(caCert)
		if !ok {
			return nil, err
		}

		listener = tls.NewListener(listener, &tls.Config{
			NextProtos:   []string{"h2"},
			Certificates: []tls.Certificate{serverCert},
			ClientCAs:    caCertPool,
			ClientAuth:   tls.RequireAndVerifyClientCert,
			MinVersion:   tls.VersionTLS12,
		})
	}

	return &Server{
		grpcServer:  server,
		ds:          cfg.Store,
		logger:      cfg.Logger,
		cfg:         cfg,
		listener:    listener,
		waitService: waitService,
	}, nil
}

func (s *Server) Serve() error {
	s.logger.Infow("serve gRPC", "address", s.listener.Addr())
	return s.grpcServer.Serve(s.listener)
}

func (s *Server) Stop() error {
	s.logger.Infow("stop gRPC")
	return nil
}

func (s *Server) WaitService() *WaitService {
	return s.waitService
}
