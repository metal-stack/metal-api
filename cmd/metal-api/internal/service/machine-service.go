package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/google/uuid"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/headscale"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/issues"
	"github.com/metal-stack/metal-lib/auditing"

	"github.com/avast/retry-go/v4"
	"github.com/aws/aws-sdk-go/service/s3"

	s3server "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/s3client"
	"github.com/metal-stack/security"

	"golang.org/x/crypto/ssh"

	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"github.com/metal-stack/metal-lib/pkg/tag"

	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	mdm "github.com/metal-stack/masterdata-api/pkg/client"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/ipam"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"

	"github.com/metal-stack/metal-lib/bus"
)

type machineResource struct {
	webResource
	bus.Publisher
	ipamer          ipam.IPAMer
	mdc             mdm.Client
	actor           *asyncActor
	s3Client        *s3server.Client
	userGetter      security.UserGetter
	reasonMinLength uint
	headscaleClient *headscale.HeadscaleClient
	ipmiSuperUser   metal.MachineIPMISuperUser
}

// machineAllocationSpec is a specification for a machine allocation
type machineAllocationSpec struct {
	Creator            string
	UUID               string
	Name               string
	Description        string
	Hostname           string
	ProjectID          string
	PartitionID        string
	Machine            *metal.Machine
	Size               *metal.Size
	Image              *metal.Image
	FilesystemLayoutID *string
	SSHPubKeys         []string
	UserData           string
	Tags               []string
	Networks           v1.MachineAllocationNetworks
	IPs                []string
	Role               metal.Role
	VPN                *metal.MachineVPN
	PlacementTags      []string
	EgressRules        []metal.EgressRule
	IngressRules       []metal.IngressRule
	DNSServers         []string
	NTPServers         []string
}

// allocationNetwork is intermediate struct to create machine networks from regular networks during machine allocation
type allocationNetwork struct {
	network *metal.Network
	ips     []metal.IP
	auto    bool

	networkType metal.NetworkType
}

// allocationNetworkMap is a map of allocationNetworks with the network id as the key
type allocationNetworkMap map[string]*allocationNetwork

// The MachineAllocation contains the allocated machine or an error.
type MachineAllocation struct {
	Machine *metal.Machine
	Err     error
}

// An Allocation is a queue of allocated machines. You can read the machines
// to get the next allocated one.
type Allocation <-chan MachineAllocation

// An Allocator is a callback for some piece of code if this wants to read
// allocated machines.
type Allocator func(Allocation) error

// NewMachine returns a webservice for machine specific endpoints.
func NewMachine(
	log *slog.Logger,
	ds *datastore.RethinkStore,
	pub bus.Publisher,
	ep *bus.Endpoints,
	ipamer ipam.IPAMer,
	mdc mdm.Client,
	s3Client *s3server.Client,
	userGetter security.UserGetter,
	reasonMinLength uint,
	headscaleClient *headscale.HeadscaleClient,
	ipmiSuperUser metal.MachineIPMISuperUser,
) (*restful.WebService, error) {
	r := machineResource{
		webResource: webResource{
			log: log,
			ds:  ds,
		},
		Publisher:       pub,
		ipamer:          ipamer,
		mdc:             mdc,
		s3Client:        s3Client,
		userGetter:      userGetter,
		reasonMinLength: reasonMinLength,
		headscaleClient: headscaleClient,
		ipmiSuperUser:   ipmiSuperUser,
	}

	var err error
	r.actor, err = newAsyncActor(log, ep, ds, ipamer)
	if err != nil {
		return nil, fmt.Errorf("cannot create async actor: %w", err)
	}

	return r.webService(), nil
}

