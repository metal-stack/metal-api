package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"runtime"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"github.com/go-stack/stack"

	restful "github.com/emicklei/go-restful"
	"go.uber.org/zap"
)

type webResource struct {
	*zap.SugaredLogger
	log *zap.Logger
	ds  *datastore.RethinkStore
}

func sendError(log *zap.Logger, rsp *restful.Response, opname string, status int, err error) {
	s := stack.Caller(1)
	log.Error("service error", zap.String("operation", opname), zap.Int("status", status), zap.String("error", err.Error()), zap.Stringer("service-caller", s))
	er := metal.ErrorResponse{
		StatusCode: status,
		Message:    err.Error(),
		Operation:  opname,
	}
	response, merr := json.Marshal(er)
	log.Info("response", zap.String("resp", string(response)))
	if merr != nil {
		rsp.WriteError(status, fmt.Errorf("unable to format error string: %v", err))
		return
	}
	rsp.WriteErrorString(status, string(response))
}

func checkError(log *zap.Logger, rsp *restful.Response, opname string, err error) bool {
	if err != nil {
		if metal.IsNotFound(err) {
			sendError(log, rsp, opname, http.StatusNotFound, err)
			return true
		}
		if metal.IsConflict(err) {
			sendError(log, rsp, opname, http.StatusConflict, err)
			return true
		}
		sendError(log, rsp, opname, http.StatusBadRequest, err)
		return true
	}
	return false
}

type entityGetter func(id string) (interface{}, error)

func (wr *webResource) handleReflectResponse(opname string, response *restful.Response, res []reflect.Value) {
	data := res[0].Interface()
	var err error
	if !res[1].IsNil() {
		err = res[1].Elem().Interface().(error)
	}
	if checkError(wr.log, response, opname, err) {
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
		wr.handleReflectResponse(opname, response, res)
	}
}

func (wr *webResource) restListGet(h interface{}) restful.RouteFunction {
	f := reflect.ValueOf(h)
	opname := runtime.FuncForPC(f.Pointer()).Name()
	return func(request *restful.Request, response *restful.Response) {
		res := f.Call(nil)
		wr.handleReflectResponse(opname, response, res)
	}
}
