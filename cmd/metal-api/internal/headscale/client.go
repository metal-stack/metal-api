package headscale

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	headscalecore "github.com/juanfont/headscale"
	headscalev1 "github.com/juanfont/headscale/gen/go/headscale/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

var (
	apiKeyExpiration = 24 * 90 * time.Hour
)

type HeadscaleClient struct {
	client headscalev1.HeadscaleServiceClient

	address             string
	ControlPlaneAddress string

	apiKeyPrefix    string
	oldAPIKeyPrefix string

	mu         sync.Mutex
	wg         sync.WaitGroup
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
		ControlPlaneAddress: controlPlaneAddr,

		mu:     sync.Mutex{},
		wg:     sync.WaitGroup{},
		logger: logger,
	}
	h.ctx, h.cancelFunc = context.WithCancel(context.Background())

	// Validate API Key
	if h.apiKeyPrefix, err = getApiKeyPrefix(apiKey); err != nil {
		return nil, err
	}

	if err = h.connect(apiKey); err != nil {
		return nil, fmt.Errorf("failed to connect to Headscale server: %w", err)
	}

	if err = h.replaceApiKey(); err != nil {
		return nil, fmt.Errorf("failed to replace Headscale API Key: %w", err)
	}

	go h.rotateApiKey()

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

// Refresh API key when apiKeyExpiration reaches it's half life
func (h *HeadscaleClient) rotateApiKey() {
	h.wg.Add(1)
	defer h.wg.Done()

	ticker := time.NewTicker(apiKeyExpiration / 2)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			// Try replacement until it works
			// Log error messages
			for {
				if err := h.replaceApiKey(); err != nil {
					h.logger.Error("failed to replace Headscale API Key", zap.Error(err))
					time.Sleep(time.Second)
					continue
				}

				break
			}
		}
	}
}

// Expires current API Key and replaces it with a new one
func (h *HeadscaleClient) replaceApiKey() (err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// If old API key is already deleted
	// replace the current with a new one
	if h.oldAPIKeyPrefix == "" {
		// Create new API Key
		expiration := time.Now().Add(apiKeyExpiration)
		createRequest := &headscalev1.CreateApiKeyRequest{
			Expiration: timestamppb.New(expiration),
		}
		response, err := h.client.CreateApiKey(h.ctx, createRequest)
		if err != nil {
			return fmt.Errorf("failed to create new API Key: %w", err)
		}

		// Validate new API Key
		var newApiKeyPrefix string
		if newApiKeyPrefix, err = getApiKeyPrefix(response.ApiKey); err != nil {
			return fmt.Errorf("generated invalid API Key during rotation: %w", err)
		}

		// Connect to Headscale server with new key
		if err = h.connect(response.ApiKey); err != nil {
			return fmt.Errorf("failed to reconnect to Headscale server: %w", err)
		}
		h.apiKeyPrefix = newApiKeyPrefix
		h.oldAPIKeyPrefix = h.apiKeyPrefix
	}

	// Expire the old API Key
	expireRequest := &headscalev1.ExpireApiKeyRequest{
		Prefix: h.oldAPIKeyPrefix,
	}

	if _, err = h.client.ExpireApiKey(h.ctx, expireRequest); err != nil {
		return fmt.Errorf("failed to expire current API Key: %w", err)
	}

	h.oldAPIKeyPrefix = ""

	return nil
}

func (h *HeadscaleClient) NamespaceExists(name string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	getNSRequest := &headscalev1.GetNamespaceRequest{
		Name: name,
	}
	if _, err := h.client.GetNamespace(h.ctx, getNSRequest); err != nil {
		return false
	}

	return true
}

func (h *HeadscaleClient) CreateNamespace(name string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	req := &headscalev1.CreateNamespaceRequest{
		Name: name,
	}
	_, err := h.client.CreateNamespace(h.ctx, req)
	if err != nil && !errors.Is(headscalecore.Error("Namespace already exists"), err) {
		return fmt.Errorf("failed to create new VPN namespace: %w", err)
	}

	return nil
}

func (h *HeadscaleClient) CreatePreAuthKey(namespace string, expiration time.Time) (key string, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

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
	h.wg.Wait()

	return h.conn.Close()
}

func getApiKeyPrefix(apiKey string) (prefix string, err error) {
	s := strings.Split(apiKey, ".")
	if len(s) != 2 {
		return "", fmt.Errorf("API Key had invalid format, should look like prefix.password")
	}

	return s[0], nil
}
