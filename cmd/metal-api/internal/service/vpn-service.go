package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"time"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"

	"github.com/emicklei/go-restful/v3"

	headscalev1 "github.com/juanfont/headscale/gen/go/headscale/v1"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/headscale"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/metal-lib/pkg/pointer"
)

type vpnResource struct {
	webResource
	headscaleClient *headscale.HeadscaleClient
}

// NewVPN returns a webservice for VPN specific endpoints.
func NewVPN(
	log *slog.Logger,
	headscaleClient *headscale.HeadscaleClient,
) *restful.WebService {
	r := vpnResource{
		webResource: webResource{
			log: log,
		},
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

	ws.Route(ws.POST("/authkey").
		To(admin(r.getVPNAuthKey)).
		Operation("getVPNAuthKey").
		Doc("create auth key to connect to project's VPN").
		Reads(v1.VPNRequest{}).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.VPNResponse{}).
		Returns(http.StatusOK, "OK", v1.VPNResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r *vpnResource) getVPNAuthKey(request *restful.Request, response *restful.Response) {
	if r.headscaleClient == nil {
		r.sendError(request, response, httperrors.InternalServerError(featureDisabledErr))
		return
	}
	var requestPayload v1.VPNRequest
	if err := request.ReadEntity(&requestPayload); err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	pid := requestPayload.Pid
	if ok := r.headscaleClient.UserExists(request.Request.Context(), pid); !ok {
		r.sendError(
			request, response,
			httperrors.NotFound(fmt.Errorf("vpn user doesn't exist for project with ID %s", pid)),
		)
		return
	}

	expiration := time.Now()
	if requestPayload.Expiration != nil {
		expiration = expiration.Add(*requestPayload.Expiration)
	} else {
		expiration = expiration.Add(time.Hour)
	}
	key, err := r.headscaleClient.CreatePreAuthKey(request.Request.Context(), pid, expiration, requestPayload.Ephemeral)
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

type headscaleMachineLister interface {
	MachinesConnected(ctx context.Context) ([]*headscalev1.Machine, error)
}

func EvaluateVPNConnected(log *slog.Logger, ds *datastore.RethinkStore, lister headscaleMachineLister) error {
	ms, err := ds.ListMachines()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	headscaleMachines, err := lister.MachinesConnected(ctx)
	if err != nil {
		return err
	}

	var errs []error
	for _, m := range ms {
		m := m
		if m.Allocation == nil || m.Allocation.VPN == nil {
			continue
		}

		index := slices.IndexFunc(headscaleMachines, func(hm *headscalev1.Machine) bool {
			if hm.Name != m.ID {
				return false
			}

			if pointer.SafeDeref(hm.User).Name != m.Allocation.Project {
				return false
			}

			return true
		})

		if index < 0 {
			continue
		}

		connected := headscaleMachines[index].Online

		if m.Allocation.VPN.Connected == connected {
			log.Info("not updating vpn because already up-to-date", "machine", m.ID, "connected", connected)
			continue
		}

		old := m
		m.Allocation.VPN.Connected = connected

		err := ds.UpdateMachine(&old, &m)
		if err != nil {
			errs = append(errs, err)
			log.Error("unable to update vpn connected state, continue anyway", "machine", m.ID, "error", err)
			continue
		}

		log.Info("updated vpn connected state", "machine", m.ID, "connected", connected)
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors occurred when evaluating machine vpn connections")
	}

	return nil
}
