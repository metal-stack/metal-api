package service

import (
	"fmt"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/grpc"
	"net/http"

	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/metal-lib/zapup"
	"go.uber.org/zap"

	mdm "github.com/metal-stack/masterdata-api/pkg/client"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/ipam"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/utils"

	restfulspec "github.com/emicklei/go-restful-openapi"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-lib/bus"
)

type firewallResource struct {
	webResource
	bus.Publisher
	ipamer     ipam.IPAMer
	mdc        mdm.Client
	waitServer *grpc.WaitServer
	actor      *asyncActor
}

// NewFirewall returns a webservice for firewall specific endpoints.
func NewFirewall(
	ds *datastore.RethinkStore,
	ipamer ipam.IPAMer,
	ep *bus.Endpoints,
	mdc mdm.Client,
	waitServer *grpc.WaitServer) (*restful.WebService, error) {
	r := firewallResource{
		webResource: webResource{
			ds: ds,
		},
		ipamer:     ipamer,
		mdc:        mdc,
		waitServer: waitServer,
	}

	var err error
	r.actor, err = newAsyncActor(zapup.MustRootLogger(), ep, ds, ipamer)
	if err != nil {
		return nil, fmt.Errorf("cannot create async actor: %w", err)
	}

	return r.webService(), nil
}

// webService creates the webservice endpoint
func (r firewallResource) webService() *restful.WebService {
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

	ws.Route(ws.GET("/find").
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

func (r firewallResource) findFirewall(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	fw, err := r.ds.FindMachineByID(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	imgs, err := r.ds.ListImages()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	if !fw.IsFirewall(imgs.ByID()) {
		sendError(utils.Logger(request), response, utils.CurrentFuncName(), httperrors.NotFound(fmt.Errorf("machine is not a firewall")))
		return
	}

	err = response.WriteHeaderAndEntity(http.StatusOK, makeFirewallResponse(fw, r.ds, utils.Logger(request).Sugar()))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r firewallResource) findFirewalls(request *restful.Request, response *restful.Response) {
	var requestPayload datastore.MachineSearchQuery
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	var possibleFws metal.Machines
	err = r.ds.SearchMachines(&requestPayload, &possibleFws)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	imgs, err := r.ds.ListImages()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	fws := metal.Machines{}
	imageMap := imgs.ByID()
	for i := range possibleFws {
		if possibleFws[i].IsFirewall(imageMap) {
			fws = append(fws, possibleFws[i])
		}
	}

	err = response.WriteHeaderAndEntity(http.StatusOK, makeFirewallResponseList(fws, r.ds, utils.Logger(request).Sugar()))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r firewallResource) listFirewalls(request *restful.Request, response *restful.Response) {
	possibleFws, err := r.ds.ListMachines()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	// potentially a little unefficient because images are also retrieved for creating the machine list response later
	imgs, err := r.ds.ListImages()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	var fws metal.Machines
	imageMap := imgs.ByID()
	for i := range possibleFws {
		if possibleFws[i].IsFirewall(imageMap) {
			fws = append(fws, possibleFws[i])
		}
	}

	err = response.WriteHeaderAndEntity(http.StatusOK, makeFirewallResponseList(fws, r.ds, utils.Logger(request).Sugar()))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r firewallResource) allocateFirewall(request *restful.Request, response *restful.Response) {
	var requestPayload v1.FirewallCreateRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	var uuid string
	if requestPayload.UUID != nil {
		uuid = *requestPayload.UUID
	}
	var name string
	if requestPayload.Name != nil {
		name = *requestPayload.Name
	}
	var description string
	if requestPayload.Description != nil {
		description = *requestPayload.Description
	}
	hostname := "metal"
	if requestPayload.Hostname != nil {
		hostname = *requestPayload.Hostname
	}
	var userdata string
	if requestPayload.UserData != nil {
		userdata = *requestPayload.UserData
	}
	if requestPayload.Networks != nil && len(requestPayload.Networks) <= 0 {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("network ids cannot be empty")) {
			return
		}
	}
	ha := false
	if requestPayload.HA != nil {
		ha = *requestPayload.HA
	}
	if ha {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("highly-available firewall not supported for the time being")) {
			return
		}
	}

	image, err := r.ds.FindImage(requestPayload.ImageID)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	if !image.HasFeature(metal.ImageFeatureFirewall) {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("given image is not usable for a firewall, features: %s", image.ImageFeatureString())) {
			return
		}
	}

	spec := machineAllocationSpec{
		UUID:        uuid,
		Name:        name,
		Description: description,
		Hostname:    hostname,
		ProjectID:   requestPayload.ProjectID,
		PartitionID: requestPayload.PartitionID,
		SizeID:      requestPayload.SizeID,
		Image:       image,
		SSHPubKeys:  requestPayload.SSHPubKeys,
		UserData:    userdata,
		Tags:        requestPayload.Tags,
		Networks:    requestPayload.Networks,
		IPs:         requestPayload.IPs,
		HA:          ha,
		IsFirewall:  true,
	}

	m, err := allocateMachine(utils.Logger(request).Sugar(), r.ds, r.ipamer, &spec, r.mdc, r.actor, r.waitServer)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, makeMachineResponse(m, r.ds, utils.Logger(request).Sugar()))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func makeFirewallResponse(fw *metal.Machine, ds *datastore.RethinkStore, logger *zap.SugaredLogger) *v1.FirewallResponse {
	return &v1.FirewallResponse{MachineResponse: *makeMachineResponse(fw, ds, logger)}
}

func makeFirewallResponseList(fws metal.Machines, ds *datastore.RethinkStore, logger *zap.SugaredLogger) []*v1.FirewallResponse {
	machineResponseList := makeMachineResponseList(fws, ds, logger)

	firewallResponseList := []*v1.FirewallResponse{}
	for i := range machineResponseList {
		firewallResponseList = append(firewallResponseList, &v1.FirewallResponse{MachineResponse: *machineResponseList[i]})
	}

	return firewallResponseList

}
