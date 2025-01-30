package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"slices"
	"strconv"

	"connectrpc.com/connect"
	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	mdm "github.com/metal-stack/masterdata-api/pkg/client"
	"github.com/samber/lo"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/ipam"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-lib/auditing"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/metal-lib/pkg/pointer"
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
		Param(ws.QueryParameter("force", "if true update forcefully").DataType("boolean").DefaultValue("false")).
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
	ctx := request.Request.Context()
	consumption, err := r.getNetworkUsage(ctx, nw)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, v1.NewNetworkResponse(nw, consumption))
}

func (r *networkResource) listNetworks(request *restful.Request, response *restful.Response) {
	nws, err := r.ds.ListNetworks()
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}
	ctx := request.Request.Context()
	var result []*v1.NetworkResponse
	for i := range nws {
		consumption, err := r.getNetworkUsage(ctx, &nws[i])
		if err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}

		result = append(result, v1.NewNetworkResponse(&nws[i], consumption))
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

	if err := requestPayload.Validate(); err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	var nws metal.Networks
	err = r.ds.SearchNetworks(&requestPayload, &nws)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	ctx := request.Request.Context()
	result := []*v1.NetworkResponse{}
	for i := range nws {
		consumption, err := r.getNetworkUsage(ctx, &nws[i])
		if err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}

		result = append(result, v1.NewNetworkResponse(&nws[i], consumption))
	}

	r.send(request, response, http.StatusOK, result)
}

// TODO allow creation of networks with childprefixlength which are not privatesuper
// See https://github.com/metal-stack/metal-api/issues/16
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

	var childPrefixLength = metal.ChildPrefixLength{}
	for af, length := range requestPayload.DefaultChildPrefixLength {
		childPrefixLength[metal.ToAddressFamily(string(af))] = length
	}

	prefixes, destPrefixes, addressFamilies, err := validatePrefixesAndAddressFamilies(requestPayload.Prefixes, requestPayload.DestinationPrefixes, childPrefixLength, privateSuper)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
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
			var nw metal.Network
			err := r.ds.FindNetwork(&datastore.NetworkSearchQuery{
				PartitionID:  &partition.ID,
				PrivateSuper: pointer.Pointer(true),
			}, &nw)
			if err != nil && !metal.IsNotFound(err) {
				r.sendError(request, response, defaultError(err))
				return
			}
			if nw.ID != "" {
				r.sendError(request, response, httperrors.BadRequest(fmt.Errorf("partition with id %q already has a private super network", partition.ID)))
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

	err = validateAdditionalAnnouncableCIDRs(requestPayload.AdditionalAnnouncableCIDRs, privateSuper)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
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
		Prefixes:                   prefixes,
		DestinationPrefixes:        destPrefixes,
		DefaultChildPrefixLength:   childPrefixLength,
		PartitionID:                partitionID,
		ProjectID:                  projectID,
		Nat:                        nat,
		PrivateSuper:               privateSuper,
		Underlay:                   underlay,
		Vrf:                        vrf,
		Labels:                     labels,
		AddressFamilies:            addressFamilies,
		AdditionalAnnouncableCIDRs: requestPayload.AdditionalAnnouncableCIDRs,
	}

	ctx := request.Request.Context()
	for _, p := range nw.Prefixes {
		err := r.ipamer.CreatePrefix(ctx, p)
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

	consumption, err := r.getNetworkUsage(ctx, nw)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusCreated, v1.NewNetworkResponse(nw, consumption))
}

func validateAdditionalAnnouncableCIDRs(additionalCidrs []string, privateSuper bool) error {
	if len(additionalCidrs) == 0 {
		return nil
	}

	if !privateSuper {
		return errors.New("additionalannouncablecidrs can only be set in a private super network")
	}

	for _, cidr := range additionalCidrs {
		_, err := netip.ParsePrefix(cidr)
		if err != nil {
			return fmt.Errorf("given cidr:%q in additionalannouncablecidrs is malformed:%w", cidr, err)
		}
	}

	return nil
}

