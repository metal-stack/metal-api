package service

import (
	"fmt"
	"net/http"

	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/security"
	"go.uber.org/zap"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-lib/httperrors"
)

type userResource struct {
	webResource
	userGetter security.UserGetter
}

// NewUser returns a webservice for user specific endpoints.
func NewUser(log *zap.SugaredLogger, userGetter security.UserGetter) *restful.WebService {
	r := userResource{
		webResource: webResource{
			log: log,
		},
		userGetter: userGetter,
	}
	return r.webService()
}

func (r *userResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path(BasePath + "v1/user").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"user"}

	ws.Route(ws.GET("/me").
		To(viewer(r.getMe)).
		Operation("getMe").
		Doc("extract the connecting user from auth header").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.User{}).
		Returns(http.StatusOK, "OK", v1.User{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r *userResource) getMe(request *restful.Request, response *restful.Response) {
	u, err := r.userGetter.User(request.Request)
	if err != nil {
		r.sendError(request, response, httperrors.NewHTTPError(http.StatusUnprocessableEntity, err))
		return
	}

	if u == nil {
		r.sendError(request, response, httperrors.BadRequest(fmt.Errorf("unable to extract user from token, got nil")))
		return
	}

	grps := []string{}
	for _, g := range u.Groups {
		grps = append(grps, string(g))
	}
	user := &v1.User{
		EMail:   u.EMail,
		Name:    u.Name,
		Tenant:  u.Tenant,
		Issuer:  u.Issuer,
		Subject: u.Subject,
		Groups:  grps,
	}

	r.send(request, response, http.StatusOK, user)
}
