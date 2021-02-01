package service

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/utils"
	"go.uber.org/zap"
	"inet.af/netaddr"

	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	mdm "github.com/metal-stack/masterdata-api/pkg/client"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/ipam"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/metal-lib/zapup"
	"github.com/prometheus/client_golang/prometheus"
)

type networkResource struct {
	webResource
	ipamer ipam.IPAMer
	mdc    mdm.Client
}

// NewNetwork returns a webservice for network specific endpoints.
func NewNetwork(ds *datastore.RethinkStore, ipamer ipam.IPAMer, mdc mdm.Client) *restful.WebService {
	r := networkResource{
		webResource: webResource{
			ds: ds,
		},
		ipamer: ipamer,
		mdc:    mdc,
	}
	nuc := networkUsageCollector{r: &r}
	err := prometheus.Register(nuc)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to register prometheus", zap.Error(err))
	}
	return r.webService()
}

func (r networkResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path(BasePath + "v1/network").
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

	ws.Route(ws.POST("/find").
		To(viewer(r.findNetworks)).
		Operation("findNetworks").
		Doc("get all networks that match given properties").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.NetworkFindRequest{}).
		Writes([]v1.NetworkResponse{}).
		Returns(http.StatusOK, "OK", []v1.NetworkResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.DELETE("/{id}").
		To(admin(r.deleteNetwork)).
		Operation("deleteNetwork").
		Doc("deletes a network and returns the deleted entity").
		Param(ws.PathParameter("id", "identifier of the network").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.NetworkResponse{}).
		Returns(http.StatusOK, "OK", v1.NetworkResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.PUT("/").
		To(admin(r.createNetwork)).
		Operation("createNetwork").
		Doc("create a network. if the given ID already exists a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.NetworkCreateRequest{}).
		Returns(http.StatusCreated, "Created", v1.NetworkResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/").
		To(admin(r.updateNetwork)).
		Operation("updateNetwork").
		Doc("updates a network. if the network was changed since this one was read, a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.NetworkUpdateRequest{}).
		Returns(http.StatusOK, "OK", v1.NetworkResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/allocate").
		To(editor(r.allocateNetwork)).
		Operation("allocateNetwork").
		Doc("allocates a child network from a partition's private super network").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.NetworkAllocateRequest{}).
		Returns(http.StatusCreated, "Created", v1.NetworkResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/free/{id}").
		To(editor(r.freeNetwork)).
		Operation("freeNetwork").
		Doc("free a network").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("id", "identifier of the network").DataType("string")).
		Returns(http.StatusOK, "OK", v1.NetworkResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r networkResource) findNetwork(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	nw, err := r.ds.FindNetworkByID(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	usage := getNetworkUsage(nw, r.ipamer)
	err = response.WriteHeaderAndEntity(http.StatusOK, v1.NewNetworkResponse(nw, usage))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r networkResource) listNetworks(request *restful.Request, response *restful.Response) {
	nws, err := r.ds.ListNetworks()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	var result []*v1.NetworkResponse
	for i := range nws {
		usage := getNetworkUsage(&nws[i], r.ipamer)
		result = append(result, v1.NewNetworkResponse(&nws[i], usage))
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, result)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r networkResource) findNetworks(request *restful.Request, response *restful.Response) {
	var requestPayload datastore.NetworkSearchQuery
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	var nws metal.Networks
	err = r.ds.SearchNetworks(&requestPayload, &nws)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	result := []*v1.NetworkResponse{}
	for i := range nws {
		usage := getNetworkUsage(&nws[i], r.ipamer)
		result = append(result, v1.NewNetworkResponse(&nws[i], usage))
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, result)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

// TODO allow creation of networks with childprefixlength which are not privatesuper
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
	var projectID string
	if requestPayload.ProjectID != nil {
		projectID = *requestPayload.ProjectID
	}
	var vrf uint
	if requestPayload.Vrf != nil {
		vrf = *requestPayload.Vrf
	}
	vrfShared := false
	if requestPayload.VrfShared != nil {
		vrfShared = *requestPayload.VrfShared
	}
	labels := make(map[string]string)
	if requestPayload.Labels != nil {
		labels = requestPayload.Labels
	}

	privateSuper := requestPayload.PrivateSuper
	underlay := requestPayload.Underlay
	nat := requestPayload.Nat

	if projectID != "" {
		_, err = r.mdc.Project().Get(context.Background(), &mdmv1.ProjectGetRequest{Id: projectID})
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
	}

	if len(requestPayload.Prefixes) == 0 {
		// TODO: Should return a bad request 401
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("no prefixes given")) {
			return
		}
	}
	prefixes := metal.Prefixes{}
	addressFamilies := make(map[string]bool)
	// all Prefixes must be valid and from the same addressfamily
	for _, p := range requestPayload.Prefixes {
		prefix, err := metal.NewPrefixFromCIDR(p)
		// TODO: Should return a bad request 401
		if err != nil {
			if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("given prefix %v is not a valid ip with mask: %v", p, err)) {
				return
			}
		}
		ipprefix, err := netaddr.ParseIPPrefix(p)
		if err != nil {
			if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("given prefix %v is not a valid ip with mask: %v", p, err)) {
				return
			}
		}
		if ipprefix.IP.Is4() {
			addressFamilies["ipv4"] = true
		}
		if ipprefix.IP.Is6() {
			addressFamilies["ipv6"] = true
		}
		prefixes = append(prefixes, *prefix)
	}

	if len(addressFamilies) > 1 {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("given prefixes have different addressfamilies")) {
			return
		}
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

		// We should support two private super per partition, one per addressfamily
		// the network allocate request must be configurable with addressfamily
		if privateSuper {
			boolTrue := true
			var nw metal.Network
			err := r.ds.FindNetwork(&datastore.NetworkSearchQuery{PartitionID: &partition.ID, PrivateSuper: &boolTrue}, &nw)
			if err != nil {
				if !metal.IsNotFound(err) {
					if checkError(request, response, utils.CurrentFuncName(), err) {
						return
					}
				}
			} else {
				existingsuper := nw.Prefixes[0].String()
				ipprefxexistingsuper, err := netaddr.ParseIPPrefix(existingsuper)
				if err != nil {
					if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("given prefix %v is not a valid ip with mask: %v", existingsuper, err)) {
						return
					}
				}
				newsuper := prefixes[0].String()
				ipprefixnew, err := netaddr.ParseIPPrefix(newsuper)
				if err != nil {
					if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("given prefix %v is not a valid ip with mask: %v", newsuper, err)) {
						return
					}
				}
				if (ipprefixnew.IP.Is4() && ipprefxexistingsuper.IP.Is4()) || (ipprefixnew.IP.Is6() && ipprefxexistingsuper.IP.Is6()) {
					if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("partition with id %q already has a private super network for this addressfamily", partition.ID)) {
						return
					}
				}
			}
		}
		if underlay {
			boolTrue := true
			err := r.ds.FindNetwork(&datastore.NetworkSearchQuery{PartitionID: &partition.ID, Underlay: &boolTrue}, &metal.Network{})
			if err != nil {
				if !metal.IsNotFound(err) {
					if checkError(request, response, utils.CurrentFuncName(), err) {
						return
					}
				}
			} else {
				if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("partition with id %q already has an underlay network", partition.ID)) {
					return
				}
			}
		}
		partitionID = partition.ID
	}

	if (privateSuper || underlay) && nat {
		checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("private super or underlay network is not supposed to NAT"))
		return
	}

	if vrf != 0 {
		err = acquireVRF(r.ds, vrf)
		if err != nil {
			if !metal.IsConflict(err) {
				if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("could not acquire vrf: %v", err)) {
					return
				}
			}
			if !vrfShared {
				if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("cannot acquire a unique vrf id twice except vrfShared is set to true: %v", err)) {
					return
				}
			}
		}
	}

	nw := &metal.Network{
		Base: metal.Base{
			ID:          id,
			Name:        name,
			Description: description,
		},
		Prefixes:            prefixes,
		DestinationPrefixes: destPrefixes,
		PartitionID:         partitionID,
		ProjectID:           projectID,
		Nat:                 nat,
		PrivateSuper:        privateSuper,
		Underlay:            underlay,
		Vrf:                 vrf,
		Labels:              labels,
	}

	// check if childprefixlength is set and matches addressfamily
	if requestPayload.ChildPrefixLength != nil && privateSuper {
		cpl := *requestPayload.ChildPrefixLength
		for _, p := range prefixes {
			ipprefix, err := netaddr.ParseIPPrefix(p.String())
			if err != nil {
				if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("given prefix %v is not a valid ip with mask: %v", p, err)) {
					return
				}
			}
			if cpl <= ipprefix.Bits {
				if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("given childprefixlength %d is not greater than prefix length of:%s", cpl, p.String())) {
					return
				}
			}
		}
		nw.ChildPrefixLength = requestPayload.ChildPrefixLength
	}

	for _, p := range nw.Prefixes {
		err := r.ipamer.CreatePrefix(p)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
	}

	err = r.ds.CreateNetwork(nw)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	usage := getNetworkUsage(nw, r.ipamer)
	err = response.WriteHeaderAndEntity(http.StatusCreated, v1.NewNetworkResponse(nw, usage))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

