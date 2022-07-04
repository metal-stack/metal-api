package service

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/metal-stack/security"

	"github.com/metal-stack/metal-lib/httperrors"
	"go.uber.org/zap"

	mdm "github.com/metal-stack/masterdata-api/pkg/client"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/ipam"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-lib/bus"
)

type firewallResource struct {
	webResource
	bus.Publisher
	ipamer     ipam.IPAMer
	mdc        mdm.Client
	userGetter security.UserGetter
	actor      *asyncActor
}

// NewFirewall returns a webservice for firewall specific endpoints.
func NewFirewall(
	log *zap.SugaredLogger,
	ds *datastore.RethinkStore,
	pub bus.Publisher,
	ipamer ipam.IPAMer,
	ep *bus.Endpoints,
	mdc mdm.Client,
	userGetter security.UserGetter,
) (*restful.WebService, error) {
	r := firewallResource{
		webResource: webResource{
			log: log,
			ds:  ds,
		},
		Publisher:  pub,
		ipamer:     ipamer,
		mdc:        mdc,
		userGetter: userGetter,
	}

	var err error
	r.actor, err = newAsyncActor(log, ep, ds, ipamer)
	if err != nil {
		return nil, fmt.Errorf("cannot create async actor: %w", err)
	}

	return r.webService(), nil
}

// webService creates the webservice endpoint
func (r *firewallResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path(BasePath + "v1/firewall").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"firewall"}

	ws.Route(ws.GET("/{id}").
		To(viewer(r.findFirewall)).
		Operation("findFirewall").
		Doc("get firewall by id").
		Param(ws.PathParameter("id", "identifier of the firewall").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.FirewallResponse{}).
		Returns(http.StatusOK, "OK", v1.FirewallResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/find").
		To(viewer(r.findFirewalls)).
		Operation("findFirewalls").
		Doc("find firewalls by multiple criteria").
		Reads(v1.FirewallFindRequest{}).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]v1.FirewallResponse{}).
		Returns(http.StatusOK, "OK", []v1.FirewallResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/").
		To(viewer(r.listFirewalls)).
		Operation("listFirewalls").
		Doc("get all known firewalls").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]v1.FirewallResponse{}).
		Returns(http.StatusOK, "OK", []v1.FirewallResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/allocate").
		To(editor(r.allocateFirewall)).
		Operation("allocateFirewall").
		Doc("allocate a firewall").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.FirewallCreateRequest{}).
		Returns(http.StatusOK, "OK", v1.FirewallResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r *firewallResource) findFirewall(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	fw, err := r.ds.FindMachineByID(id)
	if err != nil {
		r.SendError(response, DefaultError(err))
		return
	}

	if !fw.IsFirewall() {
		r.SendError(response, httperrors.NewHTTPError(http.StatusNotFound, httperrors.NotFound(errors.New("machine is not a firewall"))))
		return
	}

	resp, err := makeFirewallResponse(fw, r.ds)
	if err != nil {
		r.SendError(response, DefaultError(err))
		return
	}

	r.Send(response, http.StatusOK, resp)
}

func (r *firewallResource) findFirewalls(request *restful.Request, response *restful.Response) {
	var requestPayload datastore.MachineSearchQuery
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.SendError(response, httperrors.BadRequest(err))
		return
	}

	requestPayload.AllocationRole = &metal.RoleFirewall

	var fws metal.Machines
	err = r.ds.SearchMachines(&requestPayload, &fws)
	if err != nil {
		r.SendError(response, DefaultError(err))
		return
	}

	resp, err := makeFirewallResponseList(fws, r.ds)
	if err != nil {
		r.SendError(response, DefaultError(err))
		return
	}

	r.Send(response, http.StatusOK, resp)
}

func (r *firewallResource) listFirewalls(request *restful.Request, response *restful.Response) {
	var fws metal.Machines
	err := r.ds.SearchMachines(&datastore.MachineSearchQuery{
		AllocationRole: &metal.RoleFirewall,
	}, &fws)
	if err != nil {
		r.SendError(response, DefaultError(err))
		return
	}

	resp, err := makeFirewallResponseList(fws, r.ds)
	if err != nil {
		r.SendError(response, DefaultError(err))
		return
	}

	r.Send(response, http.StatusOK, resp)
}

func (r *firewallResource) allocateFirewall(request *restful.Request, response *restful.Response) {
	var requestPayload v1.FirewallCreateRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.SendError(response, httperrors.BadRequest(err))
		return
	}

	user, err := r.userGetter.User(request.Request)
	if err != nil {
		r.SendError(response, DefaultError(err))
		return
	}

	spec, err := createMachineAllocationSpec(r.ds, requestPayload.MachineAllocateRequest, metal.RoleFirewall, user)
	if err != nil {
		r.SendError(response, httperrors.BadRequest(err))
		return
	}

	m, err := allocateMachine(r.Logger(request), r.ds, r.ipamer, spec, r.mdc, r.actor, r.Publisher)
	if err != nil {
		r.SendError(response, DefaultError(err))
		return
	}

	resp, err := makeMachineResponse(m, r.ds)
	if err != nil {
		r.SendError(response, DefaultError(err))
		return
	}

	r.Send(response, http.StatusOK, resp)
}

func makeFirewallResponse(fw *metal.Machine, ds *datastore.RethinkStore) (*v1.FirewallResponse, error) {
	ms, err := makeMachineResponse(fw, ds)
	if err != nil {
		return nil, err
	}

	return &v1.FirewallResponse{MachineResponse: *ms}, nil
}

func makeFirewallResponseList(fws metal.Machines, ds *datastore.RethinkStore) ([]*v1.FirewallResponse, error) {
	machineResponseList, err := makeMachineResponseList(fws, ds)
	if err != nil {
		return nil, err
	}

	firewallResponseList := []*v1.FirewallResponse{}
	for i := range machineResponseList {
		firewallResponseList = append(firewallResponseList, &v1.FirewallResponse{MachineResponse: *machineResponseList[i]})
	}

	return firewallResponseList, nil
}