func validatePrefixesAndAddressFamilies(prefixes, destinationPrefixes []string, defaultChildPrefixLength metal.ChildPrefixLength, privateSuper bool) (metal.Prefixes, metal.Prefixes, metal.AddressFamilies, error) {

	pfxs, addressFamilies, err := validatePrefixes(prefixes)
	if err != nil {
		return nil, nil, nil, err
	}
	// all DestinationPrefixes must be valid and from the same addressfamily
	destPfxs, destinationAddressFamilies, err := validatePrefixes(destinationPrefixes)
	if err != nil {
		return nil, nil, nil, err
	}

	if len(destinationAddressFamilies) > len(addressFamilies) {
		return nil, nil, nil, fmt.Errorf("destination prefixes have more addressfamilies than prefixes")

	}

	if !privateSuper {
		return pfxs, destPfxs, addressFamilies, nil
	}

	if len(defaultChildPrefixLength) == 0 {
		return nil, nil, nil, fmt.Errorf("private super network must always contain a defaultchildprefixlength")
	}

	for af, length := range defaultChildPrefixLength {
		if !slices.Contains(addressFamilies, af) {
			return nil, nil, nil, fmt.Errorf("private super network must always contain a defaultchildprefixlength per addressfamily:%s", af)
		}

		// check if childprefixlength is set and matches addressfamily
		for _, p := range pfxs.OfFamily(af) {
			ipprefix, err := netip.ParsePrefix(p.String())
			if err != nil {
				return nil, nil, nil, fmt.Errorf("given prefix %v is not a valid ip with mask: %w", p, err)
			}
			if int(length) <= ipprefix.Bits() {
				return nil, nil, nil, fmt.Errorf("given defaultchildprefixlength %d is not greater than prefix length of:%s", length, p.String())
			}
		}
	}

	return pfxs, destPfxs, addressFamilies, nil
}

func validatePrefixes(prefixes []string) (metal.Prefixes, metal.AddressFamilies, error) {
	var (
		result          metal.Prefixes
		addressFamilies = make(map[metal.AddressFamily]bool)
	)
	for _, p := range prefixes {
		prefix, ipprefix, err := metal.NewPrefixFromCIDR(p)
		if err != nil {
			return nil, nil, fmt.Errorf("given prefix %v is not a valid ip with mask: %w", p, err)
		}
		if ipprefix.Addr().Is4() {
			addressFamilies[metal.IPv4AddressFamily] = true
		}
		if ipprefix.Addr().Is6() {
			addressFamilies[metal.IPv6AddressFamily] = true
		}
		result = append(result, *prefix)
	}
	return result, lo.Keys(addressFamilies), nil
}

// TODO add possibility to allocate from a non super network if given in the AllocateRequest and super has childprefixlength
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

	destPrefixes := metal.Prefixes{}
	for _, p := range requestPayload.DestinationPrefixes {
		prefix, _, err := metal.NewPrefixFromCIDR(p)
		if err != nil {
			r.sendError(request, response, httperrors.BadRequest(fmt.Errorf("given prefix %v is not a valid ip with mask: %w", p, err)))
			return
		}

		destPrefixes = append(destPrefixes, *prefix)
	}

	r.log.Info("network allocate", "partition", partition.ID)
	var (
		superNetwork metal.Network
	)

	err = r.ds.FindNetwork(&datastore.NetworkSearchQuery{
		PartitionID:     &partition.ID,
		PrivateSuper:    pointer.Pointer(true),
		ParentNetworkID: requestPayload.ParentNetworkID,
	}, &superNetwork)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	if superNetwork.ID == "" {
		r.sendError(request, response, httperrors.BadRequest(fmt.Errorf("no supernetwork found")))
		return
	}
	if len(superNetwork.DefaultChildPrefixLength) == 0 {
		r.sendError(request, response, httperrors.InternalServerError(fmt.Errorf("supernetwork %s has no defaultchildprefixlength specified", superNetwork.ID)))
		return
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
		AddressFamilies:     superNetwork.AddressFamilies,
	}

	// Allow configurable prefix length per AF
	length := superNetwork.DefaultChildPrefixLength
	if len(requestPayload.Length) > 0 {
		for af, l := range requestPayload.Length {
			length[metal.ToAddressFamily(string(af))] = l
		}
	}

	if requestPayload.AddressFamily != nil {
		af := metal.ToAddressFamily(string(*requestPayload.AddressFamily))
		bits, ok := length[af]
		if !ok {
			r.sendError(request, response, httperrors.BadRequest(fmt.Errorf("addressfamiliy %s specified, but no childprefixlength for this addressfamily", *requestPayload.AddressFamily)))
			return
		}
		length = metal.ChildPrefixLength{
			af: bits,
		}
		nwSpec.AddressFamilies = metal.AddressFamilies{af}
	}

	r.log.Info("network allocate", "supernetwork", superNetwork.ID, "defaultchildprefixlength", superNetwork.DefaultChildPrefixLength, "length", length)

	ctx := request.Request.Context()
	nw, err := r.createChildNetwork(ctx, nwSpec, &superNetwork, length)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	consumption, err := r.getNetworkUsage(ctx, nw)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusCreated, v1.NewNetworkResponse(nw, consumption))
}