// TODO add possibility to allocate from a non super network if given in the AllocateRequest and super has childprefixlength
func (r networkResource) allocateNetwork(request *restful.Request, response *restful.Response) {
	var requestPayload v1.NetworkAllocateRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	var name string
	if requestPayload.Name != nil {
		name = *requestPayload.Name
	}
	var description string
	if requestPayload.Description != nil {
		description = *requestPayload.Description
	}
	var projectID string
	if requestPayload.ProjectID != nil {
		projectID = *requestPayload.ProjectID
	}
	var partitionID string
	if requestPayload.PartitionID != nil {
		partitionID = *requestPayload.PartitionID
	}
	shared := false
	if requestPayload.Shared != nil {
		shared = *requestPayload.Shared
	}

	if projectID == "" {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("projectid should not be empty")) {
			return
		}
	}
	if partitionID == "" {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("partitionid should not be empty")) {
			return
		}
	}

	project, err := r.mdc.Project().Get(context.Background(), &mdmv1.ProjectGetRequest{Id: projectID})
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	partition, err := r.ds.FindPartition(partitionID)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	var superNetworks metal.Networks
	boolTrue := true
	err = r.ds.SearchNetworks(&datastore.NetworkSearchQuery{PartitionID: &partition.ID, PrivateSuper: &boolTrue}, &superNetworks)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	addressFamily := v1.IPv4AddressFamily
	if requestPayload.AddressFamily != nil {
		addressFamily = v1.ToAddressFamily(*requestPayload.AddressFamily)
	}
	zapup.MustRootLogger().Sugar().Infow("network allocate", "family", addressFamily)
	var (
		superNetwork      metal.Network
		superNetworkFound bool
	)
	for _, snw := range superNetworks {
		ipprefix, err := netaddr.ParseIPPrefix(snw.Prefixes[0].String())
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
		if addressFamily == v1.IPv4AddressFamily && ipprefix.IP.Is4() {
			superNetwork = snw
			superNetworkFound = true
		}
		if addressFamily == v1.IPv6AddressFamily && ipprefix.IP.Is6() {
			superNetwork = snw
			superNetworkFound = true
		}
	}
	if !superNetworkFound {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("no supernetwork for addressfamily:%s found", addressFamily)) {
			return
		}
	}
	zapup.MustRootLogger().Sugar().Infow("network allocate", "supernetwork", superNetwork.ID)

	nwSpec := &metal.Network{
		Base: metal.Base{
			Name:        name,
			Description: description,
		},
		PartitionID: partition.ID,
		ProjectID:   project.GetProject().GetMeta().GetId(),
		Labels:      requestPayload.Labels,
		Shared:      shared,
	}

	if superNetwork.ChildPrefixLength == nil {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("supernetwork %s has no childprefixlength specified", superNetwork.ID)) {
			return
		}
	}

	// Allow configurable prefix length
	length := *superNetwork.ChildPrefixLength
	if requestPayload.Length != nil {
		length = *requestPayload.Length
	}

	nw, err := createChildNetwork(r.ds, r.ipamer, nwSpec, &superNetwork, length)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	usage := getNetworkUsage(nw, r.ipamer)
	err = response.WriteHeaderAndEntity(http.StatusCreated, v1.NewNetworkResponse(nw, usage))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func createChildNetwork(ds *datastore.RethinkStore, ipamer ipam.IPAMer, nwSpec *metal.Network, parent *metal.Network, childLength uint8) (*metal.Network, error) {
	vrf, err := acquireRandomVRF(ds)
	if err != nil {
		return nil, fmt.Errorf("Could not acquire a vrf: %v", err)
	}

	childPrefix, err := createChildPrefix(parent.Prefixes, childLength, ipamer)
	if err != nil {
		return nil, err
	}

	if childPrefix == nil {
		return nil, fmt.Errorf("could not allocate child prefix in parent network: %s", parent.ID)
	}

	nw := &metal.Network{
		Base: metal.Base{
			Name:        nwSpec.Name,
			Description: nwSpec.Description,
		},
		Prefixes:            metal.Prefixes{*childPrefix},
		DestinationPrefixes: metal.Prefixes{},
		PartitionID:         parent.PartitionID,
		ProjectID:           nwSpec.ProjectID,
		Nat:                 false,
		PrivateSuper:        false,
		Underlay:            false,
		Shared:              nwSpec.Shared,
		Vrf:                 *vrf,
		ParentNetworkID:     parent.ID,
		Labels:              nwSpec.Labels,
	}

	err = ds.CreateNetwork(nw)
	if err != nil {
		return nil, err
	}

	return nw, nil
}

