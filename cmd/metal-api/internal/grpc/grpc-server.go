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

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/auditing"
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
	Context                  context.Context
	Publisher                bus.Publisher
	Consumer                 *bus.Consumer
	Store                    *datastore.RethinkStore
	Logger                   *zap.SugaredLogger
	GrpcPort                 int
	TlsEnabled               bool
	CaCertFile               string
	ServerCertFile           string
	ServerKeyFile            string
	ResponseInterval         time.Duration
	CheckInterval            time.Duration
	BMCSuperUserPasswordFile string
	Auditing                 auditing.Auditing

	integrationTestAllocator chan string
}

func Run(cfg *ServerConfig) error {
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

	log := cfg.Logger.Named("grpc")
	grpc_zap.ReplaceGrpcLoggerV2(log.Desugar())

	recoveryOpt := grpc_recovery.WithRecoveryHandlerContext(
		func(ctx context.Context, p any) error {
			log.Errorf("[PANIC] %s stack:%s", p, string(debug.Stack()))
			return status.Errorf(codes.Internal, "%s", p)
		},
	)

	shouldAudit := func(fullMethod string) bool {
		switch fullMethod {
		case "/api.v1.BootService/Register",
			"/api.v1.EventService/Send":
			return true
		default:
			return false
		}
	}

	server := grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(kaep),
		grpc.KeepaliveParams(kasp),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			auditing.StreamServerInterceptor(cfg.Auditing, log.Named("auditing-grpc"), shouldAudit),
			grpc_ctxtags.StreamServerInterceptor(),
			grpc_prometheus.StreamServerInterceptor,
			grpc_zap.StreamServerInterceptor(log.Desugar()),
			grpc_internalerror.StreamServerInterceptor(),
			grpc_recovery.StreamServerInterceptor(recoveryOpt),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			auditing.UnaryServerInterceptor(cfg.Auditing, log.Named("auditing-grpc"), shouldAudit),
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_prometheus.UnaryServerInterceptor,
			grpc_zap.UnaryServerInterceptor(log.Desugar()),
			grpc_internalerror.UnaryServerInterceptor(),
			grpc_recovery.UnaryServerInterceptor(recoveryOpt),
		)),
	)
	grpc_prometheus.Register(server)

	eventService := NewEventService(cfg)
	bootService := NewBootService(cfg, eventService)

	err := bootService.initWaitEndpoint()
	if err != nil {
		return err
	}

	v1.RegisterEventServiceServer(server, eventService)
	v1.RegisterBootServiceServer(server, bootService)

	// this is only for the integration test of this package
	if cfg.integrationTestAllocator != nil {
		go func() {
			for {
				machineID := <-cfg.integrationTestAllocator
				bootService.handleAllocation(machineID)
			}
		}()
	}

	addr := fmt.Sprintf(":%d", cfg.GrpcPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

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
		log.Infow("serve gRPC", "address", listener.Addr())
		err = server.Serve(listener)
	}()

	<-cfg.Context.Done()
	server.Stop()

	return err
}
