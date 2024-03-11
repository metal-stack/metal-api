package service

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	mdm "github.com/metal-stack/masterdata-api/pkg/client"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/ipam"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-lib/auditing"
	"github.com/metal-stack/metal-lib/httperrors"
)

type networkResource struct {
	webResource
	ipamer ipam.IPAMer
	mdc    mdm.Client
}

// NewNetwork returns a webservice for network specific endpoints.
func NewNetwork(log *slog.Logger, ds *datastore.RethinkStore, ipamer ipam.IPAMer, mdc mdm.Client) *restful.WebService {
	r := networkResource{
		webResource: webResource{
			log: log,
			ds:  ds,
		},
		ipamer: ipamer,
		mdc:    mdc,
	}

	return r.webService()
}

func (r *networkResource) webService() *restful.WebService {
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
		Metadata(auditing.Exclude, true).
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
		Operation("freeNetworkDeprecated").
		Doc("free a network").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Consumes(restful.MIME_JSON, "*/*").
		Param(ws.PathParameter("id", "identifier of the network").DataType("string")).
		Returns(http.StatusOK, "OK", v1.NetworkResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}).
		Deprecate())

	ws.Route(ws.DELETE("/free/{id}").
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

func (r *networkResource) findNetwork(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	nw, err := r.ds.FindNetworkByID(id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	usage := getNetworkUsage(nw, r.ipamer)

	r.send(request, response, http.StatusOK, v1.NewNetworkResponse(nw, usage))
}

func (r *networkResource) listNetworks(request *restful.Request, response *restful.Response) {
	nws, err := r.ds.ListNetworks()
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	var result []*v1.NetworkResponse
	for i := range nws {
		usage := getNetworkUsage(&nws[i], r.ipamer)
		result = append(result, v1.NewNetworkResponse(&nws[i], usage))
	}

	r.send(request, response, http.StatusOK, result)
}

func (r *networkResource) findNetworks(request *restful.Request, response *restful.Response) {
	var requestPayload datastore.NetworkSearchQuery
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	var nws metal.Networks
	err = r.ds.SearchNetworks(&requestPayload, &nws)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	result := []*v1.NetworkResponse{}
	for i := range nws {
		usage := getNetworkUsage(&nws[i], r.ipamer)
		result = append(result, v1.NewNetworkResponse(&nws[i], usage))
	}

	r.send(request, response, http.StatusOK, result)
}

func (r *networkResource) createNetwork(request *restful.Request, response *restful.Response) {
	var requestPayload v1.NetworkCreateRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
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
		_, err = r.mdc.Project().Get(request.Request.Context(), &mdmv1.ProjectGetRequest{Id: projectID})
		if err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}
	}

	if len(requestPayload.Prefixes) == 0 {
		r.sendError(request, response, httperrors.BadRequest(errors.New("no prefixes given")))
		return
	}

	prefixes := metal.Prefixes{}
	// all Prefixes must be valid
	for i := range requestPayload.Prefixes {
		p := requestPayload.Prefixes[i]
		prefix, err := metal.NewPrefixFromCIDR(p)
		if err != nil {
			r.sendError(request, response, httperrors.BadRequest(fmt.Errorf("given prefix %v is not a valid ip with mask: %w", p, err)))
			return
		}

		prefixes = append(prefixes, *prefix)
	}

	destPrefixes := metal.Prefixes{}
	for i := range requestPayload.DestinationPrefixes {
		p := requestPayload.DestinationPrefixes[i]
		prefix, err := metal.NewPrefixFromCIDR(p)
		if err != nil {
			r.sendError(request, response, httperrors.BadRequest(fmt.Errorf("given prefix %v is not a valid ip with mask: %w", p, err)))
			return
		}
		destPrefixes = append(destPrefixes, *prefix)
	}

	allNws, err := r.ds.ListNetworks()
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	if id != "" {
		for existingID := range allNws.ByID() {
			if existingID == id {
				r.sendError(request, response, httperrors.Conflict(fmt.Errorf("entity already exists")))
				return
			}
		}
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
	if err != nil {
		r.sendError(request, response, httperrors.UnprocessableEntity(err))
		return
	}

	var partitionID string
	if requestPayload.PartitionID != nil {
		partition, err := r.ds.FindPartition(*requestPayload.PartitionID)
		if err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}

		if privateSuper {
			boolTrue := true
			err := r.ds.FindNetwork(&datastore.NetworkSearchQuery{PartitionID: &partition.ID, PrivateSuper: &boolTrue}, &metal.Network{})
			if err != nil {
				if !metal.IsNotFound(err) {
					r.sendError(request, response, defaultError(err))
					return
				}
			} else {
				r.sendError(request, response, defaultError(fmt.Errorf("partition with id %q already has a private super network", partition.ID)))
				return
			}
		}
		if underlay {
			boolTrue := true
			err := r.ds.FindNetwork(&datastore.NetworkSearchQuery{PartitionID: &partition.ID, Underlay: &boolTrue}, &metal.Network{})
			if err != nil {
				if !metal.IsNotFound(err) {
					r.sendError(request, response, defaultError(err))
					return
				}
			} else {
				r.sendError(request, response, defaultError(fmt.Errorf("partition with id %q already has an underlay network", partition.ID)))
				return
			}
		}
		partitionID = partition.ID
	}

	if (privateSuper || underlay) && nat {
		r.sendError(request, response, httperrors.BadRequest(errors.New("private super or underlay network is not supposed to NAT")))
		return
	}

	if vrf != 0 {
		err = acquireVRF(r.ds, vrf)
		if err != nil {
			if !metal.IsConflict(err) {
				r.sendError(request, response, defaultError(fmt.Errorf("could not acquire vrf: %w", err)))
				return
			}
			if !vrfShared {
				r.sendError(request, response, defaultError(fmt.Errorf("cannot acquire a unique vrf id twice except vrfShared is set to true: %w", err)))
				return
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

	for _, p := range nw.Prefixes {
		err := r.ipamer.CreatePrefix(p)
		if err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}
	}

	err = r.ds.CreateNetwork(nw)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	usage := getNetworkUsage(nw, r.ipamer)

	r.send(request, response, http.StatusCreated, v1.NewNetworkResponse(nw, usage))
}

func (r *networkResource) allocateNetwork(request *restful.Request, response *restful.Response) {
	var requestPayload v1.NetworkAllocateRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
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
	var shared bool
	if requestPayload.Shared != nil {
		shared = *requestPayload.Shared
	}
	var nat bool
	if requestPayload.Nat != nil {
		nat = *requestPayload.Nat
	}

	if projectID == "" {
		r.sendError(request, response, httperrors.BadRequest(errors.New("projectid should not be empty")))
		return
	}
	if partitionID == "" {
		r.sendError(request, response, httperrors.BadRequest(errors.New("partitionid should not be empty")))
		return
	}

	project, err := r.mdc.Project().Get(request.Request.Context(), &mdmv1.ProjectGetRequest{Id: projectID})
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	partition, err := r.ds.FindPartition(partitionID)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	var superNetwork metal.Network
	boolTrue := true
	err = r.ds.FindNetwork(&datastore.NetworkSearchQuery{PartitionID: &partition.ID, PrivateSuper: &boolTrue}, &superNetwork)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	destPrefixes := metal.Prefixes{}
	for _, p := range requestPayload.DestinationPrefixes {
		prefix, err := metal.NewPrefixFromCIDR(p)
		if err != nil {
			r.sendError(request, response, httperrors.BadRequest(fmt.Errorf("given prefix %v is not a valid ip with mask: %w", p, err)))
			return
		}

		destPrefixes = append(destPrefixes, *prefix)
	}

	nwSpec := &metal.Network{
		Base: metal.Base{
			Name:        name,
			Description: description,
		},
		PartitionID:         partition.ID,
		ProjectID:           project.GetProject().GetMeta().GetId(),
		Labels:              requestPayload.Labels,
		DestinationPrefixes: destPrefixes,
		Shared:              shared,
		Nat:                 nat,
	}

	nw, err := createChildNetwork(r.ds, r.ipamer, nwSpec, &superNetwork, partition.PrivateNetworkPrefixLength)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	usage := getNetworkUsage(nw, r.ipamer)

	r.send(request, response, http.StatusCreated, v1.NewNetworkResponse(nw, usage))
}

func createChildNetwork(ds *datastore.RethinkStore, ipamer ipam.IPAMer, nwSpec *metal.Network, parent *metal.Network, childLength uint8) (*metal.Network, error) {
	vrf, err := acquireRandomVRF(ds)
	if err != nil {
		return nil, fmt.Errorf("could not acquire a vrf: %w", err)
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
		DestinationPrefixes: nwSpec.DestinationPrefixes,
		PartitionID:         parent.PartitionID,
		ProjectID:           nwSpec.ProjectID,
		Nat:                 nwSpec.Nat,
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

func (r *networkResource) freeNetwork(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	nw, err := r.ds.FindNetworkByID(id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	for _, prefix := range nw.Prefixes {
		err = r.ipamer.ReleaseChildPrefix(prefix)
		if err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}
	}

	if nw.Vrf != 0 {
		err = releaseVRF(r.ds, nw.Vrf)
		if err != nil {
			r.sendError(request, response, defaultError(fmt.Errorf("could not release vrf: %w", err)))
			return
		}
	}

	err = r.ds.DeleteNetwork(nw)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, v1.NewNetworkResponse(nw, &metal.NetworkUsage{}))
}

func (r *networkResource) updateNetwork(request *restful.Request, response *restful.Response) {
	var requestPayload v1.NetworkUpdateRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	oldNetwork, err := r.ds.FindNetworkByID(requestPayload.ID)
	if err != nil {
		r.sendError(request, response, defaultError(err))
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
		r.sendError(request, response, httperrors.BadRequest(errors.New("once a network is marked as shared it is not possible to unshare it")))
		return
	}

	var prefixesToBeRemoved metal.Prefixes
	var prefixesToBeAdded metal.Prefixes

	if len(requestPayload.Prefixes) > 0 {
		newNetwork.Prefixes, err = prefixesFromCidr(requestPayload.Prefixes)
		if err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}

		prefixesToBeRemoved = oldNetwork.SubstractPrefixes(newNetwork.Prefixes...)

		// now validate if there are ips which have a prefix to be removed as a parent
		allIPs, err := r.ds.ListIPs()
		if err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}

		err = checkAnyIPOfPrefixesInUse(allIPs, prefixesToBeRemoved)
		if err != nil {
			r.sendError(request, response, defaultError(fmt.Errorf("unable to update network: %w", err)))
			return
		}

		prefixesToBeAdded = newNetwork.SubstractPrefixes(oldNetwork.Prefixes...)
	}

	for _, p := range prefixesToBeRemoved {
		err := r.ipamer.DeletePrefix(p)
		if err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}
	}

	for _, p := range prefixesToBeAdded {
		err := r.ipamer.CreatePrefix(p)
		if err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}
	}

	if len(requestPayload.DestinationPrefixes) > 0 {
		newNetwork.DestinationPrefixes, err = prefixesFromCidr(requestPayload.DestinationPrefixes)
		if err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}
	}

	err = r.ds.UpdateNetwork(oldNetwork, &newNetwork)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	usage := getNetworkUsage(&newNetwork, r.ipamer)

	r.send(request, response, http.StatusOK, v1.NewNetworkResponse(&newNetwork, usage))
}

