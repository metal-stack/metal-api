package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	"github.com/metal-stack/metal-lib/jwt/sec"

	"github.com/go-stack/stack"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/utils"
	"github.com/metal-stack/metal-lib/httperrors"

	"github.com/emicklei/go-restful"
	"github.com/metal-stack/security"
	"go.uber.org/zap"
)

var (
	BasePath = "/"
)

type webResource struct {
	ds *datastore.RethinkStore
}

type UserDirectory struct {
	viewer security.User
	edit   security.User
	admin  security.User

	metalUsers map[string]security.User
}

func NewUserDirectory(providerTenant string) *UserDirectory {
	ud := &UserDirectory{}

	// User.Name is used as AuthType for HMAC
	ud.viewer = security.User{
		EMail:  "metal-view@metal-stack.io",
		Name:   "Metal-View",
		Groups: sec.MergeResourceAccess(metal.ViewGroups),
		Tenant: providerTenant,
	}
	ud.edit = security.User{
		EMail:  "metal-edit@metal-stack.io",
		Name:   "Metal-Edit",
		Groups: sec.MergeResourceAccess(metal.EditGroups),
		Tenant: providerTenant,
	}
	ud.admin = security.User{
		EMail:  "metal-admin@metal-stack.io",
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

func (ud *UserDirectory) UserNames() []string {
	keys := make([]string, 0, len(ud.metalUsers))
	for k := range ud.metalUsers {
		keys = append(keys, k)
	}
	return keys
}

func (ud *UserDirectory) Get(user string) security.User {
	return ud.metalUsers[user]
}

func sendError(log *zap.Logger, rsp *restful.Response, opname string, errRsp *httperrors.HTTPErrorResponse) {
	sendErrorImpl(log, rsp, opname, errRsp, 1)
}

func sendErrorImpl(log *zap.Logger, rsp *restful.Response, opname string, errRsp *httperrors.HTTPErrorResponse, stackup int) {
	s := stack.Caller(stackup)
	response, merr := json.Marshal(errRsp)
	log.Error("service error", zap.String("operation", opname), zap.Int("status", errRsp.StatusCode), zap.String("error", errRsp.Message), zap.Stringer("service-caller", s), zap.String("resp", string(response)))
	if merr != nil {
		err := rsp.WriteError(http.StatusInternalServerError, fmt.Errorf("unable to format error string: %v", merr))
		if err != nil {
			log.Error("Failed to send response", zap.Error(err))
			return
		}
		return
	}
	err := rsp.WriteErrorString(errRsp.StatusCode, string(response))
	if err != nil {
		log.Error("Failed to send response", zap.Error(err))
		return
	}
}

func checkError(rq *restful.Request, rsp *restful.Response, opname string, err error) bool {
	log := utils.Logger(rq)
	if err != nil {
		if metal.IsNotFound(err) {
			sendErrorImpl(log, rsp, opname, httperrors.NotFound(err), 2)
			return true
		}
		if metal.IsConflict(err) {
			sendErrorImpl(log, rsp, opname, httperrors.Conflict(err), 2)
			return true
		}
		if metal.IsInternal(err) {
			sendErrorImpl(log, rsp, opname, httperrors.InternalServerError(err), 2)
			return true
		}
		if mdmv1.IsNotFound(err) {
			sendErrorImpl(log, rsp, opname, httperrors.NotFound(err), 2)
			return true
		}
		if mdmv1.IsConflict(err) {
			sendErrorImpl(log, rsp, opname, httperrors.Conflict(err), 2)
			return true
		}
		if mdmv1.IsInternal(err) {
			sendErrorImpl(log, rsp, opname, httperrors.InternalServerError(err), 2)
			return true
		}
		sendErrorImpl(log, rsp, opname, httperrors.NewHTTPError(http.StatusUnprocessableEntity, err), 2)
		return true
	}
	return false
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
		log := utils.Logger(request)
		lg := log.Sugar()
		usr := security.GetUser(request.Request)
		if !usr.HasGroup(acc...) {
			err := fmt.Errorf("you are not member in one of %+v", acc)
			lg.Infow("missing group", "user", usr, "required-group", acc)
			sendError(log, response, utils.CurrentFuncName(), httperrors.NewHTTPError(http.StatusForbidden, err))
			return
		}
		rf(request, response)
	}
}

func tenant(request *restful.Request) string {
	return security.GetUser(request.Request).Tenant
}

type TenantEnsurer struct {
	allowedTenants       map[string]bool
	excludedPathSuffixes []string
}

// NewTenantEnsurer creates a new ensurer with the given tenants.
func NewTenantEnsurer(tenants, excludedPathSuffixes []string) TenantEnsurer {
	result := TenantEnsurer{
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
		err := fmt.Errorf("tenant %s not allowed", tenantID)
		sendError(utils.Logger(req), resp, utils.CurrentFuncName(), httperrors.NewHTTPError(http.StatusForbidden, err))
		return
	}
	chain.ProcessFilter(req, resp)
}

// allowed checks if the given tenant is allowed (case insensitive)
func (e *TenantEnsurer) allowed(tenant string) bool {
	return e.allowedTenants[strings.ToLower(tenant)]
}
