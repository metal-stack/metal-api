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
	"github.com/stretchr/testify/assert"
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
		isAdmin     bool
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
			name: "user with right permissions wants to create image",
			req: &http.Request{
				URL:    mustParseURL("https://api.metal-stack.io/v1/image"),
				Method: http.MethodPut,
			},
			u: &security.User{
				Name: "metal-stack-user",
			},
			permissions: Permissions{
				"metal.v1.image.create": true,
			},
			wantErr: nil,
		},
		{
			name: "user with right permissions wants to create image and has admin permission",
			req: &http.Request{
				URL:    mustParseURL("https://api.metal-stack.io/v1/image"),
				Method: http.MethodPut,
			},
			u: &security.User{
				Name: "metal-stack-user",
			},
			permissions: Permissions{
				"metal.v1.image.create.admin": true,
			},
			isAdmin: true,
			wantErr: nil,
		},
		{
			name: "user can have regular and admin permissions and is still fine",
			req: &http.Request{
				URL:    mustParseURL("https://api.metal-stack.io/v1/image"),
				Method: http.MethodPut,
			},
			u: &security.User{
				Name: "metal-stack-user",
			},
			permissions: Permissions{
				"metal.v1.image.create":       true,
				"metal.v1.image.create.admin": true,
			},
			isAdmin: true,
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
			wantErr:     fmt.Errorf("access denied: missing permission on metal.v1.image.list"),
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
			wantErr: fmt.Errorf("access denied: missing permission on metal.v1.image.list"),
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
		{
			name: "access health endpoint that needs no permissions",
			req: &http.Request{
				URL:    mustParseURL("https://api.metal-stack.io/v1/health"),
				Method: http.MethodGet,
			},
			u: &security.User{
				Name: "metal-stack-user",
			},
			permissions: Permissions{},
			wantErr:     nil,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			r, err := newRegoDecider(zaptest.NewLogger(t).Sugar(), "/")
			require.NoError(t, err)

			isAdmin, err := r.Decide(context.TODO(), tt.req, tt.u, tt.permissions)
			if diff := cmp.Diff(err, tt.wantErr, errComparer); diff != "" {
				t.Errorf("Decide() error mismatch (-want +got):\n%s", diff)
			}
			assert.Equal(t, tt.isAdmin, isAdmin, "admin condition not properly evaluated")
		})
	}
}

// nolint
func mustParseURL(u string) *url.URL {
	url, err := url.Parse(u)
	if err != nil {
		panic(err)
	}
	return url
}

func Test_regoDecider_ListPermissions(t *testing.T) {
	tests := []struct {
		name    string
		want    []string
		wantErr bool
	}{
		{
			name: "permissions are listed",
			want: []string{
				"metal.v1.image.list",
				"metal.v1.image.get",
				"metal.v1.image.get-latest",
				"metal.v1.image.delete",
				"metal.v1.image.create",
				"metal.v1.image.update",
				"metal.v1.image.delete.admin",
				"metal.v1.image.create.admin",
				"metal.v1.image.update.admin",
			},
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			r, err := newRegoDecider(zaptest.NewLogger(t).Sugar(), "/")
			require.NoError(t, err)

			got, err := r.ListPermissions(context.TODO())
			require.NoError(t, err)
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("regoDecider.ListPermissions() = %v, want %v", got, tt.want)
			}
		})
	}
}
