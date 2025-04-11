package grpc

import (
	"context"
	"testing"

	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"github.com/metal-stack/v"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetVersion(t *testing.T) {

	vs := &versionService{}
	ctx := context.Background()

	expected := v1.GetVersionResponse{Version: v.Version, Revision: v.Revision, BuildDate: v.BuildDate, GitSha1: v.GitSHA1}

	result, err := vs.Get(ctx, &v1.GetVersionRequest{})

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expected.Version, result.Version)
	assert.Equal(t, expected.Revision, result.Revision)
	assert.Equal(t, expected.BuildDate, result.BuildDate)
	assert.Equal(t, expected.GitSha1, result.GitSha1)
}
