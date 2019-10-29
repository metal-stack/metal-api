package service

import (
	"fmt"
	"net/http"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/ipam"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/tags"
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
		Path(BasePath + "v1/ip").
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

	ws.Route(ws.POST("/find").
		To(viewer(ir.findIPs)).
		Operation("findIPs").
		Doc("get all ips that match given properties").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.IPFindRequest{}).
		Writes([]v1.IPResponse{}).
		Returns(http.StatusOK, "OK", []v1.IPResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/free/{id}").
		To(editor(ir.freeIP)).
		Operation("freeIP").
		Doc("frees an ip").
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

	ws.Route(ws.POST("/tag").
		To(editor(ir.tagIP)).
		Operation("tagIP").
		Doc("updates an ip and marks it as used by a cluster or machine.").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.IPTagRequest{}).
		Writes(v1.IPResponse{}).
		Returns(http.StatusOK, "OK", v1.IPResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/untag").
		To(editor(ir.untagIP)).
		Operation("untagIP").
		Doc("updates an ip and marks it as unused by a cluster or machine.").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.IPUntagRequest{}).
		Writes(v1.IPResponse{}).
		Returns(http.StatusOK, "OK", v1.IPResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/allocate").
		To(editor(ir.allocateIP)).
		Operation("allocateIP").
		Doc("allocate an ip in the given network.").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.IPAllocateRequest{}).
		Writes(v1.IPResponse{}).
		Returns(http.StatusCreated, "Created", v1.IPResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/allocate/{ip}").
		To(editor(ir.allocateIP)).
		Operation("allocateSpecificIP").
		Param(ws.PathParameter("ip", "ip to try to allocate").DataType("string")).
		Doc("allocate a specific ip in the given network.").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.IPAllocateRequest{}).
		Writes(v1.IPResponse{}).
		Returns(http.StatusCreated, "Created", v1.IPResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (ir ipResource) findIP(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	ip, err := ir.ds.FindIPByID(id)
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

func (ir ipResource) findIPs(request *restful.Request, response *restful.Response) {
	var requestPayload datastore.IPSearchQuery
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	var ips metal.IPs
	err = ir.ds.SearchIPs(&requestPayload, &ips)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	result := []*v1.IPResponse{}
	for i := range ips {
		result = append(result, v1.NewIPResponse(&ips[i]))
	}

	response.WriteHeaderAndEntity(http.StatusOK, result)
}

func (ir ipResource) freeIP(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	ip, err := ir.ds.FindIPByID(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = validateIPDelete(ip)
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

func validateIPDelete(ip *metal.IP) error {
	s := ip.GetScope()
	if s != metal.ScopeProject {
		return fmt.Errorf("ip with scope %s can not be deleted", ip.GetScope())
	}
	return nil
}

// Checks whether an ip update is allowed:
// (1) allow update of ephemeral to static
// (2) allow update within a scope
// (3) allow update from and to scope project
// (4) deny all other updates
func validateIPUpdate(old *metal.IP, new *metal.IP) error {
	// constraint 1
	if old.Type == metal.Static && new.Type == metal.Ephemeral {
		return fmt.Errorf("cannot change type of ip address from static to ephemeral")
	}
	os := old.GetScope()
	ns := new.GetScope()
	// constraint 2
	if os == ns {
		return nil
	}
	// constraint 3
	if os == metal.ScopeProject || ns == metal.ScopeProject {
		return nil
	}
	if os == metal.ScopeMachine && ns == metal.ScopeCluster {
		return fmt.Errorf("can not use machine ip for a cluster")
	}
	if os == metal.ScopeCluster && ns == metal.ScopeMachine {
		return fmt.Errorf("can not use cluster ip for a machine")
	}
	return fmt.Errorf("can not use ip of scope %v with scope %v", os, ns)
}

func processTags(ts []string) ([]string, error) {
	t := tags.New(ts)
	machineScope := t.HasPrefix(metal.TagIPMachineID)
	clusterScope := t.HasPrefix(metal.TagIPClusterID)
	if clusterScope && machineScope {
		return nil, fmt.Errorf("tags must not contain multiple scopes")
	}
	if machineScope && len(t.Values(metal.TagIPMachineID)) > 1 {
		t.Remove(metal.TagIPMachineID)
	}
	if clusterScope && len(t.Values(metal.TagIPClusterID)) > 1 {
		t.Remove(metal.TagIPClusterID)
	}
	return t.Unique(), nil
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
	if requestPayload.ProjectID == "" {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("projectid should not be empty")) {
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

	nw, err := ir.ds.FindNetworkByID(requestPayload.NetworkID)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	p, err := ir.ds.FindProjectByID(requestPayload.ProjectID)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	tags := requestPayload.Tags
	if requestPayload.MachineID != nil {
		tags = append(tags, metal.IpTag(metal.TagIPMachineID, *requestPayload.MachineID))
	}

	if requestPayload.ClusterID != nil {
		tags = append(tags, metal.IpTag(metal.TagIPClusterID, *requestPayload.ClusterID))
	}

	tags, err = processTags(tags)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	// TODO: Following operations should span a database transaction if possible

	ipAddress, ipParentCidr, err := allocateIP(nw, specificIP, ir.ipamer)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	utils.Logger(request).Sugar().Debugw("found an ip to allocate", "ip", ipAddress, "network", nw.ID)

	ipType := metal.Ephemeral
	if requestPayload.Type == metal.Static {
		ipType = metal.Static
	}

	ip := &metal.IP{
		IPAddress:        ipAddress,
		ParentPrefixCidr: ipParentCidr,
		Name:             name,
		Description:      description,
		NetworkID:        nw.ID,
		ProjectID:        p.ID,
		Type:             ipType,
		Tags:             tags,
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

	oldIP, err := ir.ds.FindIPByID(requestPayload.IPAddress)
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

	ipType := metal.Ephemeral
	if requestPayload.Type == metal.Static {
		ipType = metal.Static
	}
	newIP.Type = ipType

	err = ir.validateAndUpateIP(oldIP, &newIP)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, v1.NewIPResponse(&newIP))
}

func (ir ipResource) validateIPUseOrRelease(ip string, clusterID, machineID *string, tags []string) (*metal.IP, error) {
	if clusterID == nil && machineID == nil {
		return nil, fmt.Errorf("need to specify either clusterId or machineId for the ip")
	}
	if machineID != nil {
		_, err := ir.ds.FindMachineByID(*machineID)
		if err != nil {
			return nil, fmt.Errorf("could not find machine with id %v", *machineID)
		}
	}
	oldIP, err := ir.ds.FindIPByID(ip)
	if err != nil {
		return nil, err
	}
	if containsClusterOrMachineTags(tags) {
		return nil, fmt.Errorf("tags specified in request must not contain internal tags like %v", []string{metal.TagIPClusterID, metal.TagIPMachineID})
	}
	return oldIP, nil
}

func (ir ipResource) validateAndUpateIP(oldIP, newIP *metal.IP) error {
	err := validateIPUpdate(oldIP, newIP)
	if err != nil {
		return err
	}
	tags, err := processTags(newIP.Tags)
	if err != nil {
		return err
	}
	newIP.Tags = tags

	err = ir.ds.UpdateIP(oldIP, newIP)
	if err != nil {
		return err
	}
	return nil
}

func (ir ipResource) tagIP(request *restful.Request, response *restful.Response) {
	var requestPayload v1.IPTagRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	oldIP, err := ir.validateIPUseOrRelease(requestPayload.IPAddress, requestPayload.ClusterID, requestPayload.MachineID, requestPayload.Tags)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	newIP := *oldIP
	tags := oldIP.Tags
	var t string
	if requestPayload.ClusterID != nil {
		t = metal.IpTag(metal.TagIPClusterID, *requestPayload.ClusterID)
	}
	if requestPayload.MachineID != nil {
		t = metal.IpTag(metal.TagIPClusterID, *requestPayload.MachineID)
	}
	tags = append(tags, t)
	tags = append(tags, requestPayload.Tags...)
	newIP.Tags = tags

	err = ir.validateAndUpateIP(oldIP, &newIP)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, v1.NewIPResponse(&newIP))
}

func (ir ipResource) untagIP(request *restful.Request, response *restful.Response) {
	var requestPayload v1.IPUntagRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	oldIP, err := ir.validateIPUseOrRelease(requestPayload.IPAddress, requestPayload.ClusterID, requestPayload.MachineID, requestPayload.Tags)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	var ct string
	if requestPayload.ClusterID != nil {
		ct = metal.IpTag(metal.TagIPClusterID, *requestPayload.ClusterID)
	}
	if requestPayload.MachineID != nil {
		ct = metal.IpTag(metal.TagIPMachineID, *requestPayload.MachineID)
	}
	tagsToRemove := map[string]interface{}{ct: nil}
	for _, t := range requestPayload.Tags {
		tagsToRemove[t] = nil
	}

	newTags := []string{}
	for _, t := range oldIP.Tags {
		if _, ok := tagsToRemove[t]; ok {
			continue
		}
		newTags = append(newTags, t)
	}

	newIP := *oldIP
	newIP.Tags = newTags
	err = ir.validateAndUpateIP(oldIP, &newIP)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, v1.NewIPResponse(&newIP))
}

func containsClusterOrMachineTags(inTags []string) bool {
	t := tags.New(inTags)
	if t.HasPrefix(metal.TagIPClusterID) || t.HasPrefix(metal.TagIPMachineID) {
		return true
	}
	return false
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