func (r networkResource) freeNetwork(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	nw, err := r.ds.FindNetworkByID(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	for _, prefix := range nw.Prefixes {
		usage, err := r.ipamer.PrefixUsage(prefix.String())
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
		if usage.UsedIPs > 2 {
			if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("cannot release child prefix %s because IPs in the prefix are still in use: %v", prefix.String(), usage.UsedIPs-2)) {
				return
			}
		}
	}

	for _, prefix := range nw.Prefixes {
		err = r.ipamer.ReleaseChildPrefix(prefix)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
	}

	if nw.Vrf != 0 {
		err = releaseVRF(r.ds, nw.Vrf)
		if err != nil {
			if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("could not release vrf: %v", err)) {
				return
			}
		}
	}

	err = r.ds.DeleteNetwork(nw)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, v1.NewNetworkResponse(nw, &metal.NetworkUsage{}))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r networkResource) updateNetwork(request *restful.Request, response *restful.Response) {
	var requestPayload v1.NetworkUpdateRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	oldNetwork, err := r.ds.FindNetworkByID(requestPayload.ID)
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
	if requestPayload.Labels != nil {
		newNetwork.Labels = requestPayload.Labels
	}
	if requestPayload.Shared != nil {
		newNetwork.Shared = *requestPayload.Shared
	}

	if oldNetwork.Shared && !newNetwork.Shared {
		checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("once a network is marked as shared it is not possible to unshare it"))
		return
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

	usage := getNetworkUsage(&newNetwork, r.ipamer)
	err = response.WriteHeaderAndEntity(http.StatusOK, v1.NewNetworkResponse(&newNetwork, usage))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r networkResource) deleteNetwork(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	nw, err := r.ds.FindNetworkByID(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	var children metal.Networks
	err = r.ds.SearchNetworks(&datastore.NetworkSearchQuery{ParentNetworkID: &nw.ID}, &children)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	if len(children) != 0 {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("network cannot be deleted because there are children of this network")) {
			return
		}
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

	for _, p := range nw.Prefixes {
		err := r.ipamer.DeletePrefix(p)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
	}

	if nw.Vrf != 0 {
		err = releaseVRF(r.ds, nw.Vrf)
		if err != nil {
			if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("could not release vrf: %v", err)) {
				return
			}
		}
	}

	err = r.ds.DeleteNetwork(nw)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, v1.NewNetworkResponse(nw, &metal.NetworkUsage{}))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func getNetworkUsage(nw *metal.Network, ipamer ipam.IPAMer) *metal.NetworkUsage {
	usage := &metal.NetworkUsage{}
	if nw == nil {
		return usage
	}
	for _, prefix := range nw.Prefixes {
		u, err := ipamer.PrefixUsage(prefix.String())
		if err != nil {
			continue
		}
		usage.AvailableIPs = usage.AvailableIPs + u.AvailableIPs
		usage.UsedIPs = usage.UsedIPs + u.UsedIPs
		usage.AvailablePrefixList = append(usage.AvailablePrefixList, u.AvailablePrefixList...)
		usage.AvailablePrefixes = usage.AvailablePrefixes + u.AvailablePrefixes
		usage.UsedPrefixes = usage.UsedPrefixes + u.UsedPrefixes
	}
	return usage
}

