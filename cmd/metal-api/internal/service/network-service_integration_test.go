//go:build integration
// +build integration

package service

import (
	"testing"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/stretchr/testify/require"
)

func TestSearchNetworksIntegration(t *testing.T) {
	te := createTestEnvironment(t)
	defer te.teardown()

	nfr := v1.NetworkFindRequest{
		NetworkSearchQuery: datastore.NetworkSearchQuery{
			Prefixes: []string{
				"10.0.0.0/16",
			},
		},
	}

	var resp []v1.NetworkResponse
	code := te.networkFind(t, nfr, &resp)
	require.Equal(t, 200, code)
	require.NotNil(t, resp)
	require.Len(t, resp, 1)
	require.ElementsMatch(t, resp[0].Prefixes, []string{"10.0.0.0/16"})
}
