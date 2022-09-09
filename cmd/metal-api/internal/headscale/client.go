package headscale

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	headscalecore "github.com/juanfont/headscale"
	headscalev1 "github.com/juanfont/headscale/gen/go/headscale/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

type HeadscaleClient struct {
	client headscalev1.HeadscaleServiceClient

	address             string
	controlPlaneAddress string

	ctx        context.Context
	conn       *grpc.ClientConn
	cancelFunc context.CancelFunc
	logger     *zap.SugaredLogger
}

func NewHeadscaleClient(addr, controlPlaneAddr, apiKey string, logger *zap.SugaredLogger) (client *HeadscaleClient, err error) {
	if addr != "" || apiKey != "" {
		if addr == "" {
			return nil, fmt.Errorf("headscale address should be set with api key")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("headscale api key should be set with address")
		}
	} else {
		return
	}

	h := &HeadscaleClient{
		address:             addr,
		controlPlaneAddress: controlPlaneAddr,

		logger: logger,
	}
	h.ctx, h.cancelFunc = context.WithCancel(context.Background())

	if err = h.connect(apiKey); err != nil {
		return nil, fmt.Errorf("failed to connect to Headscale server: %w", err)
	}

	return h, nil
}

// Connect or reconnect to Headscale server
func (h *HeadscaleClient) connect(apiKey string) (err error) {
	ctx, cancel := context.WithTimeout(h.ctx, 5*time.Second)
	defer cancel()

	grpcOptions := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithPerRPCCredentials(tokenAuth{
			token: apiKey,
		}),
	}

	h.conn, err = grpc.DialContext(ctx, h.address, grpcOptions...)
	if err != nil {
		return fmt.Errorf("failed to connect to headscale server %s: %w", h.address, err)
	}

	h.client = headscalev1.NewHeadscaleServiceClient(h.conn)

	return
}

func (h *HeadscaleClient) GetControlPlaneAddress() string {
	return h.controlPlaneAddress
}

func (h *HeadscaleClient) NamespaceExists(name string) bool {
	getNSRequest := &headscalev1.GetNamespaceRequest{
		Name: name,
	}
	if _, err := h.client.GetNamespace(h.ctx, getNSRequest); err != nil {
		return false
	}

	return true
}

func (h *HeadscaleClient) CreateNamespace(name string) error {
	req := &headscalev1.CreateNamespaceRequest{
		Name: name,
	}
	_, err := h.client.CreateNamespace(h.ctx, req)
	if err != nil && !errors.Is(headscalecore.Error(headscalecore.ErrNamespaceExists.Error()), err) { // FIXME: how to properly check this error?
		return fmt.Errorf("failed to create new VPN namespace: %w", err)
	}

	return nil
}

func (h *HeadscaleClient) CreatePreAuthKey(namespace string, expiration time.Time) (key string, err error) {
	req := &headscalev1.CreatePreAuthKeyRequest{
		Namespace:  namespace,
		Expiration: timestamppb.New(expiration),
	}
	resp, err := h.client.CreatePreAuthKey(h.ctx, req)
	if err != nil || resp == nil || resp.PreAuthKey == nil {
		return "", fmt.Errorf("failed to create new Auth Key: %w", err)
	}

	return resp.PreAuthKey.Key, nil
}

// Close client
func (h *HeadscaleClient) Close() error {
	h.cancelFunc()
	return h.conn.Close()
}