func createChildPrefix(parentPrefixes metal.Prefixes, childLength uint8, ipamer ipam.IPAMer) (*metal.Prefix, error) {
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

// networkUsageCollector implements the prometheus collector interface.
type networkUsageCollector struct {
	r *networkResource
}

var (
	usedIpsDesc = prometheus.NewDesc(
		"metal_network_ip_used",
		"The total number of used IPs of the network",
		[]string{"networkId", "prefixes", "destPrefixes", "partitionId", "projectId", "parentNetworkID", "vrf", "isPrivateSuper", "useNat", "isUnderlay"}, nil,
	)
	availableIpsDesc = prometheus.NewDesc(
		"metal_network_ip_available",
		"The total number of available IPs of the network",
		[]string{"networkId", "prefixes", "destPrefixes", "partitionId", "projectId", "parentNetworkID", "vrf", "isPrivateSuper", "useNat", "isUnderlay"}, nil,
	)
	usedPrefixesDesc = prometheus.NewDesc(
		"metal_network_prefix_used",
		"The total number of used prefixes of the network",
		[]string{"networkId", "prefixes", "destPrefixes", "partitionId", "projectId", "parentNetworkID", "vrf", "isPrivateSuper", "useNat", "isUnderlay"}, nil,
	)
	availablePrefixesDesc = prometheus.NewDesc(
		"metal_network_prefix_available",
		"The total number of available 2 bit prefixes of the network",
		[]string{"networkId", "prefixes", "destPrefixes", "partitionId", "projectId", "parentNetworkID", "vrf", "isPrivateSuper", "useNat", "isUnderlay"}, nil,
	)
)

func (nuc networkUsageCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(nuc, ch)
}

func (nuc networkUsageCollector) Collect(ch chan<- prometheus.Metric) {
	// FIXME bad workaround to be able to run make spec
	if nuc.r == nil || nuc.r.ds == nil {
		return
	}
	nws, err := nuc.r.ds.ListNetworks()
	if err != nil {
		zapup.MustRootLogger().Error("Failed to get network usage", zap.Error(err))
		return
	}

	for i := range nws {
		usage := getNetworkUsage(&nws[i], nuc.r.ipamer)

		privateSuper := fmt.Sprintf("%t", nws[i].PrivateSuper)
		nat := fmt.Sprintf("%t", nws[i].Nat)
		underlay := fmt.Sprintf("%t", nws[i].Underlay)
		prefixes := strings.Join(nws[i].Prefixes.String(), ",")
		destPrefixes := strings.Join(nws[i].DestinationPrefixes.String(), ",")
		vrf := strconv.FormatUint(uint64(nws[i].Vrf), 3)

		metric, err := prometheus.NewConstMetric(
			usedIpsDesc,
			prometheus.CounterValue,
			float64(usage.UsedIPs),
			nws[i].ID,
			prefixes,
			destPrefixes,
			nws[i].PartitionID,
			nws[i].ProjectID,
			nws[i].ParentNetworkID,
			vrf,
			privateSuper,
			nat,
			underlay,
		)
		if err != nil {
			zapup.MustRootLogger().Error("Failed create metric for UsedIPs", zap.Error(err))
			return
		}
		ch <- metric

		metric, err = prometheus.NewConstMetric(
			availableIpsDesc,
			prometheus.CounterValue,
			float64(usage.AvailableIPs),
			nws[i].ID,
			prefixes,
			destPrefixes,
			nws[i].PartitionID,
			nws[i].ProjectID,
			nws[i].ParentNetworkID,
			vrf,
			privateSuper,
			nat,
			underlay,
		)
		if err != nil {
			zapup.MustRootLogger().Error("Failed create metric for AvailableIPs", zap.Error(err))
			return
		}
		ch <- metric
		metric, err = prometheus.NewConstMetric(
			usedPrefixesDesc,
			prometheus.CounterValue,
			float64(usage.UsedPrefixes),
			nws[i].ID,
			prefixes,
			destPrefixes,
			nws[i].PartitionID,
			nws[i].ProjectID,
			nws[i].ParentNetworkID,
			vrf,
			privateSuper,
			nat,
			underlay,
		)
		if err != nil {
			zapup.MustRootLogger().Error("Failed create metric for UsedPrefixes", zap.Error(err))
			return
		}
		ch <- metric
		metric, err = prometheus.NewConstMetric(
			availablePrefixesDesc,
			prometheus.CounterValue,
			float64(usage.AvailablePrefixes),
			nws[i].ID,
			prefixes,
			destPrefixes,
			nws[i].PartitionID,
			nws[i].ProjectID,
			nws[i].ParentNetworkID,
			vrf,
			privateSuper,
			nat,
			underlay,
		)
		if err != nil {
			zapup.MustRootLogger().Error("Failed create metric for AvailablePrefixes", zap.Error(err))
			return
		}
		ch <- metric
	}
}
