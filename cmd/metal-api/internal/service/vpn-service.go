package service

import (
	"context"
	"fmt"
	"github.com/emicklei/go-restful/v3"
	headscalev1 "github.com/juanfont/headscale/gen/go/headscale/v1"
	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	mdm "github.com/metal-stack/masterdata-api/pkg/client"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/headscale"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-lib/httperrors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
	"net/http"
	"time"
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

	ws.Route(ws.GET("/authkey/{pid}").
		To(admin(r.getVPNAuthKey)).
		Operation("getVPNAuthKey").
		Doc("create auth key to connect to Project's VPN").
		Param(ws.PathParameter("pid", "identifier of the Project").DataType("string")).
		Writes(v1.VPNResponse{}).
		Returns(http.StatusOK, "OK", v1.VPNResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r *vpnResource) getVPNAuthKey(request *restful.Request, response *restful.Response) {
	ctx := context.Background()
	pid := request.PathParameter("pid")

	p, err := r.mdc.Project().Get(ctx, &mdmv1.ProjectGetRequest{Id: pid})
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}
	if p.GetProject() == nil {
		r.sendError(
			request, response,
			httperrors.NotFound(fmt.Errorf("Project with ID %s is not found", pid)),
		)
		return
	}

	getNSRequest := &headscalev1.GetNamespaceRequest{
		Name: p.Project.Name,
	}
	if _, err = r.headscaleClient.GetNamespace(ctx, getNSRequest); err != nil {
		r.sendError(
			request, response,
			httperrors.NotFound(fmt.Errorf("VPN namespace doesn't exist for Project with ID %s", pid)),
		)
		return
	}

	expiration := time.Now().Add(90 * 24 * time.Hour)
	createPAKRequest := &headscalev1.CreatePreAuthKeyRequest{
		Namespace:  p.Project.Name,
		Expiration: timestamppb.New(expiration),
	}
	resp, err := r.headscaleClient.CreatePreAuthKey(ctx, createPAKRequest)
	if err != nil || resp == nil || resp.PreAuthKey == nil {
		r.sendError(
			request, response,
			httperrors.InternalServerError(fmt.Errorf("failed to create new Auth Key: %w", err)),
		)
		return
	}

	authKeyResp := v1.VPNResponse{
		AuthKey: resp.PreAuthKey.Key,
	}

	r.send(request, response, http.StatusOK, authKeyResp)
}
