package headscale

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	headscalev1 "github.com/juanfont/headscale/gen/go/headscale/v1"
	"github.com/juanfont/headscale/hscontrol/db"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

type HeadscaleClient struct {
	client headscalev1.HeadscaleServiceClient

	address             string
	controlPlaneAddress string

	conn   *grpc.ClientConn
	logger *slog.Logger
}

func NewHeadscaleClient(addr, controlPlaneAddr, apiKey string, logger *slog.Logger) (client *HeadscaleClient, err error) {
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

	if err = h.connect(apiKey); err != nil {
		return nil, fmt.Errorf("failed to connect to Headscale server: %w", err)
	}

	return h, nil
}

// Connect or reconnect to Headscale server
func (h *HeadscaleClient) connect(apiKey string) (err error) {
	grpcOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithPerRPCCredentials(tokenAuth{
			token: apiKey,
		}),
	}

	h.conn, err = grpc.NewClient(h.address, grpcOptions...)
	if err != nil {
		return fmt.Errorf("failed to connect to headscale server %s: %w", h.address, err)
	}

	h.client = headscalev1.NewHeadscaleServiceClient(h.conn)

	return
}

func (h *HeadscaleClient) GetControlPlaneAddress() string {
	return h.controlPlaneAddress
}

func (h *HeadscaleClient) UserExists(ctx context.Context, name string) bool {
	req := &headscalev1.ListUsersRequest{
		Name: name,
	}
	resp, err := h.client.ListUsers(ctx, req)
	if err != nil {
		return false
	}
	// Should only return one user.
	for _, user := range resp.Users {
		if user.Name == name {
			return true
		}
	}

	return false
}

func (h *HeadscaleClient) CreateUser(ctx context.Context, name string) error {
	req := &headscalev1.CreateUserRequest{
		Name: name,
	}
	_, err := h.client.CreateUser(ctx, req)
	// TODO: this error check is pretty rough, but it's not easily possible to compare the proto error directly :/
	if err != nil && !strings.Contains(err.Error(), db.ErrUserExists.Error()) {
		return fmt.Errorf("failed to create new VPN user: %w", err)
	}

	return nil
}

func (h *HeadscaleClient) CreatePreAuthKey(ctx context.Context, user string, expiration time.Time, isEphemeral bool) (key string, err error) {
	req := &headscalev1.CreatePreAuthKeyRequest{
		User:       user,
		Expiration: timestamppb.New(expiration),
		Ephemeral:  isEphemeral,
	}
	resp, err := h.client.CreatePreAuthKey(ctx, req)
	if err != nil || resp == nil || resp.PreAuthKey == nil {
		return "", fmt.Errorf("failed to create new Auth Key: %w", err)
	}

	return resp.PreAuthKey.Key, nil
}

func (h *HeadscaleClient) NodesConnected(ctx context.Context) ([]*headscalev1.Node, error) {
	resp, err := h.client.ListNodes(ctx, &headscalev1.ListNodesRequest{})
	if err != nil || resp == nil {
		return nil, fmt.Errorf("failed to list machines: %w", err)
	}

	return resp.Nodes, nil
}

// DeleteNode removes the node entry from headscale DB
func (h *HeadscaleClient) DeleteNode(ctx context.Context, machineID, projectID string) (err error) {
	machine, err := h.getNode(ctx, machineID, projectID)
	if err != nil || machine == nil {
		return err
	}

	req := &headscalev1.DeleteNodeRequest{
		NodeId: machine.Id,
	}
	if _, err := h.client.DeleteNode(ctx, req); err != nil {
		return fmt.Errorf("failed to delete machine: %w", err)
	}

	return nil
}

func (h *HeadscaleClient) getNode(ctx context.Context, machineID, projectID string) (machine *headscalev1.Node, err error) {
	req := &headscalev1.ListNodesRequest{
		User: projectID,
	}
	resp, err := h.client.ListNodes(ctx, req)
	if err != nil || resp == nil {
		return nil, fmt.Errorf("failed to list machines: %w", err)
	}

	for _, m := range resp.Nodes {
		if m.Name == machineID {
			return m, nil
		}
	}

	return nil, nil
}

// Close client
func (h *HeadscaleClient) Close() error {
	return h.conn.Close()
}
