package health

import (
	"net/http"

	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/inconshreveable/log15"
)

type HealthCheck func() error

type healtstatus struct {
	Message string `json:"message"`
}

func New(log log15.Logger, h HealthCheck) *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path("/health").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"health"}

	ws.Route(ws.GET("/").To(check(log, h)).
		Doc("perform a healtcheck").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(http.StatusOK, "OK", nil).
		Returns(http.StatusInternalServerError, "Unhealthy", nil))
	return ws
}

func check(log log15.Logger, h HealthCheck) func(request *restful.Request, response *restful.Response) {
	return func(request *restful.Request, response *restful.Response) {
		e := h()
		if e != nil {
			s := healtstatus{Message: e.Error()}
			log.Error("unhealthy", "error", e)
			response.WriteHeaderAndEntity(http.StatusInternalServerError, s)
		} else {
			s := healtstatus{Message: "OK"}
			response.WriteEntity(s)
		}
	}
}
