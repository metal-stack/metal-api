package headscale

import (
	"context"
)

// Implements google.golang.org/grpc/credentials.PerRPCCredentials interface
type tokenAuth struct {
	token string
}

func (t tokenAuth) GetRequestMetadata(
	ctx context.Context,
	in ...string,
) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + t.token,
	}, nil
}

func (tokenAuth) RequireTransportSecurity() bool {
	return true
}
