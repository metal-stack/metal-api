package grpc

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"github.com/metal-stack/metal-lib/zapup"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"io/ioutil"
	"net"
	"time"
)

func Serve(ds *datastore.RethinkStore) {
	logger := zapup.MustRootLogger().Sugar()

	grpcPort := viper.GetInt("grpc-port")
	addr := fmt.Sprintf(":%d", grpcPort)

	kaep := keepalive.EnforcementPolicy{
		MinTime:             5 * time.Second, // If a client pings more than once every 5 seconds, terminate the connection
		PermitWithoutStream: true,            // Allow pings even when there are no active streams
	}
	kasp := keepalive.ServerParameters{
		Time:    5 * time.Second, // Ping the client if it is idle for 5 seconds to ensure the connection is still active
		Timeout: 1 * time.Second, // Wait 1 second for the ping ack before assuming the connection is dead
	}
	grpcServer := grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(kaep),
		grpc.KeepaliveParams(kasp),
	)
	v1.RegisterWaitServer(grpcServer, NewWaitServer(ds))

	tlsConfig := &tls.Config{
		NextProtos: []string{"h2"},
	}
	if viper.GetBool("grpc-tls-enabled") {
		cert, err := ioutil.ReadFile(viper.GetString("grpc-server-cert-file"))
		if err != nil {
			logger.Fatalw("failed to serve gRPC", "error", err)
		}
		key, err := ioutil.ReadFile(viper.GetString("grpc-server-key-file"))
		if err != nil {
			logger.Fatalw("failed to serve gRPC", "error", err)
		}
		serverCert, err := tls.X509KeyPair(cert, key)
		if err != nil {
			logger.Fatalw("failed to serve gRPC", "error", err)
		}

		caCert, err := ioutil.ReadFile(viper.GetString("grpc-ca-cert-file"))
		if err != nil {
			logger.Fatalw("failed to serve gRPC", "error", err)
		}
		caCertPool := x509.NewCertPool()
		ok := caCertPool.AppendCertsFromPEM(caCert)
		if !ok {
			logger.Fatalf("failed to serve gRPC", "error", "bad certificate")
		}
		tlsConfig.Certificates = []tls.Certificate{serverCert}
		tlsConfig.ClientCAs = caCertPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	} else {
		tlsConfig.ClientAuth = tls.NoClientCert
	}

	fmt.Printf("grpc on port %d\n", grpcPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Fatalw("failed to serve gRPC", "error", err)
	}

	logger.Infow("serve gRPC", "address", addr)
	err = grpcServer.Serve(tls.NewListener(listener, tlsConfig))
	if err != nil {
		logger.Fatalw("failed to serve gRPC", "error", err)
	}
}