func prefixesFromCidr(PrefixesCidr []string) (metal.Prefixes, error) {
	var prefixes metal.Prefixes
	for _, prefixCidr := range PrefixesCidr {
		Prefix, err := metal.NewPrefixFromCIDR(prefixCidr)
		if err != nil {
			return nil, err
		}
		prefixes = append(prefixes, *Prefix)
	}
	return prefixes, nil
}

func (r *networkResource) deleteNetwork(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	nw, err := r.ds.FindNetworkByID(id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	var children metal.Networks
	err = r.ds.SearchNetworks(&datastore.NetworkSearchQuery{ParentNetworkID: &nw.ID}, &children)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	if len(children) != 0 {
		r.sendError(request, response, defaultError(errors.New("network cannot be deleted because there are children of this network")))
		return
	}

	allIPs, err := r.ds.ListIPs()
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	err = checkAnyIPOfPrefixesInUse(allIPs, nw.Prefixes)
	if err != nil {
		r.sendError(request, response, defaultError(fmt.Errorf("unable to delete network: %w", err)))
		return
	}

	for _, p := range nw.Prefixes {
		err := r.ipamer.DeletePrefix(p)
		if err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}
	}

	if nw.Vrf != 0 {
		err = releaseVRF(r.ds, nw.Vrf)
		if err != nil {
			r.sendError(request, response, defaultError(fmt.Errorf("could not release vrf: %w", err)))
			return
		}
	}

	err = r.ds.DeleteNetwork(nw)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, v1.NewNetworkResponse(nw, &metal.NetworkUsage{}))
}

func getNetworkUsage(nw *metal.Network, ipamer ipam.IPAMer) *metal.NetworkUsage {
	usage := &metal.NetworkUsage{}
	if nw == nil {
		return usage
	}
	for _, prefix := range nw.Prefixes {
		u, err := ipamer.PrefixUsage(prefix.String())
		if err != nil {
			// FIXME: the error should not be swallowed here as this can return wrong usage information to the clients
			continue
		}
		usage.AvailableIPs = usage.AvailableIPs + u.AvailableIPs
		usage.UsedIPs = usage.UsedIPs + u.UsedIPs
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
