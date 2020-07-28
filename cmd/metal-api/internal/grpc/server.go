package grpc

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/metal-stack/masterdata-api/pkg/interceptors/grpc_internalerror"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"github.com/metal-stack/metal-lib/zapup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"io/ioutil"
	"net"
	"time"
)

func Serve(ws *WaitServer) error {
	logger := zapup.MustRootLogger().Sugar()

	addr := fmt.Sprintf(":%d", ws.GrpcPort)

	kaep := keepalive.EnforcementPolicy{
		MinTime:             5 * time.Second, // If a client pings more than once every 5 seconds, terminate the connection
		PermitWithoutStream: true,            // Allow pings even when there are no active streams
	}
	kasp := keepalive.ServerParameters{
		Time:    5 * time.Second, // Ping the client if it is idle for 5 seconds to ensure the connection is still active
		Timeout: 1 * time.Second, // Wait 1 second for the ping ack before assuming the connection is dead
	}

	desugaredLogger := logger.Named("grpc").Desugar()
	grpc_zap.ReplaceGrpcLoggerV2(desugaredLogger)
	grpcServer := grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(kaep),
		grpc.KeepaliveParams(kasp),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_ctxtags.StreamServerInterceptor(),
			grpc_prometheus.StreamServerInterceptor,
			grpc_zap.StreamServerInterceptor(desugaredLogger),
			grpc_internalerror.StreamServerInterceptor(),
			grpc_recovery.StreamServerInterceptor(),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_prometheus.UnaryServerInterceptor,
			grpc_zap.UnaryServerInterceptor(desugaredLogger),
			grpc_internalerror.UnaryServerInterceptor(),
			grpc_recovery.UnaryServerInterceptor(),
		)),
	)
	grpc_prometheus.Register(grpcServer)
	v1.RegisterWaitServer(grpcServer, ws)

	tlsConfig := &tls.Config{
		NextProtos: []string{"h2"},
	}
	if ws.TlsEnabled {
		cert, err := ioutil.ReadFile(ws.ServerCertFile)
		if err != nil {
			logger.Fatalw("failed to serve gRPC", "error", err)
		}
		key, err := ioutil.ReadFile(ws.ServerKeyFile)
		if err != nil {
			logger.Fatalw("failed to serve gRPC", "error", err)
		}
		serverCert, err := tls.X509KeyPair(cert, key)
		if err != nil {
			return err
		}

		caCert, err := ioutil.ReadFile(ws.CaCertFile)
		if err != nil {
			return err
		}
		caCertPool := x509.NewCertPool()
		ok := caCertPool.AppendCertsFromPEM(caCert)
		if !ok {
			return err
		}
		tlsConfig.Certificates = []tls.Certificate{serverCert}
		tlsConfig.ClientCAs = caCertPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	} else {
		tlsConfig.ClientAuth = tls.NoClientCert
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	logger.Infow("serve gRPC", "address", addr)
	return grpcServer.Serve(tls.NewListener(listener, tlsConfig))
}
