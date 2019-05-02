package service

import (
	"fmt"
	"net/http"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/ipam"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/utils"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/service/v1"

	"git.f-i-ts.de/cloud-native/metallib/httperrors"
	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
)

type ipResource struct {
	webResource
	ipamer ipam.IPAMer
}

// NewIP returns a webservice for ip specific endpoints.
func NewIP(ds *datastore.RethinkStore, ipamer ipam.IPAMer) *restful.WebService {
	ir := ipResource{
		webResource: webResource{
			ds: ds,
		},
		ipamer: ipamer,
	}
	return ir.webService()
}

func (ir ipResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path("/v1/ip").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"ip"}

	ws.Route(ws.GET("/{id}").
		To(ir.findIP).
		Operation("findIP").
		Doc("get ip by id").
		Param(ws.PathParameter("id", "identifier of the ip").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.IPDetailResponse{}).
		Returns(http.StatusOK, "OK", v1.IPDetailResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/").
		To(ir.listIPs).
		Operation("listIPs").
		Doc("get all ips").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]v1.IPListResponse{}).
		Returns(http.StatusOK, "OK", []v1.IPListResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.DELETE("/{id}").
		To(ir.deleteIP).
		Operation("deleteIP").
		Doc("deletes an ip and returns the deleted entity").
		Param(ws.PathParameter("id", "identifier of the ip").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.IPDetailResponse{}).
		Returns(http.StatusOK, "OK", v1.IPDetailResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/").To(ir.updateIP).
		Doc("updates an ip. if the ip was changed since this one was read, a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.IPUpdateRequest{}).
		Writes(v1.IPDetailResponse{}).
		Returns(http.StatusOK, "OK", v1.IPDetailResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/allocate").To(ir.allocateIP).
		Doc("allocate an ip in the given network for a project.").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.IPAllocateRequest{}).
		Writes(v1.IPDetailResponse{}).
		Returns(http.StatusCreated, "Created", v1.IPDetailResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (ir ipResource) findIP(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	ip, err := ir.ds.FindIP(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, v1.NewIPDetailResponse(ip))
}

func (ir ipResource) listIPs(request *restful.Request, response *restful.Response) {
	ips, err := ir.ds.ListIPs()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	var result []*v1.IPListResponse
	for _, ip := range ips {
		result = append(result, v1.NewIPListResponse(&ip))
	}

	response.WriteHeaderAndEntity(http.StatusOK, result)
}

func (ir ipResource) deleteIP(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	ip, err := ir.ds.FindIP(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = ir.ipamer.ReleaseIP(*ip)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = ir.ds.DeleteIP(ip)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, v1.NewIPDetailResponse(ip))
}

func (ir ipResource) allocateIP(request *restful.Request, response *restful.Response) {
	var requestPayload v1.IPAllocateRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	nw, err := ir.ds.FindNetwork(requestPayload.NetworkID)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	// TODO: Check if project exists if we get a project entity
	// TODO: Following operations should span a database transaction if possible

	ip, err := allocateIP(*nw, ir.ipamer)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	utils.Logger(request).Sugar().Debugw("found an ip to allocate", "ip", ip.IPAddress, "network", nw.ID)

	ip.Name = requestPayload.Name
	ip.Description = requestPayload.Description
	ip.ProjectID = requestPayload.ProjectID
	ip.NetworkID = nw.ID

	err = ir.ds.CreateIP(ip)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusCreated, v1.NewIPDetailResponse(ip))
}

func (ir ipResource) updateIP(request *restful.Request, response *restful.Response) {
	var requestPayload v1.IPUpdateRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	oldIP, err := ir.ds.FindIP(requestPayload.IPAddress)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	newIP := *oldIP

	if requestPayload.Name != "" {
		newIP.Name = requestPayload.Name
	}
	if requestPayload.Description != "" {
		newIP.Description = requestPayload.Description
	}

	err = ir.ds.UpdateIP(oldIP, &newIP)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, v1.NewIPDetailResponse(&newIP))
}

func allocateIP(parent metal.Network, ipamer ipam.IPAMer) (*metal.IP, error) {
	var errors []error
	var err error
	var ipAddress string
	var parentPrefixCidr string
	for _, prefix := range parent.Prefixes {
		ipAddress, err = ipamer.AllocateIP(prefix)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		if ipAddress != "" {
			parentPrefixCidr = prefix.String()
			break
		}
	}
	if ipAddress == "" {
		if len(errors) > 0 {
			return nil, fmt.Errorf("cannot allocate free ip in ipam: %v", errors)
		}
		return nil, fmt.Errorf("cannot allocate free ip in ipam")
	}
	ip := &metal.IP{
		IPAddress:        ipAddress,
		ParentPrefixCidr: parentPrefixCidr,
		NetworkID:        parent.ID,
	}
	return ip, nil
}
