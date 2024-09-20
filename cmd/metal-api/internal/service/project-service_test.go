package service

import (
	"errors"
	"fmt"
	"log/slog"
	"testing"

	restful "github.com/emicklei/go-restful/v3"
	v1 "github.com/metal-stack/masterdata-api/api/rest/v1"
	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	mdmv1mock "github.com/metal-stack/masterdata-api/api/v1/mocks"
	mdm "github.com/metal-stack/masterdata-api/pkg/client"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/security"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

type MockedProjectService struct {
	t  *testing.T
	ws *restful.WebService
}

var (
	noUser = &security.User{
		Tenant: "provider",
		Groups: []security.ResourceAccess{},
	}
	testViewUser = &security.User{
		Tenant: "provider",
		Groups: metal.ViewGroups,
	}
	testAdminUser = &security.User{
		Tenant: "provider",
		Groups: metal.AdminGroups,
	}
)

func NewMockedProjectService(t *testing.T, projectServiceMock func(mock *mdmv1mock.ProjectServiceClient), dsmock func(mock *r.Mock)) *MockedProjectService {
	psc := &mdmv1mock.ProjectServiceClient{}
	if projectServiceMock != nil {
		projectServiceMock(psc)
	}
	mdc := mdm.NewMock(psc, &mdmv1mock.TenantServiceClient{}, nil, nil)
	ds, mock := datastore.InitMockDB(t)
	if dsmock != nil {
		dsmock(mock)
	}
	ws := NewProject(slog.Default(), ds, mdc)
	return &MockedProjectService{
		t:  t,
		ws: ws,
	}
}

//nolint:golint,unused
func (m *MockedProjectService) list(user *security.User, resp interface{}) int {
	return webRequestGet(m.t, m.ws, user, user, "/v1/project", resp)
}

func (m *MockedProjectService) get(id string, user *security.User, resp interface{}) int {
	return webRequestGet(m.t, m.ws, user, user, "/v1/project/"+id, resp)
}

func (m *MockedProjectService) delete(id string, user *security.User, resp interface{}) int {
	return webRequestDelete(m.t, m.ws, user, user, "/v1/project/"+id, resp)
}

func (m *MockedProjectService) create(pcr *mdmv1.ProjectCreateRequest, user *security.User, resp interface{}) int {
	return webRequestPut(m.t, m.ws, user, pcr, "/v1/project", resp)
}

func (m *MockedProjectService) update(pcr *mdmv1.ProjectUpdateRequest, user *security.User, resp interface{}) int {
	return webRequestPost(m.t, m.ws, user, pcr, "/v1/project", resp)
}

func Test_projectResource_findProject(t *testing.T) {
	tests := []struct {
		name               string
		userScenarios      []security.User
		projectServiceMock func(mock *mdmv1mock.ProjectServiceClient)
		id                 string
		wantStatus         int
		want               *v1.ProjectResponse
		wantErr            *httperrors.HTTPErrorResponse
	}{
		{
			name:          "entity forbidden for user with no privileges",
			userScenarios: []security.User{*noUser},
			id:            "121",
			wantStatus:    403,
			wantErr:       httperrors.Forbidden(errors.New("you are not member in one of [k8s_kaas-view maas-all-all-view k8s_kaas-edit maas-all-all-edit k8s_kaas-admin maas-all-all-admin]")),
		},
		{
			name:          "entity allowed for user with view privileges",
			userScenarios: []security.User{*testViewUser},
			id:            "122",
			projectServiceMock: func(mock *mdmv1mock.ProjectServiceClient) {
				mock.On("Get", testifymock.Anything, &mdmv1.ProjectGetRequest{Id: "122"}).Return(&mdmv1.ProjectResponse{Project: &mdmv1.Project{}}, nil)
			},
			wantStatus: 422,
			wantErr:    httperrors.UnprocessableEntity(errors.New("project does not have a projectID")),
		},
		{
			name:          "entity allowed for user with admin privileges",
			userScenarios: []security.User{*testAdminUser},
			projectServiceMock: func(mock *mdmv1mock.ProjectServiceClient) {
				mock.On("Get", testifymock.Anything, &mdmv1.ProjectGetRequest{Id: "123"}).Return(&mdmv1.ProjectResponse{Project: &mdmv1.Project{}}, nil)
			},
			id:         "123",
			wantStatus: 422,
			wantErr:    httperrors.UnprocessableEntity(errors.New("project does not have a projectID")),
		},
	}
	for i := range tests {
		tt := tests[i]
		for j := range tt.userScenarios {
			user := tt.userScenarios[j]
			name := fmt.Sprintf("%s/%s", tt.name, user)
			t.Run(name, func(t *testing.T) {
				service := NewMockedProjectService(t, tt.projectServiceMock, nil)

				if tt.wantErr != nil {
					var got *httperrors.HTTPErrorResponse
					status := service.get(tt.id, &user, &got)
					require.Equal(t, tt.wantStatus, status)
					require.Equal(t, tt.wantErr, got)
					return
				}

				var got *v1.ProjectResponse
				status := service.get(tt.id, &user, &got)
				require.Equal(t, tt.wantStatus, status)
				require.Equal(t, tt.want, got)
			})
		}
	}
}

func Test_projectResource_createProject(t *testing.T) {
	tests := []struct {
		name               string
		userScenarios      []security.User
		projectServiceMock func(mock *mdmv1mock.ProjectServiceClient)
		pcr                *mdmv1.ProjectCreateRequest
		wantStatus         int
		want               *v1.ProjectResponse
		wantErr            *httperrors.HTTPErrorResponse
	}{
		{
			name:          "entity forbidden for user with no privileges",
			userScenarios: []security.User{*noUser},
			pcr:           &mdmv1.ProjectCreateRequest{Project: &mdmv1.Project{}},
			wantStatus:    403,
			wantErr:       httperrors.Forbidden(errors.New("you are not member in one of [k8s_kaas-admin maas-all-all-admin]")),
		},
		{
			name:          "entity allowed for user with view privileges",
			userScenarios: []security.User{*testViewUser},
			pcr:           &mdmv1.ProjectCreateRequest{Project: &mdmv1.Project{}},
			projectServiceMock: func(mock *mdmv1mock.ProjectServiceClient) {
				mock.On("Create", testifymock.Anything, &mdmv1.ProjectCreateRequest{Project: &mdmv1.Project{}}).Return(&mdmv1.ProjectResponse{Project: &mdmv1.Project{}}, nil)
			},
			wantStatus: 403,
			wantErr:    httperrors.Forbidden(errors.New("you are not member in one of [k8s_kaas-admin maas-all-all-admin]")),
		},
		{
			name:          "entity allowed for user with admin privileges",
			userScenarios: []security.User{*testAdminUser},
			pcr:           &mdmv1.ProjectCreateRequest{Project: &mdmv1.Project{}},
			projectServiceMock: func(mock *mdmv1mock.ProjectServiceClient) {
				mock.On("Create", testifymock.Anything, &mdmv1.ProjectCreateRequest{Project: &mdmv1.Project{}}).Return(&mdmv1.ProjectResponse{Project: &mdmv1.Project{}}, nil)
			},
			wantStatus: 400,
			wantErr:    httperrors.BadRequest(errors.New("no tenant given")),
		},
	}
	for i := range tests {
		tt := tests[i]
		for j := range tt.userScenarios {
			user := tt.userScenarios[j]
			name := fmt.Sprintf("%s/%s", tt.name, user)
			t.Run(name, func(t *testing.T) {
				service := NewMockedProjectService(t, tt.projectServiceMock, nil)

				if tt.wantErr != nil {
					var got *httperrors.HTTPErrorResponse
					status := service.create(tt.pcr, &user, &got)
					require.Equal(t, tt.wantStatus, status)
					require.Equal(t, tt.wantErr, got)
					return
				}

				var got *v1.ProjectResponse
				status := service.create(tt.pcr, &user, &got)
				require.Equal(t, tt.wantStatus, status)
				require.Equal(t, tt.want, got)
			})
		}
	}
}

func Test_projectResource_deleteProject(t *testing.T) {
	tests := []struct {
		name               string
		userScenarios      []security.User
		projectServiceMock func(mock *mdmv1mock.ProjectServiceClient)
		dsMock             func(mock *r.Mock)
		id                 string
		wantStatus         int
		want               *v1.ProjectResponse
		wantErr            *httperrors.HTTPErrorResponse
	}{
		{
			name:          "entity forbidden for user with no privileges",
			userScenarios: []security.User{*noUser},
			id:            "121",
			wantStatus:    403,
			wantErr:       httperrors.Forbidden(errors.New("you are not member in one of [k8s_kaas-admin maas-all-all-admin]")),
		},
		{
			name:          "entity allowed for user with view privileges",
			userScenarios: []security.User{*testViewUser},
			id:            "122",
			projectServiceMock: func(mock *mdmv1mock.ProjectServiceClient) {
				mock.On("Delete", testifymock.Anything, &mdmv1.ProjectDeleteRequest{Id: "122"}).Return(&mdmv1.ProjectResponse{Project: &mdmv1.Project{}}, nil)
			},
			wantStatus: 403,
			wantErr:    httperrors.Forbidden(errors.New("you are not member in one of [k8s_kaas-admin maas-all-all-admin]")),
		},
		{
			name:          "entity allowed for user with admin privileges",
			userScenarios: []security.User{*testAdminUser},
			id:            "123",
			projectServiceMock: func(mock *mdmv1mock.ProjectServiceClient) {
				mock.On("Get", testifymock.Anything, &mdmv1.ProjectGetRequest{Id: "123"}).Return(&mdmv1.ProjectResponse{Project: &mdmv1.Project{Meta: &mdmv1.Meta{Id: "123"}}}, nil)
				mock.On("Delete", testifymock.Anything, &mdmv1.ProjectDeleteRequest{Id: "123"}).Return(&mdmv1.ProjectResponse{Project: &mdmv1.Project{}}, nil)
			},
			dsMock: func(mock *r.Mock) {
				mock.On(r.DB("mockdb").Table("machine").Filter(r.MockAnything(), r.FilterOpts{})).Return([]metal.Machines{}, nil)
				mock.On(r.DB("mockdb").Table("network").Filter(r.MockAnything(), r.FilterOpts{})).Return([]metal.Networks{}, nil)
				mock.On(r.DB("mockdb").Table("ip").Filter(r.MockAnything(), r.FilterOpts{})).Return([]metal.IPs{}, nil)
				mock.On(r.DB("mockdb").Table("size")).Return([]metal.Size{}, nil)
				mock.On(r.DB("mockdb").Table("sizereservation").Filter(r.MockAnything(), r.FilterOpts{})).Return([]metal.SizeReservation{}, nil)
			},
			want:       &v1.ProjectResponse{},
			wantStatus: 200,
			wantErr:    nil,
		},
	}
	for i := range tests {
		tt := tests[i]
		for j := range tt.userScenarios {
			user := tt.userScenarios[j]
			name := fmt.Sprintf("%s/%s", tt.name, user)
			t.Run(name, func(t *testing.T) {
				service := NewMockedProjectService(t, tt.projectServiceMock, tt.dsMock)

				if tt.wantErr != nil {
					var got *httperrors.HTTPErrorResponse
					status := service.delete(tt.id, &user, &got)
					require.Equal(t, tt.wantStatus, status)
					require.Equal(t, tt.wantErr, got)
					return
				}

				var got *v1.ProjectResponse
				status := service.delete(tt.id, &user, &got)
				require.Equal(t, tt.wantStatus, status)
				require.Equal(t, tt.want, got)
			})
		}
	}
}

func Test_projectResource_updateProject(t *testing.T) {
	tests := []struct {
		name               string
		userScenarios      []security.User
		projectServiceMock func(mock *mdmv1mock.ProjectServiceClient)
		pur                *mdmv1.ProjectUpdateRequest
		wantStatus         int
		want               *v1.ProjectResponse
		wantErr            *httperrors.HTTPErrorResponse
	}{
		{
			name:          "entity forbidden for user with no privileges",
			userScenarios: []security.User{*noUser},
			pur:           &mdmv1.ProjectUpdateRequest{Project: &mdmv1.Project{}},
			wantStatus:    403,
			wantErr:       httperrors.Forbidden(errors.New("you are not member in one of [k8s_kaas-admin maas-all-all-admin]")),
		},
		{
			name:          "entity allowed for user with view privileges",
			userScenarios: []security.User{*testViewUser},
			pur:           &mdmv1.ProjectUpdateRequest{Project: &mdmv1.Project{}},
			projectServiceMock: func(mock *mdmv1mock.ProjectServiceClient) {
				mock.On("Update", testifymock.Anything, &mdmv1.ProjectUpdateRequest{Project: &mdmv1.Project{}}).Return(&mdmv1.ProjectResponse{Project: &mdmv1.Project{}}, nil)
			},
			wantStatus: 403,
			wantErr:    httperrors.Forbidden(errors.New("you are not member in one of [k8s_kaas-admin maas-all-all-admin]")),
		},
		{
			name:          "entity allowed for user with admin privileges",
			userScenarios: []security.User{*testAdminUser},
			pur:           &mdmv1.ProjectUpdateRequest{Project: &mdmv1.Project{}},
			projectServiceMock: func(mock *mdmv1mock.ProjectServiceClient) {
				mock.On("Update", testifymock.Anything, &mdmv1.ProjectUpdateRequest{Project: &mdmv1.Project{}}).Return(&mdmv1.ProjectResponse{Project: &mdmv1.Project{}}, nil)
			},
			wantStatus: 400,
			wantErr:    httperrors.BadRequest(errors.New("project and project.meta must be specified")),
		},
	}
	for i := range tests {
		tt := tests[i]
		for j := range tt.userScenarios {
			user := tt.userScenarios[j]
			name := fmt.Sprintf("%s/%s", tt.name, user)
			t.Run(name, func(t *testing.T) {
				service := NewMockedProjectService(t, tt.projectServiceMock, nil)

				if tt.wantErr != nil {
					var got *httperrors.HTTPErrorResponse
					status := service.update(tt.pur, &user, &got)
					require.Equal(t, tt.wantStatus, status)
					require.Equal(t, tt.wantErr, got)
					return
				}

				var got *v1.ProjectResponse
				status := service.update(tt.pur, &user, &got)
				require.Equal(t, tt.wantStatus, status)
				require.Equal(t, tt.want, got)
			})
		}
	}
}
