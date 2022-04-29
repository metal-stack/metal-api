package grpc

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/metal-stack/masterdata-api/pkg/interceptors/grpc_internalerror"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"github.com/metal-stack/metal-lib/bus"
	"go.uber.org/zap"
)

const (
	defaultResponseInterval = 5 * time.Second
	defaultCheckInterval    = 60 * time.Second
)

type Datasource interface {
	FindMachineByID(machineID string) (*metal.Machine, error)
	CreateMachine(*metal.Machine) error
	UpdateMachine(old, new *metal.Machine) error
	ProvisioningEventForMachine(machineID, event, message string) (*metal.ProvisioningEventContainer, error)
}

type ServerConfig struct {
	Publisher                bus.Publisher
	Datasource               Datasource
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
	*WaitService
	*SupwdService
	*EventService
	ds             Datasource
	logger         *zap.SugaredLogger
	server         *grpc.Server
	grpcPort       int
	tlsEnabled     bool
	caCertFile     string
	serverCertFile string
	serverKeyFile  string
}

func NewServer(cfg *ServerConfig) (*Server, error) {
	if cfg.ResponseInterval <= 0 {
		cfg.ResponseInterval = defaultResponseInterval
	}
	if cfg.CheckInterval <= 0 {
		cfg.CheckInterval = defaultCheckInterval
	}

	waitService, err := NewWaitService(cfg)
	if err != nil {
		return nil, err
	}
	supwdService := NewSupwdService(cfg)

	eventService := NewEventService(cfg)

	s := &Server{
		WaitService:    waitService,
		SupwdService:   supwdService,
		EventService:   eventService,
		ds:             cfg.Datasource,
		logger:         cfg.Logger,
		grpcPort:       cfg.GrpcPort,
		tlsEnabled:     cfg.TlsEnabled,
		caCertFile:     cfg.CaCertFile,
		serverCertFile: cfg.ServerCertFile,
		serverKeyFile:  cfg.ServerKeyFile,
	}

	return s, nil
}

func (s *Server) Serve() error {
	addr := fmt.Sprintf(":%d", s.grpcPort)

	kaep := keepalive.EnforcementPolicy{
		MinTime:             5 * time.Second, // If a client pings more than once every 5 seconds, terminate the connection
		PermitWithoutStream: true,            // Allow pings even when there are no active streams
	}
	kasp := keepalive.ServerParameters{
		Time:    5 * time.Second, // Ping the client if it is idle for 5 seconds to ensure the connection is still active
		Timeout: 1 * time.Second, // Wait 1 second for the ping ack before assuming the connection is dead
	}

	grpcLogger := s.logger.Named("grpc").Desugar()
	grpc_zap.ReplaceGrpcLoggerV2(grpcLogger)
	s.server = grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(kaep),
		grpc.KeepaliveParams(kasp),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_ctxtags.StreamServerInterceptor(),
			grpc_prometheus.StreamServerInterceptor,
			grpc_zap.StreamServerInterceptor(grpcLogger),
			grpc_internalerror.StreamServerInterceptor(),
			grpc_recovery.StreamServerInterceptor(),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_prometheus.UnaryServerInterceptor,
			grpc_zap.UnaryServerInterceptor(grpcLogger),
			grpc_internalerror.UnaryServerInterceptor(),
			grpc_recovery.UnaryServerInterceptor(),
		)),
	)
	grpc_prometheus.Register(s.server)

	v1.RegisterSuperUserPasswordServer(s.server, s.SupwdService)
	v1.RegisterWaitServer(s.server, s.WaitService)
	v1.RegisterEventServiceServer(s.server, s.EventService)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	if s.tlsEnabled {
		cert, err := os.ReadFile(s.serverCertFile)
		if err != nil {
			s.logger.Fatalw("failed to serve gRPC", "error", err)
		}
		key, err := os.ReadFile(s.serverKeyFile)
		if err != nil {
			s.logger.Fatalw("failed to serve gRPC", "error", err)
		}
		serverCert, err := tls.X509KeyPair(cert, key)
		if err != nil {
			return err
		}

		caCert, err := os.ReadFile(s.caCertFile)
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

	s.logger.Infow("serve gRPC", "address", addr)
	return s.server.Serve(listener)
}
