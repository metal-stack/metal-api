package utils

import (
	"runtime"
	"strings"

	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/emicklei/go-restful"
	"go.uber.org/zap"
)

// Logger returns the request logger from the request.
func Logger(rq *restful.Request) *zap.Logger {
	return zapup.RequestLogger(rq.Request)
}

// CurrentFuncName returns the name of the caller of this function.
func CurrentFuncName() string {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return "unknown"
	}
	ffpc := runtime.FuncForPC(pc)
	if ffpc == nil {
		return "unknown"
	}
	pp := strings.Split(ffpc.Name(), ".")
	return pp[len(pp)-1]
}
