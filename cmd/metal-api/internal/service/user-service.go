package service

import (
	"net/http"

	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/utils"
	"github.com/metal-stack/security"

	"go.uber.org/zap"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/metal-lib/zapup"
)

type userResource struct {
	userGetter security.UserGetter
}

// NewUser returns a webservice for user specific endpoints.
func NewUser(userGetter security.UserGetter) *restful.WebService {
	r := userResource{
		userGetter: userGetter,
	}
	return r.webService()
}

func (r userResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path(BasePath + "v1/user").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"user"}

	ws.Route(ws.GET("/{token}").
		To(viewer(r.getUser)).
		Operation("getUser").
		Doc("extract and validate user from token").
		Param(ws.PathParameter("token", "jwt token with user information").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.User{}).
		Returns(http.StatusOK, "OK", v1.User{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r userResource) getUser(request *restful.Request, response *restful.Response) {
	token := request.PathParameter("token")

	u, err := r.userGetter.UserFromToken(token)
	if checkError(request, response, utils.CurrentFuncName(), err) {
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
	err = response.WriteHeaderAndEntity(http.StatusOK, user)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}