// webService creates the webservice endpoint
func (r *machineResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path(BasePath + "v1/machine").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"machine"}

	ws.Route(ws.GET("/{id}").
		To(viewer(r.findMachine)).
		Operation("findMachine").
		Doc("get machine by id").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.MachineResponse{}).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/consolepassword").
		To(editor(r.getMachineConsolePassword)).
		Operation("getMachineConsolePassword").
		Doc("get consolepassword for machine by id").
		Reads(v1.MachineConsolePasswordRequest{}).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.MachineConsolePasswordResponse{}).
		Returns(http.StatusOK, "OK", v1.MachineConsolePasswordResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/").
		To(viewer(r.listMachines)).
		Operation("listMachines").
		Doc("get all known machines").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]v1.MachineResponse{}).
		Returns(http.StatusOK, "OK", []v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/find").
		To(viewer(r.findMachines)).
		Operation("findMachines").
		Doc("find machines by multiple criteria").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Metadata(auditing.Exclude, true).
		Reads(v1.MachineFindRequest{}).
		Writes([]v1.MachineResponse{}).
		Returns(http.StatusOK, "OK", []v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/").
		To(admin(r.updateMachine)).
		Operation("updateMachine").
		Doc("updates a machine. if the machine was changed since this one was read, a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.MachineUpdateRequest{}).
		Writes(v1.MachineResponse{}).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/allocate").
		To(editor(r.allocateMachine)).
		Operation("allocateMachine").
		Doc("allocate a machine").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.MachineAllocateRequest{}).
		Writes(v1.MachineResponse{}).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/state").
		To(editor(r.setMachineState)).
		Operation("setMachineState").
		Doc("set the state of a machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.MachineState{}).
		Writes(v1.MachineResponse{}).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.DELETE("/{id}/free").
		To(editor(r.freeMachine)).
		Operation("freeMachine").
		Doc("free a machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.MachineResponse{}).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.DELETE("/{id}").
		To(admin(r.deleteMachine)).
		Operation("deleteMachine").
		Doc("deletes a machine from the database").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.MachineResponse{}).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/issues").
		To(viewer(r.listIssues)).
		Operation("listIssues").
		Doc("returns the list of issues that exist in the API").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Metadata(auditing.Exclude, true).
		Writes([]v1.MachineIssue{}).
		Returns(http.StatusOK, "OK", []v1.MachineIssue{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/issues/evaluate").
		To(viewer(r.issues)).
		Operation("issues").
		Doc("returns machine issues").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		// is an expensive call so we audit this as well even if it does not change anything
		Reads(v1.MachineIssuesRequest{}).
		Writes([]v1.MachineIssueResponse{}).
		Returns(http.StatusOK, "OK", []v1.MachineIssueResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/ipmi").
		To(editor(r.ipmiReport)).
		Operation("ipmiReport").
		Doc("reports IPMI ip addresses leased by a management server for machines").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.MachineIpmiReports{}).
		Writes(v1.MachineIpmiReportResponse{}).
		Returns(http.StatusOK, "OK", v1.MachineIpmiReportResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/{id}/ipmi").
		To(viewer(r.findIPMIMachine)).
		Operation("findIPMIMachine").
		Doc("returns a machine including the ipmi connection data").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.MachineIPMIResponse{}).
		Returns(http.StatusOK, "OK", v1.MachineIPMIResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/ipmi/find").
		To(viewer(r.findIPMIMachines)).
		Operation("findIPMIMachines").
		Doc("returns machines including the ipmi connection data").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Metadata(auditing.Exclude, true).
		Reads(v1.MachineFindRequest{}).
		Writes([]v1.MachineIPMIResponse{}).
		Returns(http.StatusOK, "OK", []v1.MachineIPMIResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/reinstall").
		To(editor(r.reinstallMachine)).
		Operation("reinstallMachine").
		Doc("reinstall this machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.MachineReinstallRequest{}).
		Writes(v1.MachineResponse{}).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		Returns(http.StatusBadRequest, "Bad Request", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/power/on").
		To(editor(r.machineOn)).
		Operation("machineOn").
		Doc("sends a power-on to the machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.EmptyBody{}).
		Writes(v1.MachineResponse{}).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/power/off").
		To(editor(r.machineOff)).
		Operation("machineOff").
		Doc("sends a power-off to the machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.EmptyBody{}).
		Writes(v1.MachineResponse{}).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/power/reset").
		To(editor(r.machineReset)).
		Operation("machineReset").
		Doc("sends a reset to the machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.EmptyBody{}).
		Writes(v1.MachineResponse{}).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/power/cycle").
		To(editor(r.machineCycle)).
		Operation("machineCycle").
		Doc("sends a power cycle to the machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.EmptyBody{}).
		Writes(v1.MachineResponse{}).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/power/bios").
		To(editor(r.machineBios)).
		Operation("machineBios").
		Doc("boots machine into BIOS").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.EmptyBody{}).
		Writes(v1.MachineResponse{}).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/power/disk").
		To(editor(r.machineDisk)).
		Operation("machineDisk").
		Doc("boots machine from disk").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.EmptyBody{}).
		Writes(v1.MachineResponse{}).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/power/pxe").
		To(editor(r.machinePxe)).
		Operation("machinePxe").
		Doc("boots machine from PXE").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.EmptyBody{}).
		Writes(v1.MachineResponse{}).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/power/chassis-identify-led-on").
		To(editor(r.chassisIdentifyLEDOn)).
		Operation("chassisIdentifyLEDOn").
		Doc("sends a power-on to the chassis identify LED").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Param(ws.QueryParameter("description", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.EmptyBody{}).
		Writes(v1.MachineResponse{}).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/power/chassis-identify-led-off").
		To(editor(r.chassisIdentifyLEDOff)).
		Operation("chassisIdentifyLEDOff").
		Doc("sends a power-off to the chassis identify LED").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Param(ws.QueryParameter("description", "reason why the chassis identify LED has been turned off").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.EmptyBody{}).
		Writes(v1.MachineResponse{}).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/update-firmware/{id}").
		To(admin(r.updateFirmware)).
		Operation("updateFirmware").
		Doc("sends a firmware command to the machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.MachineUpdateFirmwareRequest{}).
		Writes(v1.MachineResponse{}).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r *machineResource) listMachines(request *restful.Request, response *restful.Response) {
	ms, err := r.ds.ListMachines()
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	resp, err := makeMachineResponseList(ms, r.ds)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, resp)
}

func (r *machineResource) findMachine(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	m, err := r.ds.FindMachineByID(id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	resp, err := makeMachineResponse(m, r.ds)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, resp)
}

func (r *machineResource) updateMachine(request *restful.Request, response *restful.Response) {
	var requestPayload v1.MachineUpdateRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	oldMachine, err := r.ds.FindMachineByID(requestPayload.ID)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	if oldMachine.Allocation == nil {
		r.sendError(request, response, httperrors.BadRequest(fmt.Errorf("only allocated machines can be updated")))
		return
	}

	newMachine := *oldMachine

	if requestPayload.Description != nil {
		newMachine.Allocation.Description = *requestPayload.Description
	}

	if len(requestPayload.Tags) > 0 {
		newMachine.Tags = makeMachineTags(&newMachine, requestPayload.Tags)
	}

	if len(requestPayload.SSHPubKeys) > 0 {
		newMachine.Allocation.SSHPubKeys = requestPayload.SSHPubKeys
	}

	err = r.ds.UpdateMachine(oldMachine, &newMachine)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	resp, err := makeMachineResponse(&newMachine, r.ds)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, resp)
}

func (r *machineResource) listIssues(request *restful.Request, response *restful.Response) {
	issues := issues.All()

	var issueResponse []v1.MachineIssue
	for _, issue := range issues {
		issue := issue

		issueResponse = append(issueResponse, v1.MachineIssue{
			ID:          string(issue.Type),
			Severity:    string(issue.Severity),
			Description: issue.Description,
			RefURL:      issue.RefURL,
			Details:     issue.Details,
		})
	}

	r.send(request, response, http.StatusOK, issueResponse)
}

func (r *machineResource) issues(request *restful.Request, response *restful.Response) {
	var requestPayload v1.MachineIssuesRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	var (
		ms = metal.Machines{}

		severity           = issues.SeverityMinor
		only               []issues.Type
		omit               []issues.Type
		lastErrorThreshold = issues.DefaultLastErrorThreshold()
	)

	if requestPayload.Severity != "" {
		severity, err = issues.SeverityFromString(requestPayload.Severity)
		if err != nil {
			r.sendError(request, response, httperrors.BadRequest(err))
			return
		}
	}

	if len(requestPayload.Omit) > 0 {
		for _, o := range requestPayload.Omit {
			o := o

			_, err := issues.NewIssueFromType(o)
			if err != nil {
				r.sendError(request, response, httperrors.BadRequest(err))
				return
			}

			omit = append(omit, o)
		}
	}

	if len(requestPayload.Only) > 0 {
		for _, o := range requestPayload.Only {
			o := o

			_, err := issues.NewIssueFromType(o)
			if err != nil {
				r.sendError(request, response, httperrors.BadRequest(err))
				return
			}

			only = append(only, o)
		}
	}

	if requestPayload.LastErrorThreshold > 0 {
		lastErrorThreshold = requestPayload.LastErrorThreshold
	}

	err = r.ds.SearchMachines(&requestPayload.MachineSearchQuery, &ms)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	ecs, err := r.ds.ListProvisioningEventContainers()
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	machinesWithIssues, err := issues.Find(&issues.Config{
		Machines:           ms,
		EventContainers:    ecs,
		Severity:           severity,
		Only:               only,
		Omit:               omit,
		LastErrorThreshold: lastErrorThreshold,
	})
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	var issueResponse []*v1.MachineIssueResponse
	for _, machineWithIssues := range machinesWithIssues.ToList() {
		machineWithIssues := machineWithIssues

		entry := &v1.MachineIssueResponse{
			MachineID: machineWithIssues.Machine.ID,
		}

		for _, issue := range machineWithIssues.Issues {
			issue := issue

			entry.Issues = append(entry.Issues, string(issue.Type))
		}

		issueResponse = append(issueResponse, entry)
	}

	r.send(request, response, http.StatusOK, issueResponse)
}

func (r *machineResource) getMachineConsolePassword(request *restful.Request, response *restful.Response) {
	var requestPayload v1.MachineConsolePasswordRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	if uint(len(requestPayload.Reason)) < r.reasonMinLength {
		r.sendError(request, response, httperrors.BadRequest(fmt.Errorf("reason must be at least %d characters long", r.reasonMinLength)))
		return
	}

	user, err := r.userGetter.User(request.Request)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	m, err := r.ds.FindMachineByID(requestPayload.ID)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	if m.Allocation == nil {
		r.sendError(request, response, defaultError(fmt.Errorf("machine is not allocated, no consolepassword present")))
		return
	}

	resp := v1.MachineConsolePasswordResponse{
		Common:          v1.Common{Identifiable: v1.Identifiable{ID: m.ID}, Describable: v1.Describable{Name: &m.Name, Description: &m.Description}},
		ConsolePassword: m.Allocation.ConsolePassword,
	}

	r.log.Info("consolepassword requested", "machine", m.ID, "user", user.Name, "email", user.EMail, "tenant", user.Tenant, "reason", requestPayload.Reason)

	r.send(request, response, http.StatusOK, resp)
}

func (r *machineResource) findMachines(request *restful.Request, response *restful.Response) {
	var requestPayload datastore.MachineSearchQuery
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	ms := metal.Machines{}
	err = r.ds.SearchMachines(&requestPayload, &ms)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	resp, err := makeMachineResponseList(ms, r.ds)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, resp)
}

func (r *machineResource) setMachineState(request *restful.Request, response *restful.Response) {
	var requestPayload v1.MachineState
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	machineState, err := metal.MachineStateFrom(requestPayload.Value)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	if machineState != metal.AvailableState && requestPayload.Description == "" {
		// we want a cause why this machine is not available
		r.sendError(request, response, httperrors.BadRequest(errors.New("you must supply a description")))
		return
	}

	user, err := r.userGetter.User(request.Request)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}
	userEMail := user.EMail
	if machineState == metal.AvailableState {
		userEMail = ""
	}

	id := request.PathParameter("id")
	oldMachine, err := r.ds.FindMachineByID(id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	newMachine := *oldMachine

	newMachine.State = metal.MachineState{
		Value:       machineState,
		Description: requestPayload.Description,
		Issuer:      userEMail,
	}

	err = r.ds.UpdateMachine(oldMachine, &newMachine)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	resp, err := makeMachineResponse(&newMachine, r.ds)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, resp)
}

func (r machineResource) findIPMIMachine(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	m, err := r.ds.FindMachineByID(id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	resp, err := makeMachineIPMIResponse(m, r.ds)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, resp)
}

func (r *machineResource) findIPMIMachines(request *restful.Request, response *restful.Response) {
	var requestPayload datastore.MachineSearchQuery
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	ms := metal.Machines{}
	err = r.ds.SearchMachines(&requestPayload, &ms)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	resp, err := makeMachineIPMIResponseList(ms, r.ds)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, resp)
}

// FIXME move to grpc as well
func (r *machineResource) ipmiReport(request *restful.Request, response *restful.Response) {
	var requestPayload v1.MachineIpmiReports
	logger := r.logger(request)

	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	if requestPayload.PartitionID == "" {
		r.sendError(request, response, httperrors.BadRequest(errors.New("partition id must not be empty")))
		return
	}

	p, err := r.ds.FindPartition(requestPayload.PartitionID)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	var ms metal.Machines
	err = r.ds.SearchMachines(&datastore.MachineSearchQuery{
		PartitionID: &p.ID,
	}, &ms)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	known := make(map[string]string)
	for _, m := range ms {
		uuid := m.ID
		if uuid == "" {
			continue
		}
		known[uuid] = m.IPMI.Address
	}
	resp := v1.MachineIpmiReportResponse{
		Updated: []string{},
		Created: []string{},
	}
	// create empty machines for uuids that are not yet known to the metal-api
	const defaultIPMIPort = "623"
	for uuid, report := range requestPayload.Reports {
		if uuid == "" {
			continue
		}
		if _, ok := known[uuid]; ok {
			continue
		}
		m := &metal.Machine{
			Base: metal.Base{
				ID: uuid,
			},
			PartitionID: p.ID,
			IPMI: metal.IPMI{
				Address:     report.BMCIp + ":" + defaultIPMIPort,
				LastUpdated: time.Now(),
			},
		}
		ledstate, err := metal.LEDStateFrom(report.IndicatorLEDState)
		if err == nil {
			m.LEDState = metal.ChassisIdentifyLEDState{
				Value: ledstate,
			}
		} else {
			logger.Error("unable to decode ledstate", "id", uuid, "ledstate", report.IndicatorLEDState, "error", err)
		}
		err = r.ds.CreateMachine(m)
		if err != nil {
			logger.Error("could not create machine", "id", uuid, "ipmi-ip", report.BMCIp, "m", m, "err", err)
			continue
		}
		resp.Created = append(resp.Created, uuid)
	}
	// update machine ipmi data if ipmi ip changed
	for i := range ms {
		oldMachine := ms[i]
		uuid := oldMachine.ID
		if uuid == "" {
			continue
		}
		// if oldmachine.uuid is not part of this update cycle skip it
		report, ok := requestPayload.Reports[uuid]
		if !ok {
			continue
		}
		newMachine := oldMachine

		// Replace host part of ipmi address with the ip from the ipmicatcher
		hostAndPort := strings.Split(oldMachine.IPMI.Address, ":")
		if len(hostAndPort) == 2 {
			newMachine.IPMI.Address = report.BMCIp + ":" + hostAndPort[1]
		} else if len(hostAndPort) < 2 {
			newMachine.IPMI.Address = report.BMCIp + ":" + defaultIPMIPort
		} else {
			logger.Error("not updating ipmi, address is garbage", "id", uuid, "ip", report.BMCIp, "machine", newMachine, "address", newMachine.IPMI.Address)
			continue
		}

		// machine was created by a PXE boot event and has no partition set.
		if oldMachine.PartitionID == "" {
			newMachine.PartitionID = p.ID
		}

		if newMachine.PartitionID != p.ID {
			logger.Error("could not update machine because overlapping id found", "id", uuid, "machine", newMachine, "partition", requestPayload.PartitionID)
			continue
		}

		updateFru(&newMachine, report.FRU)

		if report.BIOSVersion != "" {
			newMachine.BIOS.Version = report.BIOSVersion
		}
		if report.BMCVersion != "" {
			newMachine.IPMI.BMCVersion = report.BMCVersion
		}

		if report.PowerState != "" {
			newMachine.IPMI.PowerState = report.PowerState
		}
		if report.PowerMetric != nil {
			newMachine.IPMI.PowerMetric = &metal.PowerMetric{
				AverageConsumedWatts: report.PowerMetric.AverageConsumedWatts,
				IntervalInMin:        report.PowerMetric.IntervalInMin,
				MaxConsumedWatts:     report.PowerMetric.MaxConsumedWatts,
				MinConsumedWatts:     report.PowerMetric.MinConsumedWatts,
			}
		}

		ledstate, err := metal.LEDStateFrom(report.IndicatorLEDState)
		if err == nil {
			newMachine.LEDState = metal.ChassisIdentifyLEDState{
				Value:       ledstate,
				Description: newMachine.LEDState.Description,
			}
		} else {
			logger.Error("unable to decode ledstate", "id", uuid, "ledstate", report.IndicatorLEDState, "error", err)
		}
		newMachine.IPMI.LastUpdated = time.Now()

		err = r.ds.UpdateMachine(&oldMachine, &newMachine)
		if err != nil {
			logger.Error("could not update machine", "id", uuid, "ip", report.BMCIp, "machine", newMachine, "err", err)
			continue
		}
		resp.Updated = append(resp.Updated, uuid)
	}

	r.send(request, response, http.StatusOK, resp)
}

func updateFru(m *metal.Machine, fru *v1.MachineFru) {
	if fru == nil {
		return
	}

	m.IPMI.Fru.ChassisPartSerial = pointer.SafeDerefOrDefault(fru.ChassisPartSerial, m.IPMI.Fru.ChassisPartSerial)
	m.IPMI.Fru.ChassisPartNumber = pointer.SafeDerefOrDefault(fru.ChassisPartNumber, m.IPMI.Fru.ChassisPartNumber)
	m.IPMI.Fru.BoardMfg = pointer.SafeDerefOrDefault(fru.BoardMfg, m.IPMI.Fru.BoardMfg)
	m.IPMI.Fru.BoardMfgSerial = pointer.SafeDerefOrDefault(fru.BoardMfgSerial, m.IPMI.Fru.BoardMfgSerial)
	m.IPMI.Fru.BoardPartNumber = pointer.SafeDerefOrDefault(fru.BoardPartNumber, m.IPMI.Fru.BoardPartNumber)
	m.IPMI.Fru.ProductManufacturer = pointer.SafeDerefOrDefault(fru.ProductManufacturer, m.IPMI.Fru.ProductManufacturer)
	m.IPMI.Fru.ProductSerial = pointer.SafeDerefOrDefault(fru.ProductSerial, m.IPMI.Fru.ProductSerial)
	m.IPMI.Fru.ProductPartNumber = pointer.SafeDerefOrDefault(fru.ProductPartNumber, m.IPMI.Fru.ProductPartNumber)
}

func (r *machineResource) allocateMachine(request *restful.Request, response *restful.Response) {
	var requestPayload v1.MachineAllocateRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	user, err := r.userGetter.User(request.Request)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	spec, err := createMachineAllocationSpec(r.ds, requestPayload, nil, user)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	m, err := allocateMachine(request.Request.Context(), r.logger(request), r.ds, r.ipamer, spec, r.mdc, r.actor, r.Publisher)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	resp, err := makeMachineResponse(m, r.ds)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, resp)
}

func createMachineAllocationSpec(ds *datastore.RethinkStore, machineRequest v1.MachineAllocateRequest, firewallRequest *v1.FirewallAllocateRequest, user *security.User) (*machineAllocationSpec, error) {
	var uuid string
	if machineRequest.UUID != nil {
		uuid = *machineRequest.UUID
	}
	var name string
	if machineRequest.Name != nil {
		name = *machineRequest.Name
	}
	var description string
	if machineRequest.Description != nil {
		description = *machineRequest.Description
	}
	hostname := "metal"
	if machineRequest.Hostname != nil {
		hostname = *machineRequest.Hostname
	}
	var userdata string
	if machineRequest.UserData != nil {
		userdata = *machineRequest.UserData
	}
	if machineRequest.Networks == nil {
		return nil, errors.New("network ids cannot be nil")
	}
	if len(machineRequest.Networks) <= 0 {
		return nil, errors.New("network ids cannot be empty")
	}

	image, err := ds.FindImage(machineRequest.ImageID)
	if err != nil {
		return nil, err
	}

	var (
		egress  []metal.EgressRule
		ingress []metal.IngressRule
		role    = metal.RoleMachine
	)

	if firewallRequest != nil {
		role = metal.RoleFirewall

		if firewallRequest.FirewallRules != nil {
			for _, ruleSpec := range firewallRequest.FirewallRules.Egress {
				ruleSpec := ruleSpec

				if ruleSpec.Protocol == "" {
					ruleSpec.Protocol = string(metal.ProtocolTCP)
				}

				protocol, err := metal.ProtocolFromString(ruleSpec.Protocol)
				if err != nil {
					return nil, err
				}

				rule := metal.EgressRule{
					Protocol: protocol,
					Ports:    ruleSpec.Ports,
					To:       ruleSpec.To,
					Comment:  ruleSpec.Comment,
				}

				if err := rule.Validate(); err != nil {
					return nil, err
				}

				egress = append(egress, rule)
			}

			for _, ruleSpec := range firewallRequest.FirewallRules.Ingress {
				ruleSpec := ruleSpec

				if ruleSpec.Protocol == "" {
					ruleSpec.Protocol = string(metal.ProtocolTCP)
				}

				protocol, err := metal.ProtocolFromString(ruleSpec.Protocol)
				if err != nil {
					return nil, err
				}

				rule := metal.IngressRule{
					Protocol: protocol,
					Ports:    ruleSpec.Ports,
					To:       ruleSpec.To,
					From:     ruleSpec.From,
					Comment:  ruleSpec.Comment,
				}

				if err := rule.Validate(); err != nil {
					return nil, err
				}

				ingress = append(ingress, rule)
			}
		}
	}

	imageFeatureType := metal.ImageFeatureMachine
	if role == metal.RoleFirewall {
		imageFeatureType = metal.ImageFeatureFirewall
	}

	if !image.HasFeature(imageFeatureType) {
		return nil, fmt.Errorf("given image is not usable for a %s, features: %s", imageFeatureType, image.ImageFeatureString())
	}

	partitionID := machineRequest.PartitionID
	sizeID := machineRequest.SizeID

	if uuid == "" && partitionID == "" {
		return nil, errors.New("when no machine id is given, a partition id must be specified")
	}

	if uuid == "" && sizeID == "" {
		return nil, errors.New("when no machine id is given, a size id must be specified")
	}

	var m *metal.Machine
	// Allocation of a specific machine is requested, therefore size and partition are not given, fetch them
	if uuid != "" {
		m, err = ds.FindMachineByID(uuid)
		if err != nil {
			return nil, fmt.Errorf("uuid given but no machine found with uuid:%s err:%w", uuid, err)
		}
		sizeID = m.SizeID
		partitionID = m.PartitionID
	}

	size, err := ds.FindSize(sizeID)
	if err != nil {
		return nil, fmt.Errorf("size:%s not found err:%w", sizeID, err)
	}

	if len(machineRequest.DNSServers) > 3 {
		return nil, errors.New("please specify a maximum of three dns servers")
	}
	for _, dnsip := range machineRequest.DNSServers {
		_, err := netip.ParseAddr(dnsip)
		if err != nil {
			return nil, fmt.Errorf("IP: %s for DNS server not correct err: %w", dnsip, err)
		}
	}

	if len(machineRequest.NTPServers) <= 3 || len(machineRequest.NTPServers) > 5 {
		return nil, errors.New("please specify a minimum of 3 and a maximum of 5 ntp servers")
	}
	for _, ntpserver := range machineRequest.NTPServers {
		if net.ParseIP(ntpserver) != nil {
			_, err := netip.ParseAddr(ntpserver)
			if err != nil {
				return nil, fmt.Errorf("IP: %s for NTP server not correct err: %w", ntpserver, err)
			}
		} else {
			if !govalidator.IsDNSName(ntpserver) {
				return nil, fmt.Errorf("DNS name: %s for NTP server not correct err: %w", ntpserver, err)
			}
		}
	}

	return &machineAllocationSpec{
		Creator:            user.EMail,
		UUID:               uuid,
		Name:               name,
		Description:        description,
		Hostname:           hostname,
		ProjectID:          machineRequest.ProjectID,
		PartitionID:        partitionID,
		Machine:            m,
		Size:               size,
		Image:              image,
		SSHPubKeys:         machineRequest.SSHPubKeys,
		UserData:           userdata,
		Tags:               machineRequest.Tags,
		Networks:           machineRequest.Networks,
		IPs:                machineRequest.IPs,
		Role:               role,
		FilesystemLayoutID: machineRequest.FilesystemLayoutID,
		PlacementTags:      machineRequest.PlacementTags,
		EgressRules:        egress,
		IngressRules:       ingress,
		DNSServers:         machineRequest.DNSServers,
		NTPServers:         machineRequest.NTPServers,
	}, nil
}

func allocateMachine(ctx context.Context, logger *slog.Logger, ds *datastore.RethinkStore, ipamer ipam.IPAMer, allocationSpec *machineAllocationSpec, mdc mdm.Client, actor *asyncActor, publisher bus.Publisher) (*metal.Machine, error) {
	err := validateAllocationSpec(allocationSpec)
	if err != nil {
		return nil, err
	}

	err = isSizeAndImageCompatible(ds, *allocationSpec.Size, *allocationSpec.Image)
	if err != nil {
		return nil, err
	}

	projectID := allocationSpec.ProjectID
	p, err := mdc.Project().Get(ctx, &mdmv1.ProjectGetRequest{Id: projectID})
	if err != nil {
		return nil, err
	}

	// Check if more machine would be allocated than project quota permits
	if p.GetProject() != nil && p.GetProject().GetQuotas() != nil && p.GetProject().GetQuotas().GetMachine() != nil {
		mq := p.GetProject().GetQuotas().GetMachine()
		maxMachines := mq.GetQuota().GetValue()
		var actualMachines metal.Machines
		err := ds.SearchMachines(&datastore.MachineSearchQuery{AllocationProject: &projectID, AllocationRole: &metal.RoleFirewall}, &actualMachines)
		if err != nil {
			return nil, err
		}
		if len(actualMachines) >= int(maxMachines) {
			return nil, fmt.Errorf("project quota for machines reached max:%d", maxMachines)
		}
	}

	var fsl *metal.FilesystemLayout
	if allocationSpec.FilesystemLayoutID == nil {
		fsls, err := ds.ListFilesystemLayouts()
		if err != nil {
			return nil, err
		}

		fsl, err = fsls.From(allocationSpec.Size.ID, allocationSpec.Image.ID)
		if err != nil {
			return nil, err
		}
	} else {
		fsl, err = ds.FindFilesystemLayout(*allocationSpec.FilesystemLayoutID)
		if err != nil {
			return nil, err
		}
	}

	machineCandidate, err := findMachineCandidate(ctx, ds, allocationSpec)
	if err != nil {
		return nil, err
	}

	var firewallRules *metal.FirewallRules
	if len(allocationSpec.EgressRules) > 0 || len(allocationSpec.IngressRules) > 0 {
		firewallRules = &metal.FirewallRules{
			Egress:  allocationSpec.EgressRules,
			Ingress: allocationSpec.IngressRules,
		}
	}

	// as some fields in the allocation spec are optional, they will now be clearly defined by the machine candidate
	allocationSpec.UUID = machineCandidate.ID

	alloc := &metal.MachineAllocation{
		Creator:         allocationSpec.Creator,
		Created:         time.Now(),
		Name:            allocationSpec.Name,
		Description:     allocationSpec.Description,
		Hostname:        allocationSpec.Hostname,
		Project:         projectID,
		ImageID:         allocationSpec.Image.ID,
		UserData:        allocationSpec.UserData,
		SSHPubKeys:      allocationSpec.SSHPubKeys,
		MachineNetworks: []*metal.MachineNetwork{},
		Role:            allocationSpec.Role,
		VPN:             allocationSpec.VPN,
		FirewallRules:   firewallRules,
		UUID:            uuid.New().String(),
		DNSServers:      allocationSpec.DNSServers,
		NTPServers:      allocationSpec.NTPServers,
	}
	rollbackOnError := func(err error) error {
		if err != nil {
			cleanupMachine := &metal.Machine{
				Base: metal.Base{
					ID: allocationSpec.UUID,
				},
				Allocation: alloc,
			}
			rollbackError := actor.machineNetworkReleaser(cleanupMachine)
			if rollbackError != nil {
				logger.Error("cannot call async machine cleanup", "error", rollbackError)
			}
			old := *machineCandidate
			machineCandidate.Allocation = nil
			machineCandidate.Tags = nil
			machineCandidate.PreAllocated = false

			rollbackError = ds.UpdateMachine(&old, machineCandidate)
			if rollbackError != nil {
				logger.Error("cannot update machinecandidate to reset allocation", "error", rollbackError)
			}
		}
		return err
	}

	err = fsl.Matches(machineCandidate.Hardware)
	if err != nil {
		return nil, rollbackOnError(fmt.Errorf("unable to check for fsl match:%w", err))
	}
	alloc.FilesystemLayout = fsl

	networks, err := gatherNetworks(ds, allocationSpec)
	if err != nil {
		return nil, rollbackOnError(fmt.Errorf("unable to gather networks:%w", err))
	}
	err = makeNetworks(ctx, ds, ipamer, allocationSpec, networks, alloc)
	if err != nil {
		return nil, rollbackOnError(fmt.Errorf("unable to make networks:%w", err))
	}

	// refetch the machine to catch possible updates after dealing with the network...
	machine, err := ds.FindMachineByID(machineCandidate.ID)
	if err != nil {
		return nil, rollbackOnError(fmt.Errorf("unable to find machine:%w", err))
	}
	if machine.Allocation != nil {
		return nil, rollbackOnError(fmt.Errorf("machine %q already allocated", machine.ID))
	}

	old := *machine
	machine.Allocation = alloc
	machine.Tags = makeMachineTags(machine, allocationSpec.Tags)
	machine.PreAllocated = false

	err = ds.UpdateMachine(&old, machine)
	if err != nil {
		return nil, rollbackOnError(fmt.Errorf("error when allocating machine %q, %w", machine.ID, err))
	}

	// TODO: can be removed after metal-core refactoring
	err = publisher.Publish(metal.TopicAllocation.Name, &metal.AllocationEvent{MachineID: machine.ID})
	if err != nil {
		logger.Error("failed to publish machine allocation event, fallback should trigger on metal-hammer", "topic", metal.TopicAllocation.Name, "machineID", machine.ID, "error", err)
	} else {
		logger.Debug("published machine allocation event", "topic", metal.TopicAllocation.Name, "machineID", machine.ID)
	}

	return machine, nil
}

func validateAllocationSpec(allocationSpec *machineAllocationSpec) error {
	if allocationSpec.ProjectID == "" {
		return errors.New("project id must be specified")
	}

	if allocationSpec.Creator == "" {
		return errors.New("creator should be specified")
	}

	if !metal.AllRoles[allocationSpec.Role] {
		return fmt.Errorf("role does not exist: %s", allocationSpec.Role)
	}

	for _, ip := range allocationSpec.IPs {
		if net.ParseIP(ip) == nil {
			return fmt.Errorf("%q is not a valid IP address", ip)
		}
	}

	for _, pubKey := range allocationSpec.SSHPubKeys {
		_, _, _, _, err := ssh.ParseAuthorizedKey([]byte(pubKey))
		if err != nil {
			return fmt.Errorf("invalid public SSH key: %s", pubKey)
		}
	}

	// A firewall must have either IP or Network with auto IP acquire specified.
	if allocationSpec.Role == metal.RoleFirewall {
		if len(allocationSpec.IPs) == 0 && allocationSpec.autoNetworkN() == 0 {
			return errors.New("when no ip is given at least one auto acquire network must be specified")
		}
	}

	if noautoNetN := allocationSpec.noautoNetworkN(); noautoNetN > len(allocationSpec.IPs) {
		return errors.New("missing ip(s) for network(s) without automatic ip allocation")
	}

	return nil
}

func findMachineCandidate(ctx context.Context, ds *datastore.RethinkStore, allocationSpec *machineAllocationSpec) (*metal.Machine, error) {
	var err error
	var machine *metal.Machine
	if allocationSpec.Machine == nil {
		// requesting allocation of an arbitrary ready machine in partition with given size
		machine, err = findWaitingMachine(ctx, ds, allocationSpec)
		if err != nil {
			return nil, err
		}
	} else {
		// requesting allocation of a specific, existing machine
		machine = allocationSpec.Machine
		if machine.Allocation != nil {
			return nil, errors.New("machine is already allocated")
		}
		if allocationSpec.PartitionID != "" && machine.PartitionID != allocationSpec.PartitionID {
			return nil, fmt.Errorf("machine %q is not in the requested partition: %s", machine.ID, allocationSpec.PartitionID)
		}

		if allocationSpec.Size != nil && machine.SizeID != allocationSpec.Size.ID {
			return nil, fmt.Errorf("machine %q does not have the requested size: %s", machine.ID, allocationSpec.Size.ID)
		}
	}
	return machine, err
}

func findWaitingMachine(ctx context.Context, ds *datastore.RethinkStore, allocationSpec *machineAllocationSpec) (*metal.Machine, error) {
	size, err := ds.FindSize(allocationSpec.Size.ID)
	if err != nil {
		return nil, fmt.Errorf("size cannot be found: %w", err)
	}
	partition, err := ds.FindPartition(allocationSpec.PartitionID)
	if err != nil {
		return nil, fmt.Errorf("partition cannot be found: %w", err)
	}

	machine, err := ds.FindWaitingMachine(ctx, allocationSpec.ProjectID, partition.ID, *size, allocationSpec.PlacementTags)
	if err != nil {
		return nil, err
	}
	return machine, nil
}

// makeNetworks creates network entities and ip addresses as specified in the allocation network map.
// created networks are added to the machine allocation directly after their creation. This way, the rollback mechanism
// is enabled to clean up networks that were already created.
func makeNetworks(ctx context.Context, ds *datastore.RethinkStore, ipamer ipam.IPAMer, allocationSpec *machineAllocationSpec, networks allocationNetworkMap, alloc *metal.MachineAllocation) error {
	for _, n := range networks {
		machineNetwork, err := makeMachineNetwork(ctx, ds, ipamer, allocationSpec, n)
		if err != nil {
			return err
		}
		alloc.MachineNetworks = append(alloc.MachineNetworks, machineNetwork)
	}

	// the metal-networker expects to have the same unique ASN on all networks of this machine
	asn, err := acquireASN(ds)
	if err != nil {
		return err
	}
	for _, n := range alloc.MachineNetworks {
		n.ASN = *asn
	}

	return nil
}

func gatherNetworks(ds *datastore.RethinkStore, allocationSpec *machineAllocationSpec) (allocationNetworkMap, error) {
	partition, err := ds.FindPartition(allocationSpec.PartitionID)
	if err != nil {
		return nil, fmt.Errorf("partition cannot be found: %w", err)
	}

	var privateSuperNetworks metal.Networks
	boolTrue := true
	err = ds.SearchNetworks(&datastore.NetworkSearchQuery{PrivateSuper: &boolTrue}, &privateSuperNetworks)
	if err != nil {
		return nil, fmt.Errorf("partition has no private super network: %w", err)
	}

	specNetworks, err := gatherNetworksFromSpec(ds, allocationSpec, partition, privateSuperNetworks)
	if err != nil {
		return nil, err
	}

	var underlayNetwork *allocationNetwork
	if allocationSpec.Role == metal.RoleFirewall {
		underlayNetwork, err = gatherUnderlayNetwork(ds, partition)
		if err != nil {
			return nil, err
		}
	}

	// assemble result
	result := specNetworks
	if underlayNetwork != nil {
		result[underlayNetwork.network.ID] = underlayNetwork
	}

	return result, nil
}

func gatherNetworksFromSpec(ds *datastore.RethinkStore, allocationSpec *machineAllocationSpec, partition *metal.Partition, privateSuperNetworks metal.Networks) (allocationNetworkMap, error) {
	var partitionPrivateSuperNetwork *metal.Network
	for i := range privateSuperNetworks {
		psn := privateSuperNetworks[i]
		if partition.ID == psn.PartitionID {
			partitionPrivateSuperNetwork = &psn
			break
		}
	}
	if partitionPrivateSuperNetwork == nil {
		return nil, fmt.Errorf("partition %s does not have a private super network", partition.ID)
	}

	// what do we have to prevent:
	// - user wants to place his machine in a network that does not belong to the project in which the machine is being placed
	// - user wants a machine with a private network that is not in the partition of the machine
	// - user specifies no private network
	// - user specifies multiple, unshared private networks
	// - user specifies a shared private network in addition to an unshared one for a machine
	// - user specifies administrative networks, i.e. underlay or privatesuper networks
	// - user's private network is specified with noauto but no specific IPs are given: this would yield a machine with no ip address

	specNetworks := make(map[string]*allocationNetwork)
	var primaryPrivateNetwork *allocationNetwork
	var privateNetworks []*allocationNetwork
	var privateSharedNetworks []*allocationNetwork

	for _, networkSpec := range allocationSpec.Networks {
		auto := true
		if networkSpec.AutoAcquireIP != nil {
			auto = *networkSpec.AutoAcquireIP
		}

		network, err := ds.FindNetworkByID(networkSpec.NetworkID)
		if err != nil {
			return nil, err
		}

		if network.Underlay {
			return nil, fmt.Errorf("underlay networks are not allowed to be set explicitly: %s", network.ID)
		}
		if network.PrivateSuper {
			return nil, fmt.Errorf("private super networks are not allowed to be set explicitly: %s", network.ID)
		}

		n := &allocationNetwork{
			network:     network,
			auto:        auto,
			ips:         []metal.IP{},
			networkType: metal.External,
		}

		for _, privateSuperNetwork := range privateSuperNetworks {
			if network.ParentNetworkID != privateSuperNetwork.ID {
				continue
			}
			if network.Shared {
				n.networkType = metal.PrivateSecondaryShared
				privateSharedNetworks = append(privateSharedNetworks, n)
			} else {
				if primaryPrivateNetwork != nil {
					return nil, errors.New("multiple private networks are specified but there must be only one primary private network that must not be shared")
				}
				n.networkType = metal.PrivatePrimaryUnshared
				primaryPrivateNetwork = n
			}
			privateNetworks = append(privateNetworks, n)
		}

		specNetworks[network.ID] = n
	}

	if len(specNetworks) != len(allocationSpec.Networks) {
		return nil, errors.New("given network ids are not unique")
	}

	if len(privateNetworks) == 0 {
		return nil, errors.New("no private network given")
	}

	// if there is no unshared private network we try to determine a shared one as primary
	if primaryPrivateNetwork == nil {
		// this means that this is a machine of a shared private network
		// this is an exception where the primary private network is a shared one.
		// it must be the only private network
		if len(privateSharedNetworks) == 0 {
			return nil, errors.New("no private shared network found that could be used as primary private network")
		}
		if len(privateSharedNetworks) > 1 {
			return nil, errors.New("machines and firewalls are not allowed to be placed into multiple private, shared networks (firewall needs an unshared private network and machines may only reside in one private network)")
		}

		primaryPrivateNetwork = privateSharedNetworks[0]
		primaryPrivateNetwork.networkType = metal.PrivatePrimaryShared
	}

	if allocationSpec.Role == metal.RoleMachine && len(privateNetworks) > 1 {
		return nil, errors.New("machines are not allowed to be placed into multiple private networks")
	}

	if primaryPrivateNetwork.network.ProjectID != allocationSpec.ProjectID {
		return nil, errors.New("the given private network does not belong to the project, which is not allowed")
	}

	for _, ipString := range allocationSpec.IPs {
		ip, err := ds.FindIPByID(ipString)
		if err != nil {
			return nil, err
		}
		if ip.ProjectID != allocationSpec.ProjectID {
			return nil, fmt.Errorf("given ip %q with project id %q does not belong to the project of this allocation: %s", ip.IPAddress, ip.ProjectID, allocationSpec.ProjectID)
		}
		network, ok := specNetworks[ip.NetworkID]
		if !ok {
			return nil, fmt.Errorf("given ip %q is not in any of the given networks, which is required", ip.IPAddress)
		}
		s := ip.GetScope()
		if s != metal.ScopeMachine && s != metal.ScopeProject {
			return nil, fmt.Errorf("given ip %q is not available for direct attachment to machine because it is already in use", ip.IPAddress)
		}

		network.auto = false
		network.ips = append(network.ips, *ip)
	}

	for _, pn := range privateNetworks {
		if pn.network.PartitionID != partitionPrivateSuperNetwork.PartitionID {
			return nil, fmt.Errorf("private network %q must be located in the partition where the machine is going to be placed", pn.network.ID)
		}

		if !pn.auto && len(pn.ips) == 0 {
			return nil, fmt.Errorf("the private network %q has no auto ip acquisition, but no suitable IPs were provided, which would lead into a machine having no ip address", pn.network.ID)
		}
	}

	return specNetworks, nil
}

func gatherUnderlayNetwork(ds *datastore.RethinkStore, partition *metal.Partition) (*allocationNetwork, error) {
	boolTrue := true
	var underlays metal.Networks
	err := ds.SearchNetworks(&datastore.NetworkSearchQuery{PartitionID: &partition.ID, Underlay: &boolTrue}, &underlays)
	if err != nil {
		return nil, err
	}
	if len(underlays) == 0 {
		return nil, fmt.Errorf("no underlay found in the given partition:%s", partition.ID)
	}
	if len(underlays) > 1 {
		return nil, fmt.Errorf("more than one underlay network in partition %s in the database, which should not be the case", partition.ID)
	}
	underlay := &underlays[0]

	return &allocationNetwork{
		network:     underlay,
		auto:        true,
		networkType: metal.Underlay,
	}, nil
}

func makeMachineNetwork(ctx context.Context, ds *datastore.RethinkStore, ipamer ipam.IPAMer, allocationSpec *machineAllocationSpec, n *allocationNetwork) (*metal.MachineNetwork, error) {
	if n.auto {
		ipAddress, ipParentCidr, err := allocateRandomIP(ctx, n.network, ipamer)
		if err != nil {
			return nil, fmt.Errorf("unable to allocate an ip in network: %s %w", n.network.ID, err)
		}
		ip := &metal.IP{
			IPAddress:        ipAddress,
			ParentPrefixCidr: ipParentCidr,
			Name:             allocationSpec.Name,
			Description:      "autoassigned",
			NetworkID:        n.network.ID,
			Type:             metal.Ephemeral,
			ProjectID:        allocationSpec.ProjectID,
		}
		ip.AddMachineId(allocationSpec.UUID)
		err = ds.CreateIP(ip)
		if err != nil {
			return nil, err
		}
		n.ips = append(n.ips, *ip)
	}

	// from the makeNetworks call, a lot of ips might be set in this network
	// add a machine tag to all of them
	ipAddresses := []string{}
	for i := range n.ips {
		ip := n.ips[i]
		newIP := ip
		newIP.AddMachineId(allocationSpec.UUID)
		err := ds.UpdateIP(&ip, &newIP)
		if err != nil {
			return nil, err
		}
		ipAddresses = append(ipAddresses, ip.IPAddress)
	}

	machineNetwork := metal.MachineNetwork{
		NetworkID:           n.network.ID,
		Prefixes:            n.network.Prefixes.String(),
		IPs:                 ipAddresses,
		DestinationPrefixes: n.network.DestinationPrefixes.String(),
		PrivatePrimary:      n.networkType.PrivatePrimary,
		Private:             n.networkType.Private,
		Shared:              n.networkType.Shared,
		Underlay:            n.networkType.Underlay,
		Nat:                 n.network.Nat,
		Vrf:                 n.network.Vrf,
	}

	return &machineNetwork, nil
}

// makeMachineTags constructs the tags of the machine.
// following tags are added in the following precedence (from lowest to highest in case of duplication):
// - user given tags (from allocation spec)
// - system tags (immutable information from the metal-api that are useful for the end user, e.g. machine rack and chassis)
func makeMachineTags(m *metal.Machine, userTags []string) []string {
	labels := make(map[string]string)

	// as user labels are given as an array, we need to figure out if label-like tags were provided.
	// otherwise the user could provide confusing information like:
	// - machine.metal-stack.io/chassis=123
	// - machine.metal-stack.io/chassis=789
	userLabels := make(map[string]string)
	actualUserTags := []string{}
	for _, tag := range userTags {
		if strings.Contains(tag, "=") {
			parts := strings.SplitN(tag, "=", 2)
			userLabels[parts[0]] = parts[1]
		} else {
			actualUserTags = append(actualUserTags, tag)
		}
	}
	for k, v := range userLabels {
		labels[k] = v
	}

	for k, v := range makeMachineSystemLabels(m) {
		labels[k] = v
	}

	tags := actualUserTags
	for k, v := range labels {
		tags = append(tags, fmt.Sprintf("%s=%s", k, v))
	}

	return uniqueTags(tags)
}

func makeMachineSystemLabels(m *metal.Machine) map[string]string {
	labels := make(map[string]string)
	for _, n := range m.Allocation.MachineNetworks {
		if n.Private {
			if n.ASN != 0 {
				labels[tag.MachineNetworkPrimaryASN] = strconv.FormatInt(int64(n.ASN), 10)
				break
			}
		}
	}
	if m.RackID != "" {
		labels[tag.MachineRack] = m.RackID
	}
	if m.IPMI.Fru.ChassisPartSerial != "" {
		labels[tag.MachineChassis] = m.IPMI.Fru.ChassisPartSerial
	}
	return labels
}

// uniqueTags the last added tags will be kept!
func uniqueTags(tags []string) []string {
	tagSet := make(map[string]bool)
	for _, t := range tags {
		tagSet[t] = true
	}
	uniqueTags := []string{}
	for k := range tagSet {
		uniqueTags = append(uniqueTags, k)
	}
	return uniqueTags
}

func (r machineResource) freeMachine(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	m, err := r.ds.FindMachineByID(id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	logger := r.logger(request)

	err = publishMachineCmd(logger, m, r.Publisher, metal.ChassisIdentifyLEDOffCmd)
	if err != nil {
		logger.Error("unable to publish machine command", "command", string(metal.ChassisIdentifyLEDOffCmd), "machineID", m.ID, "error", err)
	}

	err = r.actor.freeMachine(request.Request.Context(), r.Publisher, m, r.headscaleClient, logger)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	resp, err := makeMachineResponse(m, r.ds)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, resp)

	ev := metal.ProvisioningEvent{
		Time:    time.Now(),
		Event:   metal.ProvisioningEventMachineReclaim,
		Message: "free machine called",
	}

	ctx := request.Request.Context()
	_, err = r.ds.ProvisioningEventForMachine(ctx, logger, &ev, id)
	if err != nil {
		r.log.Error("error sending provisioning event after machine free", "error", err)
	}
}

func (r *machineResource) deleteMachine(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	m, err := r.ds.FindMachineByID(id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	if m.Allocation != nil {
		r.sendError(request, response, defaultError(errors.New("cannot delete machine that is allocated")))
		return
	}

	ec, err := r.ds.FindProvisioningEventContainer(id)

	// when there's no event container, we delete the machine anyway
	if err != nil && !metal.IsNotFound(err) {
		r.sendError(request, response, defaultError(err))
		return
	}
	if err == nil && ec.Liveliness != metal.MachineLivelinessDead {
		r.sendError(request, response, defaultError(errors.New("can only delete dead machines")))
		return
	}

	switches, err := r.ds.SearchSwitchesConnectedToMachine(m)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	for i := range switches {
		old := switches[i]
		_, ok := old.MachineConnections[m.ID]
		if !ok {
			continue
		}

		newIP := old
		newIP.MachineConnections = metal.ConnectionMap{}

		for id, connection := range old.MachineConnections {
			newIP.MachineConnections[id] = connection
		}
		delete(newIP.MachineConnections, m.ID)

		err = r.ds.UpdateSwitch(&old, &newIP)
		if err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}

	}

	err = r.ds.DeleteMachine(m)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	resp, err := makeMachineResponse(m, r.ds)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, resp)
}

// reinstallMachine reinstalls the requested machine with given image by either allocating
// the machine if not yet allocated or not modifying any other allocation parameter than 'ImageID'
// and 'Reinstall' set to true.
// If the given image ID is nil, it deletes the machine instead.
func (r *machineResource) reinstallMachine(request *restful.Request, response *restful.Response) {
	var requestPayload v1.MachineReinstallRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	id := request.PathParameter("id")
	m, err := r.ds.FindMachineByID(id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	logger := r.logger(request)

	if m.Allocation != nil && m.State.Value != metal.LockedState {
		old := *m

		if m.Allocation.FilesystemLayout == nil {
			fsls, err := r.ds.ListFilesystemLayouts()
			if err != nil {
				r.sendError(request, response, defaultError(err))
				return
			}

			fsl, err := fsls.From(m.SizeID, m.Allocation.ImageID)
			if err != nil {
				r.sendError(request, response, defaultError(err))
				return
			}

			m.Allocation.FilesystemLayout = fsl
		}

		if !m.Allocation.FilesystemLayout.IsReinstallable() {
			r.sendError(request, response, defaultError(fmt.Errorf("filesystemlayout:%s is not reinstallable, abort reinstallation", m.Allocation.FilesystemLayout.ID)))
			return
		}
		m.Allocation.Reinstall = true
		m.Allocation.ImageID = requestPayload.ImageID

		resp, err := makeMachineResponse(m, r.ds)
		if err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}

		if resp.Allocation.Image != nil {
			err = r.ds.UpdateMachine(&old, m)
			if err != nil {
				r.sendError(request, response, defaultError(err))
				return
			}

			logger.Info("marked machine to get reinstalled", "machineID", m.ID)

			err = deleteVRFSwitches(r.ds, m, logger)
			if err != nil {
				r.sendError(request, response, defaultError(err))
				return
			}

			err = publishDeleteEvent(r.Publisher, m, logger)
			if err != nil {
				r.sendError(request, response, defaultError(err))
				return
			}

			err = publishMachineCmd(logger, m, r.Publisher, metal.MachineReinstallCmd)
			if err != nil {
				logger.Error("unable to publish machine command", "command", string(metal.MachineReinstallCmd), "machineID", m.ID, "error", err)
			}

			r.send(request, response, http.StatusOK, resp)

			return
		}
	}

	r.sendError(request, response, httperrors.BadRequest(errors.New("machine either locked, not allocated yet or invalid image ID specified")))
}

func deleteVRFSwitches(ds *datastore.RethinkStore, m *metal.Machine, logger *slog.Logger) error {
	logger.Info("set VRF at switch", "machineID", m.ID)
	err := retry.Do(
		func() error {
			_, err := ds.SetVrfAtSwitches(m, "")
			return err
		},
		retry.Attempts(10),
		retry.RetryIf(func(err error) bool {
			return metal.IsConflict(err)
		}),
		retry.DelayType(retry.CombineDelay(retry.BackOffDelay, retry.RandomDelay)),
		retry.LastErrorOnly(true),
	)
	if err != nil {
		logger.Error("cannot delete vrf switches", "machineID", m.ID, "error", err)
		return fmt.Errorf("cannot delete vrf switches: %w", err)
	}
	return nil
}

func publishDeleteEvent(publisher bus.Publisher, m *metal.Machine, logger *slog.Logger) error {
	logger.Info("publish machine delete event", "machineID", m.ID)
	deleteEvent := metal.MachineEvent{Type: metal.DELETE, OldMachineID: m.ID, Cmd: &metal.MachineExecCommand{TargetMachineID: m.ID, IPMI: &m.IPMI}}
	err := publisher.Publish(metal.TopicMachine.GetFQN(m.PartitionID), deleteEvent)
	if err != nil {
		logger.Error("cannot publish delete event", "machineID", m.ID, "error", err)
		return fmt.Errorf("cannot publish delete event: %w", err)
	}
	return nil
}

// MachineLiveliness evaluates whether machines are still alive or if they have died
func MachineLiveliness(ds *datastore.RethinkStore, logger *slog.Logger) error {
	logger.Info("machine liveliness was requested")

	machines, err := ds.ListMachines()
	if err != nil {
		return err
	}

	unknown := 0
	alive := 0
	dead := 0
	errs := 0
	for _, m := range machines {
		lvlness, err := evaluateMachineLiveliness(ds, m)
		if err != nil {
			logger.Error("cannot update liveliness", "error", err, "machine", m)
			errs++
			// fall through, so the rest of the machines is getting evaluated
		}
		switch lvlness {
		case metal.MachineLivelinessAlive:
			alive++
		case metal.MachineLivelinessDead:
			dead++
		case metal.MachineLivelinessUnknown:
			unknown++
		}
	}

	logger.Info("machine liveliness evaluated", "alive", alive, "dead", dead, "unknown", unknown, "errors", errs)

	return nil
}

func evaluateMachineLiveliness(ds *datastore.RethinkStore, m metal.Machine) (metal.MachineLiveliness, error) {
	provisioningEvents, err := ds.FindProvisioningEventContainer(m.ID)
	if err != nil {
		// we have no provisioning events... we cannot tell
		return metal.MachineLivelinessUnknown, fmt.Errorf("no provisioning event container found for machine: %s", m.ID)
	}

	old := *provisioningEvents

	if provisioningEvents.LastEventTime != nil {
		if time.Since(*provisioningEvents.LastEventTime) > metal.MachineDeadAfter {
			if m.Allocation != nil {
				// the machine is either dead or the customer did turn off the phone home service
				provisioningEvents.Liveliness = metal.MachineLivelinessUnknown
			} else {
				// the machine is just dead
				provisioningEvents.Liveliness = metal.MachineLivelinessDead
			}
		} else {
			provisioningEvents.Liveliness = metal.MachineLivelinessAlive
		}

		err = ds.UpdateProvisioningEventContainer(&old, provisioningEvents)
		if err != nil {
			return provisioningEvents.Liveliness, err
		}
	}

	return provisioningEvents.Liveliness, nil
}

// ResurrectMachines attempts to resurrect machines that are obviously dead
func ResurrectMachines(ctx context.Context, ds *datastore.RethinkStore, publisher bus.Publisher, ep *bus.Endpoints, ipamer ipam.IPAMer, headscaleClient *headscale.HeadscaleClient, logger *slog.Logger) error {
	logger.Info("machine resurrection was requested")

	machines, err := ds.ListMachines()
	if err != nil {
		return err
	}

	act, err := newAsyncActor(logger, ep, ds, ipamer)
	if err != nil {
		return err
	}

	for i := range machines {
		m := machines[i]
		if m.Allocation != nil {
			continue
		}

		provisioningEvents, err := ds.FindProvisioningEventContainer(m.ID)
		if err != nil {
			// we have no provisioning events... we cannot tell
			logger.Debug("no provisioningEvents found for resurrection", "machineID", m.ID, "error", err)
			continue
		}

		if provisioningEvents.LastEventTime == nil {
			continue
		}

		if provisioningEvents.Liveliness == metal.MachineLivelinessDead && time.Since(*provisioningEvents.LastEventTime) > metal.MachineResurrectAfter {
			logger.Info("resurrecting dead machine", "machineID", m.ID, "liveliness", provisioningEvents.Liveliness, "since", time.Since(*provisioningEvents.LastEventTime).String())
			err = act.freeMachine(ctx, publisher, &m, headscaleClient, logger)
			if err != nil {
				logger.Error("error during machine resurrection", "machineID", m.ID, "error", err)
			}
			continue
		}

		if provisioningEvents.FailedMachineReclaim {
			logger.Info("resurrecting machine with failed reclaim", "machineID", m.ID, "liveliness", provisioningEvents.Liveliness, "since", time.Since(*provisioningEvents.LastEventTime).String())
			err = act.freeMachine(ctx, publisher, &m, headscaleClient, logger)
			if err != nil {
				logger.Error("error during machine resurrection", "machineID", m.ID, "error", err)
			}
			continue
		}
	}

	logger.Info("finished machine resurrection")

	return nil
}

func (r *machineResource) machineOn(request *restful.Request, response *restful.Response) {
	r.machineCmd(request.Request.Context(), metal.MachineOnCmd, request, response)
}

func (r *machineResource) machineOff(request *restful.Request, response *restful.Response) {
	r.machineCmd(request.Request.Context(), metal.MachineOffCmd, request, response)
}

func (r *machineResource) machineReset(request *restful.Request, response *restful.Response) {
	r.machineCmd(request.Request.Context(), metal.MachineResetCmd, request, response)
}

func (r *machineResource) machineCycle(request *restful.Request, response *restful.Response) {
	r.machineCmd(request.Request.Context(), metal.MachineCycleCmd, request, response)
}

func (r *machineResource) machineBios(request *restful.Request, response *restful.Response) {
	r.machineCmd(request.Request.Context(), metal.MachineBiosCmd, request, response)
}

func (r *machineResource) machineDisk(request *restful.Request, response *restful.Response) {
	r.machineCmd(request.Request.Context(), metal.MachineDiskCmd, request, response)
}

func (r *machineResource) machinePxe(request *restful.Request, response *restful.Response) {
	r.machineCmd(request.Request.Context(), metal.MachinePxeCmd, request, response)
}

func (r *machineResource) chassisIdentifyLEDOn(request *restful.Request, response *restful.Response) {
	r.machineCmd(request.Request.Context(), metal.ChassisIdentifyLEDOnCmd, request, response)
}

func (r *machineResource) chassisIdentifyLEDOff(request *restful.Request, response *restful.Response) {
	r.machineCmd(request.Request.Context(), metal.ChassisIdentifyLEDOffCmd, request, response)
}

func (r *machineResource) updateFirmware(request *restful.Request, response *restful.Response) {
	if r.s3Client == nil {
		r.sendError(request, response, httperrors.InternalServerError(featureDisabledErr))
		return
	}

	var p v1.MachineUpdateFirmwareRequest
	err := request.ReadEntity(&p)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	id := request.PathParameter("id")
	m, f, err := getFirmware(r.ds, id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	alreadyInstalled := false
	switch p.Kind {
	case metal.FirmwareBIOS:
		alreadyInstalled = f.BiosVersion == p.Revision
	case metal.FirmwareBMC:
		alreadyInstalled = f.BmcVersion == p.Revision
	}
	if alreadyInstalled {
		r.sendError(request, response, defaultError(fmt.Errorf("machine's %s version is already equal %s", p.Kind, p.Revision)))
		return
	}

	rr, err := getFirmwareRevisions(r.s3Client, p.Kind, f.Vendor, f.Board)
	if err != nil {
		r.sendError(request, response, httperrors.InternalServerError(err))
		return
	}

	notAvailable := true
	for _, rev := range rr {
		if rev == p.Revision {
			notAvailable = false
			break
		}
	}
	if notAvailable {
		r.sendError(request, response, defaultError(fmt.Errorf("machine's %s firmware in version %s is not available", p.Kind, p.Revision)))
		return
	}

	key := fmt.Sprintf("%s/%s/%s/%s", p.Kind, strings.ToLower(f.Vendor), strings.ToUpper(f.Board), p.Revision)
	req, _ := r.s3Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: &r.s3Client.FirmwareBucket,
		Key:    &key,
	})
	downloadableURL, err := req.Presign(2 * time.Hour)
	if err != nil {
		r.sendError(request, response, httperrors.InternalServerError(err))
		return
	}

	evt := metal.MachineEvent{
		Type: metal.COMMAND,
		Cmd: &metal.MachineExecCommand{
			Command:         metal.UpdateFirmwareCmd,
			TargetMachineID: m.ID,
			IPMI:            &m.IPMI,
			FirmwareUpdate: &metal.FirmwareUpdate{
				Kind: p.Kind,
				URL:  downloadableURL,
			},
		},
	}

	r.logger(request).Info("publish event", "event", evt, "command", *evt.Cmd)
	err = r.Publish(metal.TopicMachine.GetFQN(m.PartitionID), evt)
	if err != nil {
		r.sendError(request, response, httperrors.InternalServerError(err))
		return
	}

	resp, err := makeMachineResponse(m, r.ds)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, resp)
}

func (r *machineResource) machineCmd(ctx context.Context, cmd metal.MachineCommand, request *restful.Request, response *restful.Response) {
	logger := r.logger(request)

	id := request.PathParameter("id")
	description := request.QueryParameter("description")

	newMachine, err := r.ds.FindMachineByID(id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	old := *newMachine
	needsUpdate := false
	switch cmd { // nolint:exhaustive
	case metal.MachineResetCmd, metal.MachineOffCmd, metal.MachineCycleCmd:
		ev := metal.ProvisioningEvent{
			Time:    time.Now(),
			Event:   metal.ProvisioningEventPlannedReboot,
			Message: string(cmd),
		}
		_, err = r.ds.ProvisioningEventForMachine(ctx, logger, &ev, id)
		if err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}
	case metal.ChassisIdentifyLEDOnCmd:
		newMachine.LEDState = metal.ChassisIdentifyLEDState{
			Value:       metal.LEDStateOn,
			Description: description,
		}
		needsUpdate = true
	case metal.ChassisIdentifyLEDOffCmd:
		newMachine.LEDState = metal.ChassisIdentifyLEDState{
			Value:       metal.LEDStateOff,
			Description: description,
		}
		needsUpdate = true
	}

	if needsUpdate {
		err = r.ds.UpdateMachine(&old, newMachine)
		if err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}
	}

	if newMachine.IPMI.User == "" && r.ipmiSuperUser.IsEnabled() {
		// when removing a machine from the database, the metal-bmc will loose the ability
		// to manage the machine after it reported it back to API.
		//
		// to mitigate this scenario, we use the super user as a fallback.
		newMachine.IPMI.User = r.ipmiSuperUser.User()
		newMachine.IPMI.Password = r.ipmiSuperUser.Password()
	}

	err = publishMachineCmd(logger, newMachine, r.Publisher, cmd)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	resp, err := makeMachineResponse(newMachine, r.ds)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, resp)
}

