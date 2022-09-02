package service

import (
	"fmt"
	"net/http"
	"time"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"

	"github.com/emicklei/go-restful/v3"
	"go.uber.org/zap"

	mdm "github.com/metal-stack/masterdata-api/pkg/client"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/headscale"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-lib/httperrors"
)

type vpnResource struct {
	webResource
	mdc             mdm.Client
	headscaleClient *headscale.HeadscaleClient
}

// NewVPN returns a webservice for VPN specific endpoints.
func NewVPN(
	log *zap.SugaredLogger,
	mdc mdm.Client,
	headscaleClient *headscale.HeadscaleClient,
) *restful.WebService {
	r := vpnResource{
		webResource: webResource{
			log: log,
		},
		mdc:             mdc,
		headscaleClient: headscaleClient,
	}

	return r.webService()
}

// webService creates the webservice endpoint
func (r *vpnResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path(BasePath + "v1/vpn").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"vpn"}

	ws.Route(ws.POST("/authkey/{pid}").
		To(admin(r.getVPNAuthKey)).
		Operation("getVPNAuthKey").
		Doc("create auth key to connect to project's VPN").
		Param(ws.PathParameter("pid", "identifier of the project").DataType("string")).
		Reads(v1.VPNRequest{}).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.VPNResponse{}).
		Returns(http.StatusOK, "OK", v1.VPNResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r *vpnResource) getVPNAuthKey(request *restful.Request, response *restful.Response) {
	var requestPayload v1.VPNRequest
	if err := request.ReadEntity(&requestPayload); err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	pid := requestPayload.Pid
	if ok := r.headscaleClient.NamespaceExists(pid); !ok {
		r.sendError(
			request, response,
			httperrors.NotFound(fmt.Errorf("vpn namespace doesn't exist for project with ID %s", pid)),
		)
		return
	}

	expiration := time.Now().Add(90 * 24 * time.Hour)
	key, err := r.headscaleClient.CreatePreAuthKey(pid, expiration)
	if err != nil {
		r.sendError(
			request, response,
			httperrors.InternalServerError(fmt.Errorf("failed to create new auth key: %w", err)),
		)
		return
	}

	authKeyResp := v1.VPNResponse{
		Address: r.headscaleClient.GetControlPlaneAddress(),
		AuthKey: key,
	}

	r.send(request, response, http.StatusOK, authKeyResp)
}
