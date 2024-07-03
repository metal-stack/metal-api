package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"

	"connectrpc.com/connect"
	"github.com/metal-stack/metal-lib/bus"
	"github.com/metal-stack/metal-lib/pkg/tag"

	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	mdm "github.com/metal-stack/masterdata-api/pkg/client"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/ipam"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/tags"
	"github.com/metal-stack/metal-lib/auditing"

	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"

	goipam "github.com/metal-stack/go-ipam"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-lib/httperrors"
)

type ipResource struct {
	webResource
	ipamer ipam.IPAMer
	mdc    mdm.Client
	actor  *asyncActor
}

// NewIP returns a webservice for ip specific endpoints.
func NewIP(log *slog.Logger, ds *datastore.RethinkStore, ep *bus.Endpoints, ipamer ipam.IPAMer, mdc mdm.Client) (*restful.WebService, error) {
	ir := ipResource{
		webResource: webResource{
			log: log,
			ds:  ds,
		},
		ipamer: ipamer,
		mdc:    mdc,
	}
	var err error
	ir.actor, err = newAsyncActor(log, ep, ds, ipamer)
	if err != nil {
		return nil, fmt.Errorf("cannot create async actor: %w", err)
	}
	return ir.webService(), nil
}

func (r *ipResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path(BasePath + "v1/ip").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"ip"}

	ws.Route(ws.GET("/{id}").
		To(viewer(r.findIP)).
		Operation("findIP").
		Doc("get ip by id").
		Param(ws.PathParameter("id", "identifier of the ip").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.IPResponse{}).
		Returns(http.StatusOK, "OK", v1.IPResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/").
		To(viewer(r.listIPs)).
		Operation("listIPs").
		Doc("get all ips").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]v1.IPResponse{}).
		Returns(http.StatusOK, "OK", []v1.IPResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/find").
		To(viewer(r.findIPs)).
		Operation("findIPs").
		Doc("get all ips that match given properties").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Metadata(auditing.Exclude, true).
		Reads(v1.IPFindRequest{}).
		Writes([]v1.IPResponse{}).
		Returns(http.StatusOK, "OK", []v1.IPResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.DELETE("/free/{id}").
		To(editor(r.freeIP)).
		Operation("freeIP").
		Doc("frees an ip").
		Param(ws.PathParameter("id", "identifier of the ip").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.IPResponse{}).
		Returns(http.StatusOK, "OK", v1.IPResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/").
		To(editor(r.updateIP)).
		Operation("updateIP").
		Doc("updates an ip. if the ip was changed since this one was read, a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.IPUpdateRequest{}).
		Writes(v1.IPResponse{}).
		Returns(http.StatusOK, "OK", v1.IPResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/allocate").
		To(editor(r.allocateIP)).
		Operation("allocateIP").
		Doc("allocate an ip in the given network.").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.IPAllocateRequest{}).
		Writes(v1.IPResponse{}).
		Returns(http.StatusCreated, "Created", v1.IPResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/allocate/{ip}").
		To(editor(r.allocateIP)).
		Operation("allocateSpecificIP").
		Param(ws.PathParameter("ip", "ip to try to allocate").DataType("string")).
		Doc("allocate a specific ip in the given network.").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.IPAllocateRequest{}).
		Writes(v1.IPResponse{}).
		Returns(http.StatusCreated, "Created", v1.IPResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r *ipResource) findIP(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	ip, err := r.ds.FindIPByID(id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, v1.NewIPResponse(ip))
}

func (r *ipResource) listIPs(request *restful.Request, response *restful.Response) {
	ips, err := r.ds.ListIPs()
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	result := []*v1.IPResponse{}
	for i := range ips {
		result = append(result, v1.NewIPResponse(&ips[i]))
	}

	r.send(request, response, http.StatusOK, result)
}

func (r *ipResource) findIPs(request *restful.Request, response *restful.Response) {
	var requestPayload datastore.IPSearchQuery
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	var ips metal.IPs
	err = r.ds.SearchIPs(&requestPayload, &ips)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	result := []*v1.IPResponse{}
	for i := range ips {
		result = append(result, v1.NewIPResponse(&ips[i]))
	}

	r.send(request, response, http.StatusOK, result)
}

func (r *ipResource) freeIP(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	ip, err := r.ds.FindIPByID(id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	err = validateIPDelete(ip)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	err = r.actor.releaseIP(*ip)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, v1.NewIPResponse(ip))
}

func validateIPDelete(ip *metal.IP) error {
	s := ip.GetScope()
	if s == metal.ScopeMachine {
		return errors.New("ip with machine scope can not be deleted")
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
		return errors.New("cannot change type of ip address from static to ephemeral")
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

	return fmt.Errorf("can not use ip of scope %v with scope %v", os, ns)
}

func processTags(ts []string) []string {
	t := tags.New(ts)
	return t.Unique()
}

func (r *ipResource) allocateIP(request *restful.Request, response *restful.Response) {
	specificIP := request.PathParameter("ip")
	var requestPayload v1.IPAllocateRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	if requestPayload.NetworkID == "" {
		r.sendError(request, response, httperrors.BadRequest(errors.New("networkid should not be empty")))
		return
	}
	if requestPayload.ProjectID == "" {
		r.sendError(request, response, httperrors.BadRequest(errors.New("projectid should not be empty")))
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

	nw, err := r.ds.FindNetworkByID(requestPayload.NetworkID)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	p, err := r.mdc.Project().Get(request.Request.Context(), &mdmv1.ProjectGetRequest{Id: requestPayload.ProjectID})
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	if p.Project == nil || p.Project.Meta == nil {
		r.sendError(request, response, defaultError(fmt.Errorf("error retrieving project %q", requestPayload.ProjectID)))
		return
	}

	// for private, unshared networks the project id must be the same
	// for external networks the project id is not checked
	if !nw.Shared && nw.ParentNetworkID != "" && p.Project.Meta.Id != nw.ProjectID {
		r.sendError(request, response, defaultError(fmt.Errorf("can not allocate ip for project %q because network belongs to %q and the network is not shared", p.Project.Meta.Id, nw.ProjectID)))
		return
	}

	tags := requestPayload.Tags
	if requestPayload.MachineID != nil {
		tags = append(tags, metal.IpTag(tag.MachineID, *requestPayload.MachineID))
	}

	tags = processTags(tags)

	// TODO: Following operations should span a database transaction if possible

	var (
		ipAddress    string
		ipParentCidr string
	)

	ctx := request.Request.Context()

	if specificIP == "" {
		ipAddress, ipParentCidr, err = allocateRandomIP(ctx, nw, r.ipamer)
		if err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}
	} else {
		ipAddress, ipParentCidr, err = allocateSpecificIP(ctx, nw, specificIP, r.ipamer)
		if err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}
	}

	r.logger(request).Debug("allocated ip in ipam", "ip", ipAddress, "network", nw.ID)

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
		ProjectID:        p.GetProject().GetMeta().GetId(),
		Type:             ipType,
		Tags:             tags,
	}

	err = r.ds.CreateIP(ip)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusCreated, v1.NewIPResponse(ip))
}

func (r *ipResource) updateIP(request *restful.Request, response *restful.Response) {
	var requestPayload v1.IPUpdateRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	oldIP, err := r.ds.FindIPByID(requestPayload.IPAddress)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	newIP := *oldIP
	if requestPayload.Name != nil {
		newIP.Name = *requestPayload.Name
	}
	if requestPayload.Description != nil {
		newIP.Description = *requestPayload.Description
	}
	if requestPayload.Tags != nil {
		newIP.Tags = requestPayload.Tags
	}
	if requestPayload.Type == metal.Static || requestPayload.Type == metal.Ephemeral {
		newIP.Type = requestPayload.Type
	}

	err = validateIPUpdate(oldIP, &newIP)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}
	newIP.Tags = processTags(newIP.Tags)

	err = r.ds.UpdateIP(oldIP, &newIP)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, v1.NewIPResponse(&newIP))
}

