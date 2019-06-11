package service

import (
	"fmt"
	"net/http"
	"strconv"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/utils"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/ipam"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	v1 "git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/service/v1"
	"git.f-i-ts.de/cloud-native/metallib/httperrors"
	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
)

type networkResource struct {
	webResource
	ipamer ipam.IPAMer
}

// NewNetwork returns a webservice for network specific endpoints.
func NewNetwork(ds *datastore.RethinkStore, ipamer ipam.IPAMer) *restful.WebService {
	r := networkResource{
		webResource: webResource{
			ds: ds,
		},
		ipamer: ipamer,
	}
	return r.webService()
}

func (r networkResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path("/v1/network").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"network"}

	ws.Route(ws.GET("/{id}").
		To(viewer(r.findNetwork)).
		Operation("findNetwork").
		Doc("get network by id").
		Param(ws.PathParameter("id", "identifier of the network").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.NetworkResponse{}).
		Returns(http.StatusOK, "OK", v1.NetworkResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/").
		To(viewer(r.listNetworks)).
		Operation("listNetworks").
		Doc("get all networks").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]v1.NetworkResponse{}).
		Returns(http.StatusOK, "OK", []v1.NetworkResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.DELETE("/{id}").
		To(editor(r.deleteNetwork)).
		Operation("deleteNetwork").
		Doc("deletes an network and returns the deleted entity").
		Param(ws.PathParameter("id", "identifier of the network").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.NetworkResponse{}).
		Returns(http.StatusOK, "OK", v1.NetworkResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.PUT("/").
		To(editor(r.createNetwork)).
		Operation("createNetwork").
		Doc("create an network. if the given ID already exists a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.NetworkCreateRequest{}).
		Returns(http.StatusCreated, "Created", v1.NetworkResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/").
		To(editor(r.updateNetwork)).
		Operation("updateNetwork").
		Doc("updates an network. if the network was changed since this one was read, a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.NetworkUpdateRequest{}).
		Returns(http.StatusOK, "OK", v1.NetworkResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r networkResource) findNetwork(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	nw, err := r.ds.FindNetwork(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	usage := r.getNetworkUsage(nw)
	response.WriteHeaderAndEntity(http.StatusOK, v1.NewNetworkResponse(nw, usage))
}

func (r networkResource) listNetworks(request *restful.Request, response *restful.Response) {
	nws, err := r.ds.ListNetworks()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	result := []*v1.NetworkResponse{}
	for i := range nws {
		usage := r.getNetworkUsage(&nws[i])
		result = append(result, v1.NewNetworkResponse(&nws[i], usage))
	}

	response.WriteHeaderAndEntity(http.StatusOK, result)
}

func (r networkResource) createNetwork(request *restful.Request, response *restful.Response) {
	var requestPayload v1.NetworkCreateRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	var id string
	if requestPayload.ID != nil {
		id = *requestPayload.ID
	}
	var name string
	if requestPayload.Name != nil {
		name = *requestPayload.Name
	}
	var description string
	if requestPayload.Description != nil {
		description = *requestPayload.Description
	}
	var projectid string
	if requestPayload.ProjectID != nil {
		projectid = *requestPayload.ProjectID
	}
	var vrfID uint
	if requestPayload.Vrf != nil {
		vrfID = *requestPayload.Vrf
	}
	primary := requestPayload.Primary
	underlay := requestPayload.Underlay
	nat := requestPayload.Nat

	if len(requestPayload.Prefixes) == 0 {
		// TODO: Should return a bad request 401
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("no prefixes given")) {
			return
		}
	}
	prefixes := metal.Prefixes{}
	// all Prefixes must be valid
	for _, p := range requestPayload.Prefixes {
		prefix, err := metal.NewPrefixFromCIDR(p)
		// TODO: Should return a bad request 401
		if err != nil {
			if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("given prefix %v is not a valid ip with mask: %v", p, err)) {
				return
			}
		}
		prefixes = append(prefixes, *prefix)
	}

	destPrefixes := metal.Prefixes{}
	for _, p := range requestPayload.DestinationPrefixes {
		prefix, err := metal.NewPrefixFromCIDR(p)
		if err != nil {
			if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("given prefix %v is not a valid ip with mask: %v", p, err)) {
				return
			}
		}
		destPrefixes = append(destPrefixes, *prefix)
	}

	allNws, err := r.ds.ListNetworks()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	existingPrefixes := metal.Prefixes{}
	existingPrefixesMap := make(map[string]bool)
	for _, nw := range allNws {
		for _, p := range nw.Prefixes {
			_, ok := existingPrefixesMap[p.String()]
			if !ok {
				existingPrefixes = append(existingPrefixes, p)
				existingPrefixesMap[p.String()] = true
			}
		}
	}

	err = r.ipamer.PrefixesOverlapping(existingPrefixes, prefixes)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	var partitionID string
	if requestPayload.PartitionID != nil {
		partition, err := r.ds.FindPartition(*requestPayload.PartitionID)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}

		if primary {
			primaryNetworks, err := r.ds.SearchPrimaryNetwork(partition.ID)
			if checkError(request, response, utils.CurrentFuncName(), err) {
				return
			}
			if len(primaryNetworks) != 0 {
				if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("partition with id %q already has a primary network", partition.ID)) {
					return
				}
			}
		}
		if underlay {
			underlayNetworks, err := r.ds.SearchUnderlayNetwork(partition.ID)
			if checkError(request, response, utils.CurrentFuncName(), err) {
				return
			}
			if len(underlayNetworks) != 0 {
				if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("partition with id %q already has a underlay network", partition.ID)) {
					return
				}
			}
		}
		partitionID = partition.ID
	}

	if (primary || underlay) && nat {
		checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("primary or underlay network is not supposed to NAT"))
		return
	}

	// TODO: Check if project exists if we get a project entity

	nw := &metal.Network{
		Base: metal.Base{
			ID:          id,
			Name:        name,
			Description: description,
		},
		Prefixes:            prefixes,
		DestinationPrefixes: destPrefixes,
		PartitionID:         partitionID,
		ProjectID:           projectid,
		Nat:                 nat,
		Primary:             primary,
		Underlay:            underlay,
		Vrf:                 vrfID,
	}

	for _, p := range nw.Prefixes {
		err := r.ipamer.CreatePrefix(p)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
	}

	if vrfID != 0 {
		vrfID := strconv.FormatUint(uint64(vrfID), 10)

		_, err := r.ds.FindVrf(vrfID)
		if err != nil {
			if metal.IsNotFound(err) {
				vrf := &metal.Vrf{
					Base: metal.Base{
						ID: vrfID,
					},
					ProjectID: projectid,
				}

				err = r.ds.CreateVrf(vrf)
				if checkError(request, response, utils.CurrentFuncName(), err) {
					return
				}
			} else {
				if checkError(request, response, utils.CurrentFuncName(), err) {
					return
				}
			}
		}
	}

	err = r.ds.CreateNetwork(nw)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	usage := r.getNetworkUsage(nw)

	response.WriteHeaderAndEntity(http.StatusCreated, v1.NewNetworkResponse(nw, usage))
}