func publishMachineCmd(logger *slog.Logger, m *metal.Machine, publisher bus.Publisher, cmd metal.MachineCommand) error {
	evt := metal.MachineEvent{
		Type: metal.COMMAND,
		Cmd: &metal.MachineExecCommand{
			Command:         cmd,
			TargetMachineID: m.ID,
			IPMI:            &m.IPMI,
		},
	}

	logger.Info("publish event", "event", evt, "command", *evt.Cmd)
	err := publisher.Publish(metal.TopicMachine.GetFQN(m.PartitionID), evt)
	if err != nil {
		return err
	}

	return nil
}

func makeMachineResponse(m *metal.Machine, ds *datastore.RethinkStore) (*v1.MachineResponse, error) {
	s, p, i, ec, err := findMachineReferencedEntities(m, ds)
	if err != nil {
		return nil, err
	}
	return v1.NewMachineResponse(m, s, p, i, ec), nil
}

func makeMachineResponseList(ms metal.Machines, ds *datastore.RethinkStore) ([]*v1.MachineResponse, error) {
	sMap, pMap, iMap, ecMap, err := getMachineReferencedEntityMaps(ds)
	if err != nil {
		return nil, err
	}

	result := []*v1.MachineResponse{}

	for index := range ms {
		var s *metal.Size
		if ms[index].SizeID != "" {
			sizeEntity := sMap[ms[index].SizeID]
			s = &sizeEntity
		}
		var p *metal.Partition
		if ms[index].PartitionID != "" {
			partitionEntity := pMap[ms[index].PartitionID]
			p = &partitionEntity
		}
		var i *metal.Image
		if ms[index].Allocation != nil {
			if ms[index].Allocation.ImageID != "" {
				imageEntity := iMap[ms[index].Allocation.ImageID]
				i = &imageEntity
			}
		}
		ec := ecMap[ms[index].ID]
		result = append(result, v1.NewMachineResponse(&ms[index], s, p, i, &ec))
	}

	return result, nil
}

