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

	restful "github.com/emicklei/go-restful"
	"go.uber.org/zap"
)

// emptyBody is useful because with go-restful you cannot define an insert / update endpoint
// without specifying a payload for reading. it would immediately intercept the request and
// return 406: Not Acceptable to the client.
type emptyBody struct{}

type webResource struct {
	ds *datastore.RethinkStore
}

func sendError(log *zap.Logger, rsp *restful.Response, opname string, errRsp *httperrors.HTTPErrorResponse) {
	s := stack.Caller(1)
	log.Error("service error", zap.String("operation", opname), zap.Int("status", errRsp.StatusCode), zap.String("error", errRsp.Message), zap.Stringer("service-caller", s))
	response, merr := json.Marshal(errRsp)
	log.Info("response", zap.String("resp", string(response)))
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
			sendError(log, rsp, opname, httperrors.NotFound(err))
			return true
		}
		if metal.IsConflict(err) {
			sendError(log, rsp, opname, httperrors.Conflict(err))
			return true
		}
		if metal.IsInternal(err) {
			sendError(log, rsp, opname, httperrors.InternalServerError(err))
			return true
		}
		sendError(log, rsp, opname, httperrors.NewHTTPError(http.StatusUnprocessableEntity, err))
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
