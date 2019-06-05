package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"runtime"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/utils"
	"git.f-i-ts.de/cloud-native/metallib/httperrors"
	"github.com/go-stack/stack"

	"github.com/metal-pod/security"
	restful "github.com/emicklei/go-restful"
	"go.uber.org/zap"
)

// Some predefined users
var (
	Viewer = security.User{
		EMail:  "metal-view@fi-ts.io",
		Name:   "Metal-View",
		Groups: []security.RessourceAccess{metal.ViewAccess},
	}
	Editor = security.User{
		EMail:  "metal-edit@fi-ts.io",
		Name:   "Metal-Edit",
		Groups: []security.RessourceAccess{metal.ViewAccess, metal.EditAccess},
	}
	Admin = security.User{
		EMail:  "metal-admin@fi-ts.io",
		Name:   "Metal-Admin",
		Groups: []security.RessourceAccess{metal.ViewAccess, metal.EditAccess, metal.AdminAccess},
	}
	MetalUsers = map[string]security.User{
		"view":  Viewer,
		"edit":  Editor,
		"admin": Admin,
	}
)

// emptyBody is useful because with go-restful you cannot define an insert / update endpoint
// without specifying a payload for reading. it would immediately intercept the request and
// return 406: Not Acceptable to the client.
type emptyBody struct{}

type webResource struct {
	ds *datastore.RethinkStore
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

func (wr *webResource) restEntityGet(h interface{}) restful.RouteFunction {
	f := reflect.ValueOf(h)
	opname := runtime.FuncForPC(f.Pointer()).Name()
	return func(request *restful.Request, response *restful.Response) {
		id := request.PathParameter("id")
		par := reflect.ValueOf(id)
		res := f.Call([]reflect.Value{par})
		wr.handleReflectResponse(opname, request, response, res)
	}
}

func (wr *webResource) restListGet(h interface{}) restful.RouteFunction {
	f := reflect.ValueOf(h)
	opname := runtime.FuncForPC(f.Pointer()).Name()
	return func(request *restful.Request, response *restful.Response) {
		res := f.Call(nil)
		wr.handleReflectResponse(opname, request, response, res)
	}
}

func viewer(rf restful.RouteFunction) restful.RouteFunction {
	return oneOf(rf, metal.ViewAccess)
}

func editor(rf restful.RouteFunction) restful.RouteFunction {
	return oneOf(rf, metal.EditAccess)
}

func admin(rf restful.RouteFunction) restful.RouteFunction {
	return oneOf(rf, metal.AdminAccess)
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