func allocateSpecificIP(ctx context.Context, parent *metal.Network, specificIP string, ipamer ipam.IPAMer) (ipAddress, parentPrefixCidr string, err error) {
	parsedIP, err := netip.ParseAddr(specificIP)
	if err != nil {
		return "", "", fmt.Errorf("unable to parse specific ip: %w", err)
	}

	for _, prefix := range parent.Prefixes {
		pfx, err := netip.ParsePrefix(prefix.String())
		if err != nil {
			return "", "", fmt.Errorf("unable to parse prefix: %w", err)
		}

		if !pfx.Contains(parsedIP) {
			continue
		}
		ipAddress, err = ipamer.AllocateSpecificIP(ctx, prefix, specificIP)
		var connectErr *connect.Error
		if errors.As(err, &connectErr) {
			if connectErr.Code() == connect.CodeAlreadyExists {
				return "", "", metal.Conflict("ip already allocated")
			}
		}
		if err != nil {
			return "", "", err
		}

		return ipAddress, prefix.String(), nil
	}

	return "", "", fmt.Errorf("specific ip not contained in any of the defined prefixes")
}

func allocateRandomIP(ctx context.Context, parent *metal.Network, ipamer ipam.IPAMer) (ipAddress, parentPrefixCidr string, err error) {
	for _, prefix := range parent.Prefixes {
		ipAddress, err = ipamer.AllocateIP(ctx, prefix)
		if err != nil && errors.Is(err, goipam.ErrNoIPAvailable) {
			continue
		}
		if err != nil {
			return "", "", err
		}

		return ipAddress, prefix.String(), nil
	}

	return "", "", metal.Internal("cannot allocate free ip in ipam, no ips left")
}