func (r *networkResource) createChildNetwork(ctx context.Context, nwSpec *metal.Network, parent *metal.Network, childLengths metal.ChildPrefixLength) (*metal.Network, error) {
	vrf, err := acquireRandomVRF(r.ds)
	if err != nil {
		return nil, fmt.Errorf("could not acquire a vrf: %w", err)
	}

	var childPrefixes = metal.Prefixes{}
	for af, childLength := range childLengths {
		childPrefix, err := r.createChildPrefix(ctx, parent.Prefixes, af, childLength)
		if err != nil {
			return nil, err
		}
		if childPrefix == nil {
			return nil, fmt.Errorf("could not allocate child prefix in parent network: %s for addressfamily: %s length:%d", parent.ID, af, childLength)
		}
		childPrefixes = append(childPrefixes, *childPrefix)
	}

	nw := &metal.Network{
		Base: metal.Base{
			Name:        nwSpec.Name,
			Description: nwSpec.Description,
		},
		Prefixes:            childPrefixes,
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
		AddressFamilies:     nwSpec.AddressFamilies,
	}

	err = r.ds.CreateNetwork(nw)
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

	ctx := request.Request.Context()
	for _, prefix := range nw.Prefixes {
		err = r.ipamer.ReleaseChildPrefix(ctx, prefix)
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

	r.send(request, response, http.StatusOK, v1.NewNetworkResponse(nw, &v1.NetworkConsumption{}))
}

func (r *networkResource) updateNetwork(request *restful.Request, response *restful.Response) {
	var (
		requestPayload v1.NetworkUpdateRequest
		forceParam     = request.QueryParameter("force")
	)
	if forceParam == "" {
		forceParam = "false"
	}

	force, err := strconv.ParseBool(forceParam)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}
	err = request.ReadEntity(&requestPayload)
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

	if len(requestPayload.DefaultChildPrefixLength) > 0 && !oldNetwork.PrivateSuper {
		r.sendError(request, response, defaultError(errors.New("defaultchildprefixlength can only be set on privatesuper")))
		return
	}

	var (
		prefixesToBeRemoved metal.Prefixes
		prefixesToBeAdded   metal.Prefixes
	)

	if len(requestPayload.Prefixes) > 0 {
		prefixes, _, err := validatePrefixes(requestPayload.Prefixes)
		if err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}
		newNetwork.Prefixes = prefixes

		prefixesToBeRemoved = oldNetwork.SubtractPrefixes(newNetwork.Prefixes...)

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

		prefixesToBeAdded = newNetwork.SubtractPrefixes(oldNetwork.Prefixes...)
	}

	if len(requestPayload.DestinationPrefixes) > 0 {
		destPrefixes, _, err := validatePrefixes(requestPayload.DestinationPrefixes)
		if err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}
		newNetwork.DestinationPrefixes = destPrefixes
	}

	_, _, addressFamilies, err := validatePrefixesAndAddressFamilies(newNetwork.Prefixes.String(), newNetwork.DestinationPrefixes.String(), oldNetwork.DefaultChildPrefixLength, oldNetwork.PrivateSuper)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}
	newNetwork.AddressFamilies = addressFamilies

	if len(requestPayload.DefaultChildPrefixLength) > 0 {
		for af, defaultChildPrefixLength := range requestPayload.DefaultChildPrefixLength {
			if !slices.Contains(newNetwork.AddressFamilies, af) {
				r.sendError(request, response, defaultError(fmt.Errorf("no addressfamily %q present for defaultchildprefixlength: %d", af, defaultChildPrefixLength)))
				return
			}
		}
		newNetwork.DefaultChildPrefixLength = requestPayload.DefaultChildPrefixLength
	}

	ctx := request.Request.Context()

	for _, p := range prefixesToBeRemoved {
		err := r.ipamer.DeletePrefix(ctx, p)
		if err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}
	}

	for _, p := range prefixesToBeAdded {
		err := r.ipamer.CreatePrefix(ctx, p)
		if err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}
	}

	err = validateAdditionalAnnouncableCIDRs(requestPayload.AdditionalAnnouncableCIDRs, oldNetwork.PrivateSuper)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}
	newNetwork.AdditionalAnnouncableCIDRs = requestPayload.AdditionalAnnouncableCIDRs

	err = validateAdditionalAnnouncableCIDRs(requestPayload.AdditionalAnnouncableCIDRs, oldNetwork.PrivateSuper)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}
	for _, oldcidr := range oldNetwork.AdditionalAnnouncableCIDRs {
		if !force && !slices.Contains(requestPayload.AdditionalAnnouncableCIDRs, oldcidr) {
			r.sendError(request, response, httperrors.BadRequest(fmt.Errorf("you cannot remove %q from additionalannouncablecidrs without force flag set", oldcidr)))
			return
		}
	}

	newNetwork.AdditionalAnnouncableCIDRs = requestPayload.AdditionalAnnouncableCIDRs

	err = r.ds.UpdateNetwork(oldNetwork, &newNetwork)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	usage, err := r.getNetworkUsage(ctx, &newNetwork)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, v1.NewNetworkResponse(&newNetwork, usage))
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

	ctx := request.Request.Context()
	for _, p := range nw.Prefixes {
		err := r.ipamer.DeletePrefix(ctx, p)
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

	r.send(request, response, http.StatusOK, v1.NewNetworkResponse(nw, &v1.NetworkConsumption{}))
}

