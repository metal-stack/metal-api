package permissions

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/security"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

var (
	errComparer = cmp.Comparer(func(x, y error) bool {
		if x == nil && y == nil {
			return true
		}
		if x == nil {
			return false
		}
		if y == nil {
			return false
		}
		return x.Error() == y.Error()
	})
)

func Test_regoDecider_Decide(t *testing.T) {
	tests := []struct {
		name        string
		req         *http.Request
		u           *security.User
		permissions Permissions
		wantErr     error
	}{
		{
			name: "user with right permissions wants to list images",
			req: &http.Request{
				URL:    mustParseURL("https://api.metal-stack.io/v1/image"),
				Method: http.MethodGet,
			},
			u: &security.User{
				Name: "metal-stack-user",
			},
			permissions: Permissions{
				"metal.v1.image.list": true,
			},
			wantErr: nil,
		},
		{
			name: "user has no permissions",
			req: &http.Request{
				URL:    mustParseURL("https://api.metal-stack.io/v1/image"),
				Method: http.MethodGet,
			},
			u: &security.User{
				Name: "metal-stack-user",
			},
			permissions: Permissions{},
			wantErr:     fmt.Errorf("access denied"),
		},
		{
			name: "user has wrong permissions",
			req: &http.Request{
				URL:    mustParseURL("https://api.metal-stack.io/v1/image"),
				Method: http.MethodGet,
			},
			u: &security.User{
				Name: "metal-stack-user",
			},
			permissions: Permissions{
				"something": true,
			},
			wantErr: fmt.Errorf("access denied"),
		},
		{
			name: "wrong method used endpoint",
			req: &http.Request{
				URL:    mustParseURL("https://api.metal-stack.io/v1/image"),
				Method: http.MethodDelete,
			},
			u: &security.User{
				Name: "metal-stack-user",
			},
			permissions: Permissions{
				"metal.v1.image.list": true,
			},
			wantErr: fmt.Errorf("access denied"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := newRegoDecider(zaptest.NewLogger(t).Sugar(), "/")
			require.NoError(t, err)

			err = r.Decide(context.TODO(), tt.req, tt.u, tt.permissions)
			if diff := cmp.Diff(err, tt.wantErr, errComparer); diff != "" {
				t.Errorf("Decide() error mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func mustParseURL(u string) *url.URL {
	url, err := url.Parse(u)
	if err != nil {
		panic(err)
	}
	return url
}
