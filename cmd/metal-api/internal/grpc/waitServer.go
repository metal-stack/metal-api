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
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"io/ioutil"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"github.com/metal-stack/metal-lib/bus"
	"github.com/metal-stack/metal-lib/zapup"
	"go.uber.org/zap"
)

const (
	receiverHandlerTimeout = 15 * time.Second
	allocationTopicTTL     = time.Duration(30) * time.Second
)

func timeoutHandler(err bus.TimeoutError) error {
	zapup.MustRootLogger().Sugar().Error("Timeout processing event", "event", err.Event())
	return nil
}

type Datasource interface {
	FindMachineByID(machineID string) (*metal.Machine, error)
	UpdateMachine(old, new *metal.Machine) error
}

type WaitServerConfig struct {
	Publisher             bus.Publisher
	Datasource            Datasource
	Logger                *zap.SugaredLogger
	NsqTlsConfig          *bus.TLSConfig
	NsqlookupdHttpAddress string
	GrpcPort              int
	TlsEnabled            bool
	CaCertFile            string
	ServerCertFile        string
	ServerKeyFile         string
}

type WaitServer struct {
	bus.Publisher
	server         *grpc.Server
	ds             Datasource
	logger         *zap.SugaredLogger
	queueLock      *sync.RWMutex
	queue          map[string]chan bool
	GrpcPort       int
	TlsEnabled     bool
	CaCertFile     string
	ServerCertFile string
	ServerKeyFile  string
}

func NewWaitServer(cfg *WaitServerConfig) (*WaitServer, error) {
	c, err := bus.NewConsumer(zapup.MustRootLogger(), cfg.NsqTlsConfig, cfg.NsqlookupdHttpAddress)
	if err != nil {
		return nil, err
	}

	s := &WaitServer{
		Publisher:      cfg.Publisher,
		ds:             cfg.Datasource,
		logger:         cfg.Logger,
		queueLock:      new(sync.RWMutex),
		queue:          make(map[string]chan bool),
		GrpcPort:       cfg.GrpcPort,
		TlsEnabled:     cfg.TlsEnabled,
		CaCertFile:     cfg.CaCertFile,
		ServerCertFile: cfg.ServerCertFile,
		ServerKeyFile:  cfg.ServerKeyFile,
	}

	rand.Seed(time.Now().Unix())
	channel := fmt.Sprintf("alloc-%d#ephemeral", rand.Int())
	err = c.With(bus.LogLevel(bus.Warning)).
		MustRegister(metal.TopicAllocation.Name, channel).
		Consume(metal.AllocationEvent{}, func(message interface{}) error {
			evt := message.(*metal.AllocationEvent)
			s.logger.Debugw("got message", "topic", metal.TopicAllocation.Name, "channel", channel, "machineID", evt.MachineID)
			s.handleAllocation(evt.MachineID)
			return nil
		}, 5, bus.Timeout(receiverHandlerTimeout, timeoutHandler), bus.TTL(allocationTopicTTL))
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *WaitServer) NotifyAllocated(machineID string) error {
	err := s.Publish(metal.TopicAllocation.Name, &metal.AllocationEvent{MachineID: machineID})
	if err != nil {
		s.logger.Errorw("failed to publish machine allocation event", "topic", metal.TopicAllocation.Name, "machineID", machineID, "error", err)
	} else {
		s.logger.Debugw("published machine allocation event", "topic", metal.TopicAllocation.Name, "machineID", machineID)
	}
	return err
}

func (s *WaitServer) Serve() error {
	addr := fmt.Sprintf(":%d", s.GrpcPort)

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
	v1.RegisterWaitServer(s.server, s)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	if s.TlsEnabled {
		cert, err := ioutil.ReadFile(s.ServerCertFile)
		if err != nil {
			s.logger.Fatalw("failed to serve gRPC", "error", err)
		}
		key, err := ioutil.ReadFile(s.ServerKeyFile)
		if err != nil {
			s.logger.Fatalw("failed to serve gRPC", "error", err)
		}
		serverCert, err := tls.X509KeyPair(cert, key)
		if err != nil {
			return err
		}

		caCert, err := ioutil.ReadFile(s.CaCertFile)
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
		})
	}

	s.logger.Infow("serve gRPC", "address", addr)
	return s.server.Serve(listener)
}

func (s *WaitServer) Wait(req *v1.WaitRequest, srv v1.Wait_WaitServer) error {
	machineID := req.MachineID
	s.logger.Infow("wait for allocation called by", "machineID", machineID)

	m, err := s.ds.FindMachineByID(machineID)
	if err != nil {
		return err
	}
	allocated := m.Allocation != nil
	if allocated {
		return nil
	}

	// machine is not yet allocated, so we set the waiting flag
	err = s.updateWaitingFlag(machineID, true)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			return
		}
		err := s.updateWaitingFlag(machineID, false)
		if err != nil {
			s.logger.Errorw("unable to remove waiting flag from machine", "machineID", machineID, "error", err)
		}
	}()

	// we also create and listen to a channel that will be used as soon as the machine is allocated
	s.queueLock.Lock()
	can, ok := s.queue[machineID]
	if !ok {
		can = make(chan bool)
		s.queue[machineID] = can
	}
	s.queueLock.Unlock()

	defer func() {
		s.queueLock.Lock()
		delete(s.queue, machineID)
		close(can)
		s.queueLock.Unlock()
	}()

	nextCheck := time.Now()
	ctx := srv.Context()
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return err
		case allocated := <-can:
			if allocated {
				return nil
			}
		case now := <-time.After(5 * time.Second):
			if now.After(nextCheck) {
				m, err = s.ds.FindMachineByID(machineID)
				if err != nil {
					return err
				}
				allocated := m.Allocation != nil
				if allocated {
					return nil
				}
				nextCheck = now.Add(60 * time.Second)
			}
			err = sendKeepPatientResponse(srv)
			if err != nil {
				return err
			}
		}
	}
}

// https://github.com/grpc/grpc-go/issues/1229#issuecomment-302755717
func sendKeepPatientResponse(srv v1.Wait_WaitServer) error {
	errChan := make(chan error, 1)
	ctx := srv.Context()
	go func() {
		errChan <- srv.Send(&v1.KeepPatientResponse{})
		close(errChan)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}

func (s *WaitServer) handleAllocation(machineID string) {
	s.queueLock.RLock()
	defer s.queueLock.RUnlock()

	can, ok := s.queue[machineID]
	if ok {
		can <- true
	}
}

func (s *WaitServer) updateWaitingFlag(machineID string, flag bool) error {
	m, err := s.ds.FindMachineByID(machineID)
	if err != nil {
		return err
	}
	old := *m
	m.Waiting = flag
	return s.ds.UpdateMachine(&old, m)
}