func makeMachineIPMIResponse(m *metal.Machine, ds *datastore.RethinkStore) (*v1.MachineIPMIResponse, error) {
	s, p, i, ec, err := findMachineReferencedEntities(m, ds)
	if err != nil {
		return nil, err
	}
	return v1.NewMachineIPMIResponse(m, s, p, i, ec), nil
}

func makeMachineIPMIResponseList(ms metal.Machines, ds *datastore.RethinkStore) ([]*v1.MachineIPMIResponse, error) {
	sMap, pMap, iMap, ecMap, err := getMachineReferencedEntityMaps(ds)
	if err != nil {
		return nil, err
	}

	result := []*v1.MachineIPMIResponse{}

	for index := range ms {
		var s *metal.Size
		if ms[index].SizeID != "" {
			sizeEntity := sMap[ms[index].SizeID]
			s = &sizeEntity
		}
		var p *metal.Partition
		if ms[index].PartitionID != "" {
			partitionEntity := pMap[ms[index].PartitionID]
			p = &partitionEntity
		}
		var i *metal.Image
		if ms[index].Allocation != nil {
			if ms[index].Allocation.ImageID != "" {
				imageEntity := iMap[ms[index].Allocation.ImageID]
				i = &imageEntity
			}
		}
		ec := ecMap[ms[index].ID]
		result = append(result, v1.NewMachineIPMIResponse(&ms[index], s, p, i, &ec))
	}

	return result, nil
}