func (r networkResource) updateNetwork(request *restful.Request, response *restful.Response) {
	var requestPayload v1.NetworkUpdateRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	oldNetwork, err := r.ds.FindNetwork(requestPayload.ID)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	newNetwork := *oldNetwork

	if requestPayload.Name != nil {
		newNetwork.Name = *requestPayload.Name
	}
	if requestPayload.Description != nil {
		newNetwork.Description = *requestPayload.Description
	}

	var prefixesToBeRemoved metal.Prefixes
	var prefixesToBeAdded metal.Prefixes

	if len(requestPayload.Prefixes) > 0 {
		var prefixesFromRequest metal.Prefixes
		for _, prefixCidr := range requestPayload.Prefixes {
			requestPrefix, err := metal.NewPrefixFromCIDR(prefixCidr)
			if err != nil {
				if checkError(request, response, utils.CurrentFuncName(), err) {
					return
				}
			}
			prefixesFromRequest = append(prefixesFromRequest, *requestPrefix)
		}
		newNetwork.Prefixes = prefixesFromRequest

		prefixesToBeRemoved = oldNetwork.SubstractPrefixes(prefixesFromRequest...)

		// now validate if there are ips which have a prefix to be removed as a parent
		allIPs, err := r.ds.ListIPs()
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
		err = checkAnyIPOfPrefixesInUse(allIPs, prefixesToBeRemoved)
		if err != nil {
			if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("unable to update network: %v", err)) {
				return
			}
		}

		prefixesToBeAdded = newNetwork.SubstractPrefixes(oldNetwork.Prefixes...)
	}

	for _, p := range prefixesToBeRemoved {
		err := r.ipamer.DeletePrefix(p)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
	}

	for _, p := range prefixesToBeAdded {
		err := r.ipamer.CreatePrefix(p)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
	}

	err = r.ds.UpdateNetwork(oldNetwork, &newNetwork)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	usage := r.getNetworkUsage(&newNetwork)

	response.WriteHeaderAndEntity(http.StatusOK, v1.NewNetworkResponse(&newNetwork, usage))
}

func (r networkResource) deleteNetwork(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	nw, err := r.ds.FindNetwork(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	allIPs, err := r.ds.ListIPs()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = checkAnyIPOfPrefixesInUse(allIPs, nw.Prefixes)
	if err != nil {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("unable to delete network: %v", err)) {
			return
		}
	}

	err = r.ds.DeleteNetwork(nw)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, v1.NewNetworkResponse(nw, v1.NetworkUsage{}))
}

func (r networkResource) getNetworkUsage(nw *metal.Network) v1.NetworkUsage {
	usage := v1.NetworkUsage{}
	for _, prefix := range nw.Prefixes {
		u, err := r.ipamer.PrefixUsage(prefix.String())
		if err != nil {
			continue
		}
		usage.AvailableIPs = usage.AvailableIPs + u.AvailableIPs
		usage.UsedIPs = usage.UsedIPs + u.UsedIPs
		usage.AvailablePrefixes = usage.AvailablePrefixes + u.AvailablePrefixes
		usage.UsedPrefixes = usage.UsedPrefixes + u.UsedPrefixes
	}
	return usage
}

func createChildPrefix(parentPrefixes metal.Prefixes, childLength int, ipamer ipam.IPAMer) (*metal.Prefix, error) {
	var errors []error
	var err error
	var childPrefix *metal.Prefix
	for _, parentPrefix := range parentPrefixes {
		childPrefix, err = ipamer.AllocateChildPrefix(parentPrefix, childLength)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		if childPrefix != nil {
			break
		}
	}
	if childPrefix == nil {
		if len(errors) > 0 {
			return nil, fmt.Errorf("cannot allocate free child prefix in ipam: %v", errors)
		}
		return nil, fmt.Errorf("cannot allocate free child prefix in one of the given parent prefixes in ipam: %v", parentPrefixes)
	}

	return childPrefix, nil
}

func checkAnyIPOfPrefixesInUse(ips []metal.IP, prefixes metal.Prefixes) error {
	for _, ip := range ips {
		for _, p := range prefixes {
			if ip.ParentPrefixCidr == p.String() {
				return fmt.Errorf("prefix %s has ip %s in use", p.String(), ip.IPAddress)
			}
		}
	}
	return nil
}
