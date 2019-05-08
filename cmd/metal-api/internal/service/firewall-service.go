package service

import (
	"fmt"
	"net/http"

	"git.f-i-ts.de/cloud-native/metallib/httperrors"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/ipam"
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
		Writes(v1.MachineDetailResponse{}).
		Returns(http.StatusOK, "OK", v1.MachineDetailResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/").
		To(r.listFirewalls).
		Operation("listFirewalls").
		Doc("get all known firewalls").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]v1.MachineListResponse{}).
		Returns(http.StatusOK, "OK", []v1.MachineListResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/allocate").
		To(r.allocateFirewall).
		Doc("allocate a firewall").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.FirewallCreateRequest{}).
		Returns(http.StatusOK, "OK", v1.MachineDetailResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r firewallResource) findFirewall(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	fw, err := r.ds.FindMachine(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	// TODO: Check if fw, otherwise return not found

	response.WriteHeaderAndEntity(http.StatusOK, makeMachineDetailResponse(fw, r.ds, utils.Logger(request).Sugar()))
}

func (r firewallResource) listFirewalls(request *restful.Request, response *restful.Response) {
	fws, err := r.ds.ListMachines()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	// FIXME filter firewalls from machines in these responses.

	response.WriteHeaderAndEntity(http.StatusOK, makeMachineListResponse(fws, r.ds, utils.Logger(request).Sugar()))
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

	spec := machineAllocationSpec{
		UUID:        uuid,
		Name:        name,
		Description: description,
		Tenant:      requestPayload.Tenant,
		Hostname:    hostname,
		ProjectID:   requestPayload.ProjectID,
		PartitionID: requestPayload.PartitionID,
		SizeID:      requestPayload.SizeID,
		ImageID:     requestPayload.ImageID,
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
