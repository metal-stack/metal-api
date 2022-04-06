package service

import (
	"fmt"
	"testing"

	restful "github.com/emicklei/go-restful/v3"
	v1 "github.com/metal-stack/masterdata-api/api/rest/v1"
	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	mdmv1mock "github.com/metal-stack/masterdata-api/api/v1/mocks"
	mdm "github.com/metal-stack/masterdata-api/pkg/client"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/security"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockedTenantService struct {
	t  *testing.T
	ws *restful.WebService
}

func NewMockedTenantService(t *testing.T, tenantServiceMock func(mock *mdmv1mock.TenantServiceClient)) *MockedTenantService {
	tsc := &mdmv1mock.TenantServiceClient{}
	if tenantServiceMock != nil {
		tenantServiceMock(tsc)
	}
	mdc := mdm.NewMock(&mdmv1mock.ProjectServiceClient{}, tsc)
	ws := NewTenant(mdc)
	return &MockedTenantService{
		t:  t,
		ws: ws,
	}
}

//nolint:golint,unused
func (m *MockedTenantService) list(user *security.User, resp interface{}) int {
	return webRequestGet(m.t, m.ws, user, user, "/v1/tenant", resp)
}

func (m *MockedTenantService) get(id string, user *security.User, resp interface{}) int {
	return webRequestGet(m.t, m.ws, user, user, "/v1/tenant/"+id, resp)
}

func Test_tenantResource_getTenant(t *testing.T) {

	tests := []struct {
		name              string
		userScenarios     []security.User
		tenantServiceMock func(mock *mdmv1mock.TenantServiceClient)
		id                string
		wantStatus        int
		want              *v1.TenantResponse
		wantErr           *httperrors.HTTPErrorResponse
	}{
		{
			name:          "entity forbidden for user with no privileges",
			userScenarios: []security.User{*noUser},
			id:            "121",
			wantStatus:    403,
			wantErr:       httperrors.Forbidden(fmt.Errorf("you are not member in one of [k8s_kaas-view maas-all-all-view k8s_kaas-edit maas-all-all-edit k8s_kaas-admin maas-all-all-admin]")),
		},
		{
			name:          "entity allowed for user with view privileges",
			userScenarios: []security.User{*testViewUser},
			id:            "122",
			tenantServiceMock: func(mock *mdmv1mock.TenantServiceClient) {
				mock.On("Get", testifymock.Anything, &mdmv1.TenantGetRequest{Id: "122"}).Return(&mdmv1.TenantResponse{Tenant: &mdmv1.Tenant{Name: "t122"}}, nil)
			},
			want:       &v1.TenantResponse{Tenant: v1.Tenant{Name: "t122"}},
			wantStatus: 200,
			wantErr:    nil,
		},
		{
			name:          "entity allowed for user with admin privileges",
			userScenarios: []security.User{*testAdminUser},
			tenantServiceMock: func(mock *mdmv1mock.TenantServiceClient) {
				mock.On("Get", testifymock.Anything, &mdmv1.TenantGetRequest{Id: "123"}).Return(&mdmv1.TenantResponse{Tenant: &mdmv1.Tenant{Name: "t123"}}, nil)
			},
			id:         "123",
			want:       &v1.TenantResponse{Tenant: v1.Tenant{Name: "t123"}},
			wantStatus: 200,
			wantErr:    nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		for _, user := range tt.userScenarios {
			user := user
			name := fmt.Sprintf("%s/%s", tt.name, user)
			t.Run(name, func(t *testing.T) {
				service := NewMockedTenantService(t, tt.tenantServiceMock)
				if tt.wantErr != nil {
					got := httperrors.HTTPErrorResponse{}
					status := service.get(tt.id, &user, &got)
					require.Equal(t, tt.wantStatus, status)
					require.Equal(t, *tt.wantErr, got)
					return
				}

				got := v1.TenantResponse{}
				status := service.get(tt.id, &user, &got)
				require.Equal(t, tt.wantStatus, status)
				require.Equal(t, *tt.want, got)
			})
		}
	}
}

func Test_tenantResource_listTenants(t *testing.T) {

	tests := []struct {
		name              string
		userScenarios     []security.User
		tenantServiceMock func(mock *mdmv1mock.TenantServiceClient)
		wantStatus        int
		want              []*v1.Tenant
		wantErr           *httperrors.HTTPErrorResponse
	}{
		{
			name:          "entity forbidden for user with no privileges",
			userScenarios: []security.User{*noUser},
			wantStatus:    403,
			wantErr:       httperrors.Forbidden(fmt.Errorf("you are not member in one of [k8s_kaas-view maas-all-all-view k8s_kaas-edit maas-all-all-edit k8s_kaas-admin maas-all-all-admin]")),
		},
		{
			name:          "entity allowed for user with view privileges",
			userScenarios: []security.User{*testViewUser},
			tenantServiceMock: func(mock *mdmv1mock.TenantServiceClient) {
				mock.On("Find", testifymock.Anything, &mdmv1.TenantFindRequest{}).Return(&mdmv1.TenantListResponse{Tenants: []*mdmv1.Tenant{{Name: "t121"}, {Name: "t122"}}}, nil)
			},
			want:       []*v1.Tenant{{Name: "t121"}, {Name: "t122"}},
			wantStatus: 200,
			wantErr:    nil,
		},
		{
			name:          "entity allowed for user with admin privileges",
			userScenarios: []security.User{*testAdminUser},
			tenantServiceMock: func(mock *mdmv1mock.TenantServiceClient) {
				mock.On("Find", testifymock.Anything, &mdmv1.TenantFindRequest{}).Return(&mdmv1.TenantListResponse{Tenants: []*mdmv1.Tenant{{Name: "t123"}}}, nil)
			},
			want:       []*v1.Tenant{{Name: "t123"}},
			wantStatus: 200,
			wantErr:    nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		for _, user := range tt.userScenarios {
			user := user
			name := fmt.Sprintf("%s/%s", tt.name, user)
			t.Run(name, func(t *testing.T) {
				service := NewMockedTenantService(t, tt.tenantServiceMock)

				if tt.wantErr != nil {
					got := httperrors.HTTPErrorResponse{}
					status := service.list(&user, &got)
					require.Equal(t, tt.wantStatus, status)
					require.Equal(t, *tt.wantErr, got)
					return
				}

				var got []*v1.Tenant
				status := service.list(&user, &got)
				require.Equal(t, tt.wantStatus, status)
				require.Equal(t, tt.want, got)
			})
		}
	}
}
