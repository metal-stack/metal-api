package service

import (
	"fmt"
	"net/http"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/ipam"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/utils"

	v1 "git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/service/v1"

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
		To(viewer(ir.findIP)).
		Operation("findIP").
		Doc("get ip by id").
		Param(ws.PathParameter("id", "identifier of the ip").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.IPResponse{}).
		Returns(http.StatusOK, "OK", v1.IPResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/").
		To(viewer(ir.listIPs)).
		Operation("listIPs").
		Doc("get all ips").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]v1.IPResponse{}).
		Returns(http.StatusOK, "OK", []v1.IPResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.DELETE("/{id}").
		To(editor(ir.deleteIP)).
		Operation("deleteIP").
		Doc("deletes an ip and returns the deleted entity").
		Param(ws.PathParameter("id", "identifier of the ip").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.IPResponse{}).
		Returns(http.StatusOK, "OK", v1.IPResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/").
		To(editor(ir.updateIP)).
		Operation("updateIP").
		Doc("updates an ip. if the ip was changed since this one was read, a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.IPUpdateRequest{}).
		Writes(v1.IPResponse{}).
		Returns(http.StatusOK, "OK", v1.IPResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/allocate").
		To(editor(ir.allocateIP)).
		Operation("allocateIP").
		Doc("allocate an ip in the given network for a project.").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.IPAllocateRequest{}).
		Writes(v1.IPResponse{}).
		Returns(http.StatusCreated, "Created", v1.IPResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/allocate/{ip}").
		To(editor(ir.allocateIP)).
		Operation("allocateSpecificIP").
		Param(ws.PathParameter("ip", "ip to try to allocate").DataType("string")).
		Doc("allocate an specific ip in the given network for a project.").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.IPAllocateRequest{}).
		Writes(v1.IPResponse{}).
		Returns(http.StatusCreated, "Created", v1.IPResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (ir ipResource) findIP(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	ip, err := ir.ds.FindIP(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, v1.NewIPResponse(ip))
}

func (ir ipResource) listIPs(request *restful.Request, response *restful.Response) {
	ips, err := ir.ds.ListIPs()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	result := []*v1.IPResponse{}
	for i := range ips {
		result = append(result, v1.NewIPResponse(&ips[i]))
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

	response.WriteHeaderAndEntity(http.StatusOK, v1.NewIPResponse(ip))
}

func (ir ipResource) allocateIP(request *restful.Request, response *restful.Response) {
	specificIP := request.PathParameter("ip")
	var requestPayload v1.IPAllocateRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	if requestPayload.NetworkID == "" {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("networkid should not be empty")) {
			return
		}
	}

	var name string
	if requestPayload.Name != nil {
		name = *requestPayload.Name
	}
	var description string
	if requestPayload.Description != nil {
		description = *requestPayload.Description
	}

	nw, err := ir.ds.FindNetwork(requestPayload.NetworkID)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	// TODO: Check if project exists if we get a project entity
	// TODO: Following operations should span a database transaction if possible

	ipAddress, ipParentCidr, err := allocateIP(nw, specificIP, ir.ipamer)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	utils.Logger(request).Sugar().Debugw("found an ip to allocate", "ip", ipAddress, "network", nw.ID)

	ip := &metal.IP{
		IPAddress:        ipAddress,
		ParentPrefixCidr: ipParentCidr,
		Name:             name,
		Description:      description,
		NetworkID:        nw.ID,
		ProjectID:        requestPayload.ProjectID,
	}

	err = ir.ds.CreateIP(ip)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusCreated, v1.NewIPResponse(ip))
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

	if requestPayload.Name != nil {
		newIP.Name = *requestPayload.Name
	}
	if requestPayload.Description != nil {
		newIP.Description = *requestPayload.Description
	}

	err = ir.ds.UpdateIP(oldIP, &newIP)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, v1.NewIPResponse(&newIP))
}

func allocateIP(parent *metal.Network, specificIP string, ipamer ipam.IPAMer) (string, string, error) {
	var errors []error
	var err error
	var ipAddress string
	var parentPrefixCidr string
	for _, prefix := range parent.Prefixes {
		if specificIP == "" {
			ipAddress, err = ipamer.AllocateIP(prefix)
		} else {
			ipAddress, err = ipamer.AllocateSpecificIP(prefix, specificIP)
		}
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
			return "", "", fmt.Errorf("cannot allocate free ip in ipam: %v", errors)
		}
		return "", "", fmt.Errorf("cannot allocate free ip in ipam")
	}

	return ipAddress, parentPrefixCidr, nil
}