func (r *networkResource) getNetworkUsage(ctx context.Context, nw *metal.Network) (*v1.NetworkConsumption, error) {
	consumption := &v1.NetworkConsumption{}
	if nw == nil {
		return consumption, nil
	}
	for _, prefix := range nw.Prefixes {
		pfx, err := netip.ParsePrefix(prefix.String())
		if err != nil {
			return nil, err
		}
		af := metal.IPv4AddressFamily
		if pfx.Addr().Is6() {
			af = metal.IPv6AddressFamily
		}
		u, err := r.ipamer.PrefixUsage(ctx, prefix.String())
		if err != nil {
			return nil, err
		}
		switch af {
		case metal.IPv4AddressFamily:
			if consumption.IPv4 == nil {
				consumption.IPv4 = &v1.NetworkUsage{}
			}
			consumption.IPv4.AvailableIPs += u.AvailableIPs
			consumption.IPv4.UsedIPs += u.UsedIPs
			consumption.IPv4.AvailablePrefixes += u.AvailablePrefixes
			consumption.IPv4.UsedPrefixes += u.UsedPrefixes
		case metal.IPv6AddressFamily:
			if consumption.IPv4 == nil {
				consumption.IPv4 = &v1.NetworkUsage{}
			}
			consumption.IPv6.AvailableIPs += u.AvailableIPs
			consumption.IPv6.UsedIPs += u.UsedIPs
			consumption.IPv6.AvailablePrefixes += u.AvailablePrefixes
			consumption.IPv6.UsedPrefixes += u.UsedPrefixes
		}

	}
	return consumption, nil
}

func (r *networkResource) createChildPrefix(ctx context.Context, parentPrefixes metal.Prefixes, af metal.AddressFamily, childLength uint8) (*metal.Prefix, error) {
	var (
		errs []error
	)

	for _, parentPrefix := range parentPrefixes.OfFamily(af) {
		childPrefix, err := r.ipamer.AllocateChildPrefix(ctx, parentPrefix, childLength)
		if err != nil {
			var connectErr *connect.Error
			if errors.As(err, &connectErr) {
				if connectErr.Code() == connect.CodeNotFound {
					continue
				}
			}
			errs = append(errs, err)
			continue
		}

		return childPrefix, nil
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("cannot allocate free child prefix in ipam: %w", errors.Join(errs...))
	}

	return nil, fmt.Errorf("cannot allocate free child prefix in one of the given parent prefixes in ipam: %s", parentPrefixes.String())
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
