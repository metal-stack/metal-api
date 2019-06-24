package service

import (
	"fmt"
	"net/http"
	"strings"

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
		To(viewer(r.findFirewall)).
		Operation("findFirewall").
		Doc("get firewall by id").
		Param(ws.PathParameter("id", "identifier of the firewall").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.FirewallResponse{}).
		Returns(http.StatusOK, "OK", v1.FirewallResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/find").
		To(viewer(r.searchFirewall)).
		Operation("searchFirewall").
		Doc("search firewalls").
		Param(ws.QueryParameter("project", "project that this firewall is allocated with").DataType("string")).
		Param(ws.QueryParameter("partition", "the partition in which the firewall is located").DataType("string")).
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

	response.WriteHeaderAndEntity(http.StatusOK, makeFirewallResponse(fw, r.ds, utils.Logger(request).Sugar()))
}

func (r firewallResource) searchFirewall(request *restful.Request, response *restful.Response) {
	partition := strings.TrimSpace(request.QueryParameter("partition"))
	project := strings.TrimSpace(request.QueryParameter("project"))

	possibleFws, err := searchMachine(r.ds, "", partition, project)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	imgs, err := r.ds.ListImages()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	var fws []metal.Machine
	imageMap := imgs.ByID()
	for i := range possibleFws {
		if possibleFws[i].IsFirewall(imageMap) {
			fws = append(fws, possibleFws[i])
		}
	}

	response.WriteHeaderAndEntity(http.StatusOK, makeFirewallResponseList(fws, r.ds, utils.Logger(request).Sugar()))
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
	imageMap := imgs.ByID()
	for i := range possibleFws {
		if possibleFws[i].IsFirewall(imageMap) {
			fws = append(fws, possibleFws[i])
		}
	}

	response.WriteHeaderAndEntity(http.StatusOK, makeFirewallResponseList(fws, r.ds, utils.Logger(request).Sugar()))
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

	if !image.HasFeature(metal.ImageFeatureFirewall) {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("given image is not usable for a firewall, features: %s", image.ImageFeatureString())) {
			return
		}
	}

	hasUnderlay := false
	for _, nid := range requestPayload.NetworkIDs {
		n, err := r.ds.FindNetwork(nid)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
		if n.Underlay {
			hasUnderlay = true
			break
		}
	}
	if !hasUnderlay {
		checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("no underlay network specified"))
		return
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

	response.WriteHeaderAndEntity(http.StatusOK, makeMachineResponse(m, r.ds, utils.Logger(request).Sugar()))
}

func makeFirewallResponse(fw *metal.Machine, ds *datastore.RethinkStore, logger *zap.SugaredLogger) *v1.FirewallResponse {
	return &v1.FirewallResponse{MachineResponse: *makeMachineResponse(fw, ds, logger)}
}

func makeFirewallResponseList(fws []metal.Machine, ds *datastore.RethinkStore, logger *zap.SugaredLogger) []*v1.FirewallResponse {
	machineResponseList := makeMachineResponseList(fws, ds, logger)

	firewallResponseList := []*v1.FirewallResponse{}
	for i := range machineResponseList {
		firewallResponseList = append(firewallResponseList, &v1.FirewallResponse{MachineResponse: *machineResponseList[i]})
	}

	return firewallResponseList

}
