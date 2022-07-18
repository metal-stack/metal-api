package service

import (
	"fmt"
	"strings"

	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	"github.com/metal-stack/metal-lib/jwt/sec"
	"github.com/metal-stack/metal-lib/rest"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-lib/httperrors"

	"github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/security"
	"go.uber.org/zap"
)

const (
	viewUserEmail  = "metal-view@metal-stack.io"
	editUserEmail  = "metal-edit@metal-stack.io"
	adminUserEmail = "metal-admin@metal-stack.io"
)

// BasePath is the URL base path for the metal-api
var BasePath = "/"

type webResource struct {
	log *zap.SugaredLogger
	ds  *datastore.RethinkStore
}

// logger returns the request logger from the request.
func (w *webResource) logger(rq *restful.Request) *zap.SugaredLogger {
	return rest.GetLoggerFromContext(rq.Request, w.log)
}

func (w *webResource) sendError(rq *restful.Request, rsp *restful.Response, httperr *httperrors.HTTPErrorResponse) {
	w.logger(rq).Desugar().WithOptions(zap.AddCallerSkip(1)).Sugar().Errorw("service error", "status", httperr.StatusCode, "error", httperr.Message)
	w.send(rq, rsp, httperr.StatusCode, httperr)
}

func (w *webResource) send(rq *restful.Request, rsp *restful.Response, status int, value any) {
	send(w.logger(rq), rsp, status, value)
}

func defaultError(err error) *httperrors.HTTPErrorResponse {
	if metal.IsNotFound(err) {
		return httperrors.NotFound(err)
	}
	if metal.IsConflict(err) {
		return httperrors.Conflict(err)
	}
	if metal.IsInternal(err) {
		return httperrors.InternalServerError(err)
	}
	if mdmv1.IsNotFound(err) {
		return httperrors.NotFound(err)
	}
	if mdmv1.IsConflict(err) {
		return httperrors.Conflict(err)
	}
	if mdmv1.IsInternal(err) {
		return httperrors.InternalServerError(err)
	}

	return httperrors.UnprocessableEntity(err)
}

func send(log *zap.SugaredLogger, rsp *restful.Response, status int, value any) {
	err := rsp.WriteHeaderAndEntity(status, value)
	if err != nil {
		log.Errorw("failed to send response", "error", err)
	}
}

// UserDirectory is the directory of users
type UserDirectory struct {
	viewer security.User
	edit   security.User
	admin  security.User

	metalUsers map[string]security.User
}

// NewUserDirectory creates a new user directory with default users
func NewUserDirectory(providerTenant string) *UserDirectory {
	ud := &UserDirectory{}

	// User.Name is used as AuthType for HMAC
	ud.viewer = security.User{
		EMail:  viewUserEmail,
		Name:   "Metal-View",
		Groups: sec.MergeResourceAccess(metal.ViewGroups),
		Tenant: providerTenant,
	}
	ud.edit = security.User{
		EMail:  editUserEmail,
		Name:   "Metal-Edit",
		Groups: sec.MergeResourceAccess(metal.EditGroups),
		Tenant: providerTenant,
	}
	ud.admin = security.User{
		EMail:  adminUserEmail,
		Name:   "Metal-Admin",
		Groups: sec.MergeResourceAccess(metal.AdminGroups),
		Tenant: providerTenant,
	}
	ud.metalUsers = map[string]security.User{
		"view":  ud.viewer,
		"edit":  ud.edit,
		"admin": ud.admin,
	}

	return ud
}

// UserNames returns the list of user names in the directory.
func (ud *UserDirectory) UserNames() []string {
	keys := make([]string, 0, len(ud.metalUsers))
	for k := range ud.metalUsers {
		keys = append(keys, k)
	}
	return keys
}

// Get a user by its user name.
func (ud *UserDirectory) Get(user string) security.User {
	return ud.metalUsers[user]
}

func viewer(rf restful.RouteFunction) restful.RouteFunction {
	return oneOf(rf, metal.ViewAccess...)
}

func editor(rf restful.RouteFunction) restful.RouteFunction {
	return oneOf(rf, metal.EditAccess...)
}

func admin(rf restful.RouteFunction) restful.RouteFunction {
	return oneOf(rf, metal.AdminAccess...)
}

func oneOf(rf restful.RouteFunction, acc ...security.ResourceAccess) restful.RouteFunction {
	return func(request *restful.Request, response *restful.Response) {
		usr := security.GetUser(request.Request)
		if !usr.HasGroup(acc...) {
			log := rest.GetLoggerFromContext(request.Request, nil)
			if log != nil {
				log.Infow("missing group", "user", usr, "required-group", acc)
			}

			httperr := httperrors.Forbidden(fmt.Errorf("you are not member in one of %+v", acc))

			err := response.WriteHeaderAndEntity(httperr.StatusCode, httperr)
			if err != nil && log != nil {
				log.Errorw("failed to send response", "error", err)
			}
			return
		}
		rf(request, response)
	}
}

func tenant(request *restful.Request) string {
	return security.GetUser(request.Request).Tenant
}

// TenantEnsurer holds allowed tenants and a list of path suffixes that
type TenantEnsurer struct {
	logger               *zap.SugaredLogger
	allowedTenants       map[string]bool
	excludedPathSuffixes []string
}

// NewTenantEnsurer creates a new ensurer with the given tenants.
func NewTenantEnsurer(log *zap.SugaredLogger, tenants, excludedPathSuffixes []string) TenantEnsurer {
	result := TenantEnsurer{
		logger:               log,
		allowedTenants:       make(map[string]bool),
		excludedPathSuffixes: excludedPathSuffixes,
	}
	for _, t := range tenants {
		result.allowedTenants[strings.ToLower(t)] = true
	}
	return result
}

// EnsureAllowedTenantFilter checks if the tenant of the user is allowed.
func (e *TenantEnsurer) EnsureAllowedTenantFilter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	p := req.Request.URL.Path

	// securing health checks would break monitoring tools
	// preventing liveliness would break status of machines
	for _, suffix := range e.excludedPathSuffixes {
		if strings.HasSuffix(p, suffix) {
			chain.ProcessFilter(req, resp)
			return
		}
	}

	// enforce tenant check otherwise
	tenantID := tenant(req)
	if !e.allowed(tenantID) {
		httperror := httperrors.Forbidden(fmt.Errorf("tenant %s not allowed", tenantID))

		requestLogger := rest.GetLoggerFromContext(req.Request, e.logger) // TODO: add caller stack
		requestLogger.Errorw("service error", "status", httperror.StatusCode, "error", httperror.Message)

		send(requestLogger, resp, httperror.StatusCode, httperror.Message)
		return
	}

	chain.ProcessFilter(req, resp)
}

// allowed checks if the given tenant is allowed (case insensitive)
func (e *TenantEnsurer) allowed(tenant string) bool {
	return e.allowedTenants[strings.ToLower(tenant)]
}