func findMachineReferencedEntities(m *metal.Machine, ds *datastore.RethinkStore) (*metal.Size, *metal.Partition, *metal.Image, *metal.ProvisioningEventContainer, error) {
	var err error

	var s *metal.Size
	if m.SizeID != "" {
		if m.SizeID == metal.UnknownSize().GetID() {
			s = metal.UnknownSize()
		} else {
			s, err = ds.FindSize(m.SizeID)
			if err != nil {
				return nil, nil, nil, nil, fmt.Errorf("error finding size %q for machine %q: %w", m.SizeID, m.ID, err)
			}
		}
	}

	var p *metal.Partition
	if m.PartitionID != "" {
		p, err = ds.FindPartition(m.PartitionID)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("error finding partition %q for machine %q: %w", m.PartitionID, m.ID, err)
		}
	}

	var i *metal.Image
	if m.Allocation != nil {
		if m.Allocation.ImageID != "" {
			i, err = ds.GetImage(m.Allocation.ImageID)
			if err != nil {
				return nil, nil, nil, nil, fmt.Errorf("error finding image %q for machine %q: %w", m.Allocation.ImageID, m.ID, err)
			}
		}
	}

	var ec *metal.ProvisioningEventContainer
	try, err := ds.FindProvisioningEventContainer(m.ID)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("error finding provisioning event container for machine %q: %w", m.ID, err)
	} else {
		ec = try
	}

	return s, p, i, ec, nil
}

func getMachineReferencedEntityMaps(ds *datastore.RethinkStore) (metal.SizeMap, metal.PartitionMap, metal.ImageMap, metal.ProvisioningEventContainerMap, error) {
	s, err := ds.ListSizes()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("sizes could not be listed: %w", err)
	}

	p, err := ds.ListPartitions()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("partitions could not be listed: %w", err)
	}

	i, err := ds.ListImages()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("images could not be listed: %w", err)
	}

	ec, err := ds.ListProvisioningEventContainers()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("provisioning event containers could not be listed: %w", err)
	}

	return s.ByID(), p.ByID(), i.ByID(), ec.ByID(), nil
}

func (s machineAllocationSpec) noautoNetworkN() int {
	result := 0
	for _, n := range s.Networks {
		if n.AutoAcquireIP != nil && !*n.AutoAcquireIP {
			result++
		}
	}
	return result
}

func (s machineAllocationSpec) autoNetworkN() int {
	return len(s.Networks) - s.noautoNetworkN()
}
