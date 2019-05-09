package service

import (
	"fmt"
	"net/http"

	"git.f-i-ts.de/cloud-native/metallib/httperrors"
	"go.uber.org/zap"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/ipam"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	v1 "git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/service/v1"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/utils"

	"git.f-i-ts.de/cloud-native/metallib/bus"
	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
)

type firewallResource struct {
	webResource
	bus.Publisher
	ipamer ipam.IPAMer
}

// NewFirewall returns a webservice for firewall specific endpoints.
func NewFirewall(
	ds *datastore.RethinkStore,
	ipamer ipam.IPAMer) *restful.WebService {
	r := firewallResource{
		webResource: webResource{
			ds: ds,
		},
		ipamer: ipamer,
	}
	return r.webService()
}

// webService creates the webservice endpoint
func (r firewallResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path("/v1/firewall").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"firewall"}

	ws.Route(ws.GET("/{id}").
		To(r.findFirewall).
		Operation("findFirewall").
		Doc("get firewall by id").
		Param(ws.PathParameter("id", "identifier of the firewall").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.FirewallDetailResponse{}).
		Returns(http.StatusOK, "OK", v1.FirewallDetailResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/").
		To(r.listFirewalls).
		Operation("listFirewalls").
		Doc("get all known firewalls").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]v1.FirewallListResponse{}).
		Returns(http.StatusOK, "OK", []v1.FirewallListResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/allocate").
		To(r.allocateFirewall).
		Doc("allocate a firewall").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.FirewallCreateRequest{}).
		Returns(http.StatusOK, "OK", v1.FirewallDetailResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r firewallResource) findFirewall(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	fw, err := r.ds.FindMachine(id)
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

	response.WriteHeaderAndEntity(http.StatusOK, makeFirewallDetailResponse(fw, r.ds, utils.Logger(request).Sugar()))
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

	var fws []metal.Machine
	for i := range possibleFws {
		if possibleFws[i].IsFirewall(imgs.ByID()) {
			fws = append(fws, possibleFws[i])
		}
	}

	response.WriteHeaderAndEntity(http.StatusOK, makeFirewallListResponse(fws, r.ds, utils.Logger(request).Sugar()))
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
	if requestPayload.NetworkIDs != nil && len(requestPayload.NetworkIDs) <= 0 {
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

	if !image.HasFeature(metal.ImageFeatureMachine) {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("given image is not usable for a firewall, features: %s", image.ImageFeatureString())) {
			return
		}
	}

	spec := machineAllocationSpec{
		UUID:        uuid,
		Name:        name,
		Description: description,
		Tenant:      requestPayload.Tenant,
		Hostname:    hostname,
		ProjectID:   requestPayload.ProjectID,
		PartitionID: requestPayload.PartitionID,
		SizeID:      requestPayload.SizeID,
		Image:       image,
		SSHPubKeys:  requestPayload.SSHPubKeys,
		UserData:    userdata,
		Tags:        requestPayload.Tags,
		NetworkIDs:  requestPayload.NetworkIDs,
		IPs:         []string{}, // for the time being not supported
		HA:          ha,
	}

	m, err := allocateMachine(r.ds, r.ipamer, &spec)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, makeMachineDetailResponse(m, r.ds, utils.Logger(request).Sugar()))
}

func makeFirewallDetailResponse(fw *metal.Machine, ds *datastore.RethinkStore, logger *zap.SugaredLogger) *v1.FirewallDetailResponse {
	return &v1.FirewallDetailResponse{MachineDetailResponse: *makeMachineDetailResponse(fw, ds, logger)}
}

func makeFirewallListResponse(fws []metal.Machine, ds *datastore.RethinkStore, logger *zap.SugaredLogger) []*v1.FirewallListResponse {
	machineListResponse := makeMachineListResponse(fws, ds, logger)

	firewallListResponse := []*v1.FirewallListResponse{}
	for i := range machineListResponse {
		firewallListResponse = append(firewallListResponse, &v1.FirewallListResponse{MachineListResponse: *machineListResponse[i]})
	}

	return firewallListResponse

}
