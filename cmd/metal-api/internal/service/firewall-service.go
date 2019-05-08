package service

import (
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
		Writes(v1.FirewallDetailResponse{}).
		Returns(http.StatusOK, "OK", v1.FirewallDetailResponse{}).
		Returns(http.StatusNotFound, "Not Found", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/").
		To(r.listFirewalls).
		Operation("listFirewalls").
		Doc("get all known firewalls").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]v1.FirewallListResponse{}).
		Returns(http.StatusOK, "OK", []v1.FirewallListResponse{}).
		Returns(http.StatusNotFound, "Not Found", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/allocate").
		To(r.allocateFirewall).
		Doc("allocate a firewall").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.FirewallCreateRequest{}).
		Returns(http.StatusOK, "OK", v1.FirewallDetailResponse{}).
		Returns(http.StatusNotFound, "No free firewall for allocation found", httperrors.HTTPErrorResponse{}).
		Returns(http.StatusUnprocessableEntity, "Unprocessable Entity", httperrors.HTTPErrorResponse{}))

	return ws
}

// FIXME filter firewalls from machines in these responses.

func (r firewallResource) findFirewall(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	fw, err := r.ds.FindMachine(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	response.WriteHeaderAndEntity(http.StatusOK, v1.NewFirewallDetailResponse(fw))
}

func (r firewallResource) listFirewalls(request *restful.Request, response *restful.Response) {
	fws, err := r.ds.ListMachines()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	result := []*v1.FirewallListResponse{}
	for _, fw := range fws {
		result = append(result, v1.NewFirewallListResponse(&fw))
	}

	response.WriteHeaderAndEntity(http.StatusOK, result)
}

func (r firewallResource) allocateFirewall(request *restful.Request, response *restful.Response) {
	var requestPayload v1.FirewallCreateRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	// FIXME:
	// allocateRequest := requestPayload.AllocateMachine

	// fw, err := allocateMachine(r.ds, r.ipamer, &allocateRequest)
	// if checkError(request, response, utils.CurrentFuncName(), err) {
	// 	return
	// }
	// response.WriteHeaderAndEntity(http.StatusOK, v1.NewFirewallDetailResponse(fw))
}
