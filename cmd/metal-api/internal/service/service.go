package service

import (
	"encoding/json"
	"fmt"
	"git.f-i-ts.de/cloud-native/metallib/jwt/sec"
	"net/http"
	"reflect"
	"strings"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/utils"
	"git.f-i-ts.de/cloud-native/metallib/httperrors"
	"github.com/go-stack/stack"

	"github.com/emicklei/go-restful"
	"github.com/metal-pod/security"
	"go.uber.org/zap"
)

// Some predefined users
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
		EMail:  "metal-view@metal-pod.io",
		Name:   "Metal-View",
		Groups: sec.MergeRessourceAccess(metal.ViewGroups),
		Tenant: providerTenant,
	}
	ud.edit = security.User{
		EMail:  "metal-edit@metal-pod.io",
		Name:   "Metal-Edit",
		Groups: sec.MergeRessourceAccess(metal.EditGroups),
		Tenant: providerTenant,
	}
	ud.admin = security.User{
		EMail:  "metal-admin@metal-pod.io",
		Name:   "Metal-Admin",
		Groups: sec.MergeRessourceAccess(metal.AdminGroups),
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
		rsp.WriteError(http.StatusInternalServerError, fmt.Errorf("unable to format error string: %v", merr))
		return
	}
	rsp.WriteErrorString(errRsp.StatusCode, string(response))
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
		sendErrorImpl(log, rsp, opname, httperrors.NewHTTPError(http.StatusUnprocessableEntity, err), 2)
		return true
	}
	return false
}

func (wr *webResource) handleReflectResponse(opname string, req *restful.Request, response *restful.Response, res []reflect.Value) {
	data := res[0].Interface()
	var err error
	if !res[1].IsNil() {
		err = res[1].Elem().Interface().(error)
	}
	if checkError(req, response, opname, err) {
		return
	}
	response.WriteEntity(data)
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

func oneOf(rf restful.RouteFunction, acc ...security.RessourceAccess) restful.RouteFunction {
	return func(request *restful.Request, response *restful.Response) {
		log := utils.Logger(request)
		lg := log.Sugar()
		usr := security.GetUser(request.Request)
		if !usr.HasGroup(acc...) {
			err := fmt.Errorf("you are not member in one of %+v", acc)
			lg.Infow("missing group", "user", usr, "required-group", acc)
			response.WriteHeaderAndEntity(http.StatusForbidden, httperrors.NewHTTPError(http.StatusForbidden, err))
			return
		}
		rf(request, response)
	}
}

func tenant(request *restful.Request) string {
	return security.GetUser(request.Request).Tenant
}

type TenantEnsurer struct {
	allowedTenants map[string]bool
}

// NewTenantEnsurer creates a new ensurer with the given tenants.
func NewTenantEnsurer(tenants []string) TenantEnsurer {
	result := TenantEnsurer{}
	result.allowedTenants = make(map[string]bool)
	for _, t := range tenants {
		result.allowedTenants[strings.ToLower(t)] = true
	}
	return result
}

// EnsureAllowedTenantFilter checks if the tenant of the user is allowed.
func (e *TenantEnsurer) EnsureAllowedTenantFilter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	tenantId := tenant(req)

	if !e.allowed(tenantId) {
		err := fmt.Errorf("tenant %s not allowed", tenantId)
		resp.WriteHeaderAndEntity(http.StatusForbidden, httperrors.NewHTTPError(http.StatusForbidden, err))
		return
	}
	chain.ProcessFilter(req, resp)
}

// allowed checks if the given tenant is allowed (case insensitive)
func (e *TenantEnsurer) allowed(tenant string) bool {
	return e.allowedTenants[strings.ToLower(tenant)]
}
