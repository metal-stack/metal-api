package headscale

import (
	"context"
	"fmt"

	headscalev1 "github.com/juanfont/headscale/gen/go/headscale/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	"strings"
	"time"
)

type HeadscaleClient struct {
	headscalev1.HeadscaleServiceClient

	address             string
	apiKeyPrefix        string
	ControlPlaneAddress string

	ctx        context.Context
	conn       *grpc.ClientConn
	cancelFunc context.CancelFunc
}

func NewHeadscaleClient(addr, controlPlaneAddr, apiKey string) (client *HeadscaleClient, err error) {
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
		ControlPlaneAddress: controlPlaneAddr,
	}
	if err = h.connect(apiKey); err != nil {
		return nil, fmt.Errorf("failed to connect to Headscale server: %w", err)
	}

	return h, nil
}

// Connect or reconnect to Headscale server
func (h *HeadscaleClient) connect(apiKey string) (err error) {
	// Validate API Key
	if h.apiKeyPrefix, err = getApiKeyPrefix(apiKey); err != nil {
		return err
	}

	h.ctx, h.cancelFunc = context.WithTimeout(context.Background(), 5*time.Second)
	grpcOptions := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithPerRPCCredentials(tokenAuth{
			token: apiKey,
		}),
	}

	h.conn, err = grpc.DialContext(h.ctx, h.address, grpcOptions...)
	if err != nil {
		return fmt.Errorf("failed to connect to headscale server %s: %w", h.address, err)
	}

	h.HeadscaleServiceClient = headscalev1.NewHeadscaleServiceClient(h.conn)

	return
}

// Expires current API Key and replaces it with a new one
func (h *HeadscaleClient) ReplaceApiKey() (err error) {
	oldApiKeyPrefix := h.apiKeyPrefix

	// Create new API Key
	expiration := time.Now().Add(24 * 90 * time.Hour)
	createRequest := &headscalev1.CreateApiKeyRequest{
		Expiration: timestamppb.New(expiration),
	}
	response, err := h.CreateApiKey(h.ctx, createRequest)
	if err != nil {
		return fmt.Errorf("failed to create new API Key: %w", err)
	}

	// Connect to Headscale server with new key
	if err = h.connect(response.ApiKey); err != nil {
		return fmt.Errorf("failed to reconnect to Headscale server: %w", err)
	}

	// Expire old API Key
	expireRequest := &headscalev1.ExpireApiKeyRequest{
		Prefix: oldApiKeyPrefix,
	}

	if _, err = h.ExpireApiKey(h.ctx, expireRequest); err != nil {
		return fmt.Errorf("failed to expire current API Key: %w", err)
	}

	return nil
}

// Close client
func (h *HeadscaleClient) Close() error {
	h.cancelFunc()
	return h.conn.Close()
}

func getApiKeyPrefix(apiKey string) (prefix string, err error) {
	s := strings.Split(apiKey, ".")
	if len(s) != 2 {
		return "", fmt.Errorf("API Key had invalid format, should look like prefix.password")
	}

	return s[0], nil
}
