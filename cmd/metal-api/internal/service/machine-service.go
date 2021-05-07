package service

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/avast/retry-go"
	s3server "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/s3client"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/grpc"
	"github.com/pkg/errors"

	"golang.org/x/crypto/ssh"

	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/metal-lib/pkg/tag"
	"github.com/metal-stack/metal-lib/zapup"
	"go.uber.org/zap"

	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	mdm "github.com/metal-stack/masterdata-api/pkg/client"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/ipam"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/utils"

	"github.com/dustin/go-humanize"
	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metrics"
	"github.com/metal-stack/metal-lib/bus"
)

type machineResource struct {
	webResource
	bus.Publisher
	ipamer     ipam.IPAMer
	mdc        mdm.Client
	actor      *asyncActor
	grpcServer *grpc.Server
	s3Client   *s3server.Client
}

// machineAllocationSpec is a specification for a machine allocation
type machineAllocationSpec struct {
	UUID        string
	Name        string
	Description string
	Hostname    string
	ProjectID   string
	PartitionID string
	SizeID      string
	Image       *metal.Image
	SSHPubKeys  []string
	UserData    string
	Tags        []string
	Networks    v1.MachineAllocationNetworks
	IPs         []string
	HA          bool
	IsFirewall  bool
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
	ds *datastore.RethinkStore,
	pub bus.Publisher,
	ep *bus.Endpoints,
	ipamer ipam.IPAMer,
	mdc mdm.Client,
	grpcServer *grpc.Server,
	s3Client *s3server.Client) (*restful.WebService, error) {

	r := machineResource{
		webResource: webResource{
			ds: ds,
		},
		Publisher:  pub,
		ipamer:     ipamer,
		mdc:        mdc,
		grpcServer: grpcServer,
		s3Client:   s3Client,
	}
	var err error
	r.actor, err = newAsyncActor(zapup.MustRootLogger(), ep, ds, ipamer)
	if err != nil {
		return nil, fmt.Errorf("cannot create async actor: %w", err)
	}

	return r.webService(), nil
}

// webService creates the webservice endpoint
func (r machineResource) webService() *restful.WebService {
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
		Reads(v1.MachineFindRequest{}).
		Writes([]v1.MachineResponse{}).
		Returns(http.StatusOK, "OK", []v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/register").
		To(editor(r.registerMachine)).
		Operation("registerMachine").
		Doc("register a machine").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.MachineRegisterRequest{}).
		Writes(v1.MachineResponse{}).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		Returns(http.StatusCreated, "Created", v1.MachineResponse{}).
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

	ws.Route(ws.POST("/{id}/finalize-allocation").
		To(editor(r.finalizeAllocation)).
		Operation("finalizeAllocation").
		Doc("finalize the allocation of the machine by reconfiguring the switch, sent on successful image installation").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.MachineFinalizeAllocationRequest{}).
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

	ws.Route(ws.POST("/{id}/chassis-identify-led-state").
		To(editor(r.setChassisIdentifyLEDState)).
		Operation("setChassisIdentifyLEDState").
		Doc("set the state of a chassis identify LED").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.ChassisIdentifyLEDState{}).
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

	ws.Route(ws.POST("/{id}/abort-reinstall").
		To(editor(r.abortReinstallMachine)).
		Operation("abortReinstallMachine").
		Doc("abort reinstall this machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.MachineAbortReinstallRequest{}).
		Writes(v1.BootInfo{}).
		Returns(http.StatusOK, "OK", v1.BootInfo{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/{id}/event").
		To(viewer(r.getProvisioningEventContainer)).
		Operation("getProvisioningEventContainer").
		Doc("get the current machine provisioning event container").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.MachineRecentProvisioningEvents{}).
		Returns(http.StatusOK, "OK", v1.MachineRecentProvisioningEvents{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/event").
		To(editor(r.addProvisioningEvent)).
		Operation("addProvisioningEvent").
		Doc("adds a machine provisioning event").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.MachineProvisioningEvent{}).
		Writes(v1.MachineRecentProvisioningEvents{}).
		Returns(http.StatusOK, "OK", v1.MachineRecentProvisioningEvents{}).
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

func (r machineResource) listMachines(request *restful.Request, response *restful.Response) {
	ms, err := r.ds.ListMachines()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, makeMachineResponseList(ms, r.ds, utils.Logger(request).Sugar()))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r machineResource) findMachine(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	m, err := r.ds.FindMachineByID(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	resp := makeMachineResponse(m, r.ds, utils.Logger(request).Sugar())
	err = response.WriteHeaderAndEntity(http.StatusOK, resp)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r machineResource) findMachines(request *restful.Request, response *restful.Response) {
	var requestPayload datastore.MachineSearchQuery
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	ms := metal.Machines{}
	err = r.ds.SearchMachines(&requestPayload, &ms)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, makeMachineResponseList(ms, r.ds, utils.Logger(request).Sugar()))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r machineResource) setMachineState(request *restful.Request, response *restful.Response) {
	var requestPayload v1.MachineState
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	machineState, err := metal.MachineStateFrom(requestPayload.Value)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	if machineState != metal.AvailableState && requestPayload.Description == "" {
		// we want a cause why this machine is not available
		if checkError(request, response, utils.CurrentFuncName(), errors.New("you must supply a description")) {
			return
		}
	}

	id := request.PathParameter("id")
	oldMachine, err := r.ds.FindMachineByID(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	newMachine := *oldMachine

	newMachine.State = metal.MachineState{
		Value:       machineState,
		Description: requestPayload.Description,
	}

	err = r.ds.UpdateMachine(oldMachine, &newMachine)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, makeMachineResponse(&newMachine, r.ds, utils.Logger(request).Sugar()))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r machineResource) setChassisIdentifyLEDState(request *restful.Request, response *restful.Response) {
	var requestPayload v1.ChassisIdentifyLEDState
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	ledState, err := metal.LEDStateFrom(requestPayload.Value)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	if ledState == metal.LEDStateOff && requestPayload.Description == "" {
		// we want a cause why this chassis identify LED is off
		if checkError(request, response, utils.CurrentFuncName(), errors.New("you must supply a description")) {
			return
		}
	}

	id := request.PathParameter("id")
	oldMachine, err := r.ds.FindMachineByID(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	newMachine := *oldMachine

	newMachine.LEDState = metal.ChassisIdentifyLEDState{
		Value:       ledState,
		Description: requestPayload.Description,
	}

	err = r.ds.UpdateMachine(oldMachine, &newMachine)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = response.WriteHeaderAndEntity(http.StatusOK, makeMachineResponse(&newMachine, r.ds, utils.Logger(request).Sugar()))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r machineResource) registerMachine(request *restful.Request, response *restful.Response) {
	var requestPayload v1.MachineRegisterRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	if requestPayload.UUID == "" {
		if checkError(request, response, utils.CurrentFuncName(), errors.New("uuid cannot be empty")) {
			return
		}
	}

	partition, err := r.ds.FindPartition(requestPayload.PartitionID)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	machineHardware := v1.NewMetalMachineHardware(&requestPayload.Hardware)
	size, _, err := r.ds.FromHardware(machineHardware)
	if err != nil {
		size = metal.UnknownSize
		utils.Logger(request).Sugar().Errorw("no size found for hardware, defaulting to unknown size", "hardware", machineHardware, "error", err)
	}

	m, err := r.ds.FindMachineByID(requestPayload.UUID)
	if err != nil && !metal.IsNotFound(err) {
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
	}

	returnCode := http.StatusOK

	if m == nil {
		// machine is not in the database, create it
		name := fmt.Sprintf("%d-core/%s", machineHardware.CPUCores, humanize.Bytes(machineHardware.Memory))
		descr := fmt.Sprintf("a machine with %d core(s) and %s of RAM", machineHardware.CPUCores, humanize.Bytes(machineHardware.Memory))
		m = &metal.Machine{
			Base: metal.Base{
				ID:          requestPayload.UUID,
				Name:        name,
				Description: descr,
			},
			Allocation:  nil,
			SizeID:      size.ID,
			PartitionID: partition.ID,
			Hardware:    machineHardware,
			BIOS: metal.BIOS{
				Version: requestPayload.BIOS.Version,
				Vendor:  requestPayload.BIOS.Vendor,
				Date:    requestPayload.BIOS.Date,
			},
			State: metal.MachineState{
				Value: metal.AvailableState,
			},
			LEDState: metal.ChassisIdentifyLEDState{
				Value:       metal.LEDStateOff,
				Description: "Machine registered",
			},
			Tags: requestPayload.Tags,
			IPMI: v1.NewMetalIPMI(&requestPayload.IPMI),
		}

		err = r.ds.CreateMachine(m)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}

		returnCode = http.StatusCreated
	} else {
		// machine has already registered, update it
		old := *m

		m.SizeID = size.ID
		m.PartitionID = partition.ID
		m.Hardware = machineHardware
		m.BIOS.Version = requestPayload.BIOS.Version
		m.BIOS.Vendor = requestPayload.BIOS.Vendor
		m.BIOS.Date = requestPayload.BIOS.Date
		m.IPMI = v1.NewMetalIPMI(&requestPayload.IPMI)

		err = r.ds.UpdateMachine(&old, m)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
	}

	ec, err := r.ds.FindProvisioningEventContainer(m.ID)
	if err != nil && !metal.IsNotFound(err) {
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
	}

	if ec == nil {
		err = r.ds.CreateProvisioningEventContainer(&metal.ProvisioningEventContainer{
			Base:                         metal.Base{ID: m.ID},
			Liveliness:                   metal.MachineLivelinessAlive,
			IncompleteProvisioningCycles: "0",
		},
		)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
	}

	old := *m
	err = connectMachineWithSwitches(r.ds, m)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = r.ds.UpdateMachine(&old, m)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = response.WriteHeaderAndEntity(returnCode, makeMachineResponse(m, r.ds, utils.Logger(request).Sugar()))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r machineResource) findIPMIMachine(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	m, err := r.ds.FindMachineByID(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, makeMachineIPMIResponse(m, r.ds, utils.Logger(request).Sugar()))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r machineResource) findIPMIMachines(request *restful.Request, response *restful.Response) {
	var requestPayload datastore.MachineSearchQuery
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	ms := metal.Machines{}
	err = r.ds.SearchMachines(&requestPayload, &ms)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, makeMachineIPMIResponseList(ms, r.ds, utils.Logger(request).Sugar()))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r machineResource) ipmiReport(request *restful.Request, response *restful.Response) {
	var requestPayload v1.MachineIpmiReports
	log := utils.Logger(request)
	logger := log.Sugar()
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	if requestPayload.PartitionID == "" {
		err := errors.New("partition id is empty")
		checkError(request, response, utils.CurrentFuncName(), err)
		return
	}

	p, err := r.ds.FindPartition(requestPayload.PartitionID)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	var ms metal.Machines
	err = r.ds.SearchMachines(&datastore.MachineSearchQuery{}, &ms)
	if checkError(request, response, utils.CurrentFuncName(), err) {
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
				Address: report.BMCIp + ":" + defaultIPMIPort,
			},
		}
		err = r.ds.CreateMachine(m)
		if err != nil {
			logger.Errorf("could not create machine", "id", uuid, "ipmi-ip", report.BMCIp, "m", m, "err", err)
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
			logger.Errorf("not updating ipmi, address is garbage", "id", uuid, "ip", report.BMCIp, "machine", newMachine, "address", newMachine.IPMI.Address)
			continue
		}

		// machine was created by a PXE boot event and has no partition set.
		if oldMachine.PartitionID == "" {
			newMachine.PartitionID = p.ID
		}

		if newMachine.PartitionID != p.ID {
			logger.Errorf("could not update machine because overlapping id found", "id", uuid, "machine", newMachine, "partition", requestPayload.PartitionID)
			continue
		}

		updateFru(&newMachine, report.FRU)

		if report.BIOSVersion != "" {
			newMachine.BIOS.Version = report.BIOSVersion
		}
		if report.BMCVersion != "" {
			newMachine.IPMI.BMCVersion = report.BMCVersion
		}

		err = r.ds.UpdateMachine(&oldMachine, &newMachine)
		if err != nil {
			logger.Errorf("could not update machine", "id", uuid, "ip", report.BMCIp, "machine", newMachine, "err", err)
			continue
		}
		resp.Updated = append(resp.Updated, uuid)
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, resp)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func updateFru(m *metal.Machine, fru *v1.MachineFru) {
	if fru == nil {
		return
	}

	m.IPMI.Fru.ChassisPartSerial = utils.StrValueDefault(fru.ChassisPartSerial, m.IPMI.Fru.ChassisPartSerial)
	m.IPMI.Fru.ChassisPartNumber = utils.StrValueDefault(fru.ChassisPartNumber, m.IPMI.Fru.ChassisPartNumber)
	m.IPMI.Fru.BoardMfg = utils.StrValueDefault(fru.BoardMfg, m.IPMI.Fru.BoardMfg)
	m.IPMI.Fru.BoardMfgSerial = utils.StrValueDefault(fru.BoardMfgSerial, m.IPMI.Fru.BoardMfgSerial)
	m.IPMI.Fru.BoardPartNumber = utils.StrValueDefault(fru.BoardPartNumber, m.IPMI.Fru.BoardPartNumber)
	m.IPMI.Fru.ProductManufacturer = utils.StrValueDefault(fru.ProductManufacturer, m.IPMI.Fru.ProductManufacturer)
	m.IPMI.Fru.ProductSerial = utils.StrValueDefault(fru.ProductSerial, m.IPMI.Fru.ProductSerial)
	m.IPMI.Fru.ProductPartNumber = utils.StrValueDefault(fru.ProductPartNumber, m.IPMI.Fru.ProductPartNumber)
}

func (r machineResource) allocateMachine(request *restful.Request, response *restful.Response) {
	var requestPayload v1.MachineAllocateRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	var uuid string
	if requestPayload.UUID != nil {
		uuid = *requestPayload.UUID
	}
	var name string
	if requestPayload.Name != nil {
		name = *requestPayload.Name
	}
	var description string
	if requestPayload.Description != nil {
		description = *requestPayload.Description
	}
	hostname := "metal"
	if requestPayload.Hostname != nil {
		hostname = *requestPayload.Hostname
	}
	var userdata string
	if requestPayload.UserData != nil {
		userdata = *requestPayload.UserData
	}

	image, err := r.ds.FindImage(requestPayload.ImageID)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	if !image.HasFeature(metal.ImageFeatureMachine) {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("given image is not usable for a machine, features: %s", image.ImageFeatureString())) {
			return
		}
	}

	spec := machineAllocationSpec{
		UUID:        uuid,
		Name:        name,
		Description: description,
		Hostname:    hostname,
		ProjectID:   requestPayload.ProjectID,
		PartitionID: requestPayload.PartitionID,
		SizeID:      requestPayload.SizeID,
		Image:       image,
		SSHPubKeys:  requestPayload.SSHPubKeys,
		UserData:    userdata,
		Tags:        requestPayload.Tags,
		Networks:    requestPayload.Networks,
		IPs:         requestPayload.IPs,
		HA:          false,
		IsFirewall:  false,
	}

	m, err := allocateMachine(utils.Logger(request).Sugar(), r.ds, r.ipamer, &spec, r.mdc, r.actor, r.grpcServer)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		utils.Logger(request).Sugar().Errorw("machine allocation went wrong", "error", err)
		return
	}

	err = response.WriteHeaderAndEntity(http.StatusOK, makeMachineResponse(m, r.ds, utils.Logger(request).Sugar()))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func allocateMachine(logger *zap.SugaredLogger, ds *datastore.RethinkStore, ipamer ipam.IPAMer, allocationSpec *machineAllocationSpec, mdc mdm.Client, actor *asyncActor, ws *grpc.Server) (*metal.Machine, error) {
	err := validateAllocationSpec(allocationSpec)
	if err != nil {
		return nil, err
	}
	projectID := allocationSpec.ProjectID
	p, err := mdc.Project().Get(context.Background(), &mdmv1.ProjectGetRequest{Id: projectID})
	if err != nil {
		return nil, err
	}

	// Check if more machine would be allocated than project quota permits
	if p.GetProject() != nil && p.GetProject().GetQuotas() != nil && p.GetProject().GetQuotas().GetMachine() != nil {
		mq := p.GetProject().GetQuotas().GetMachine()
		maxMachines := mq.GetQuota().GetValue()
		var actualMachines metal.Machines
		err := ds.SearchMachines(&datastore.MachineSearchQuery{AllocationProject: &projectID}, &actualMachines)
		if err != nil {
			return nil, err
		}
		machineCount := int32(-1)
		imageMap, err := ds.ListImages()
		if err != nil {
			return nil, err
		}
		for _, m := range actualMachines {
			if m.IsFirewall(imageMap.ByID()) {
				continue
			}
			machineCount++
		}
		if machineCount >= maxMachines {
			return nil, fmt.Errorf("project quota for machines reached max:%d", maxMachines)
		}
	}

	var machineCandidate *metal.Machine
	err = retry.Do(
		func() error {
			var err2 error
			machineCandidate, err2 = findMachineCandidate(ds, allocationSpec)
			return err2
		},
		retry.Attempts(10),
		retry.RetryIf(func(err error) bool {
			return strings.Contains(err.Error(), datastore.EntityAlreadyModifiedErrorMessage)
		}),
		retry.DelayType(retry.CombineDelay(retry.BackOffDelay, retry.RandomDelay)),
		retry.LastErrorOnly(true),
	)
	if err != nil {
		return nil, err
	}
	// as some fields in the allocation spec are optional, they will now be clearly defined by the machine candidate
	allocationSpec.UUID = machineCandidate.ID
	allocationSpec.PartitionID = machineCandidate.PartitionID
	allocationSpec.SizeID = machineCandidate.SizeID

	alloc := &metal.MachineAllocation{
		Created:         time.Now(),
		Name:            allocationSpec.Name,
		Description:     allocationSpec.Description,
		Hostname:        allocationSpec.Hostname,
		Project:         projectID,
		ImageID:         allocationSpec.Image.ID,
		UserData:        allocationSpec.UserData,
		SSHPubKeys:      allocationSpec.SSHPubKeys,
		MachineNetworks: []*metal.MachineNetwork{},
	}
	rollbackOnError := func(err error) error {
		if err != nil {
			cleanupMachine := &metal.Machine{
				Base: metal.Base{
					ID: allocationSpec.UUID,
				},
				Allocation: alloc,
			}
			rollbackError := actor.machineReleaser(cleanupMachine)
			if rollbackError != nil {
				logger.Errorw("cannot call async machine cleanup", "error", rollbackError)
			}
		}
		return err
	}
	networks, err := gatherNetworks(ds, allocationSpec)
	if err != nil {
		return nil, rollbackOnError(err)
	}
	err = makeNetworks(ds, ipamer, allocationSpec, networks, alloc)
	if err != nil {
		return nil, rollbackOnError(err)
	}

	// refetch the machine to catch possible updates after dealing with the network...
	machine, err := ds.FindMachineByID(machineCandidate.ID)
	if err != nil {
		return nil, rollbackOnError(err)
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

	err = ws.NotifyAllocated(machine.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to notify machine allocation %q, %w", machine.ID, err)
	}

	return machine, nil
}

func validateAllocationSpec(allocationSpec *machineAllocationSpec) error {
	if allocationSpec.ProjectID == "" {
		return errors.New("project id must be specified")
	}

	if allocationSpec.UUID == "" && allocationSpec.PartitionID == "" {
		return errors.New("when no machine id is given, a partition id must be specified")
	}

	if allocationSpec.UUID == "" && allocationSpec.SizeID == "" {
		return errors.New("when no machine id is given, a size id must be specified")
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
	if allocationSpec.IsFirewall {
		if len(allocationSpec.IPs) == 0 && allocationSpec.autoNetworkN() == 0 {
			return errors.New("when no ip is given at least one auto acquire network must be specified")
		}
	}

	if noautoNetN := allocationSpec.noautoNetworkN(); noautoNetN > len(allocationSpec.IPs) {
		return errors.New("missing ip(s) for network(s) without automatic ip allocation")
	}

	return nil
}

func findMachineCandidate(ds *datastore.RethinkStore, allocationSpec *machineAllocationSpec) (*metal.Machine, error) {
	var err error
	var machine *metal.Machine
	if allocationSpec.UUID == "" {
		// requesting allocation of an arbitrary ready machine in partition with given size
		machine, err = findWaitingMachine(ds, allocationSpec.PartitionID, allocationSpec.SizeID)
		if err != nil {
			return nil, err
		}
	} else {
		// requesting allocation of a specific, existing machine
		machine, err = ds.FindMachineByID(allocationSpec.UUID)
		if err != nil {
			return nil, fmt.Errorf("machine cannot be found: %w", err)
		}

		if machine.Allocation != nil {
			return nil, errors.New("machine is already allocated")
		}
		if allocationSpec.PartitionID != "" && machine.PartitionID != allocationSpec.PartitionID {
			return nil, fmt.Errorf("machine %q is not in the requested partition: %s", machine.ID, allocationSpec.PartitionID)
		}

		if allocationSpec.SizeID != "" && machine.SizeID != allocationSpec.SizeID {
			return nil, fmt.Errorf("machine %q does not have the requested size: %s", machine.ID, allocationSpec.SizeID)
		}
	}
	return machine, err
}

func findWaitingMachine(ds *datastore.RethinkStore, partitionID, sizeID string) (*metal.Machine, error) {
	size, err := ds.FindSize(sizeID)
	if err != nil {
		return nil, fmt.Errorf("size cannot be found: %w", err)
	}
	partition, err := ds.FindPartition(partitionID)
	if err != nil {
		return nil, fmt.Errorf("partition cannot be found: %w", err)
	}
	machine, err := ds.FindWaitingMachine(partition.ID, size.ID)
	if err != nil {
		return nil, err
	}
	return machine, nil
}

// makeNetworks creates network entities and ip addresses as specified in the allocation network map.
// created networks are added to the machine allocation directly after their creation. This way, the rollback mechanism
// is enabled to clean up networks that were already created.
func makeNetworks(ds *datastore.RethinkStore, ipamer ipam.IPAMer, allocationSpec *machineAllocationSpec, networks allocationNetworkMap, alloc *metal.MachineAllocation) error {
	for _, n := range networks {
		machineNetwork, err := makeMachineNetwork(ds, ipamer, allocationSpec, n)
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
		return nil, errors.Wrap(err, "partition has no private super network")
	}

	specNetworks, err := gatherNetworksFromSpec(ds, allocationSpec, partition, privateSuperNetworks)
	if err != nil {
		return nil, err
	}

	var underlayNetwork *allocationNetwork
	if allocationSpec.IsFirewall {
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

	if !allocationSpec.IsFirewall && len(privateNetworks) > 1 {
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
		return nil, fmt.Errorf("no underlay found in the given partition: %w", err)
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

func makeMachineNetwork(ds *datastore.RethinkStore, ipamer ipam.IPAMer, allocationSpec *machineAllocationSpec, n *allocationNetwork) (*metal.MachineNetwork, error) {
	if n.auto {
		ipAddress, ipParentCidr, err := allocateIP(n.network, "", ipamer)
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

func (r machineResource) finalizeAllocation(request *restful.Request, response *restful.Response) {
	var requestPayload v1.MachineFinalizeAllocationRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	id := request.PathParameter("id")
	m, err := r.ds.FindMachineByID(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	if m.Allocation == nil {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("the machine %q is not allocated", id)) {
			return
		}
	}

	old := *m

	m.Allocation.ConsolePassword = requestPayload.ConsolePassword
	m.Allocation.MachineSetup = &metal.MachineSetup{
		ImageID:      m.Allocation.ImageID,
		PrimaryDisk:  requestPayload.PrimaryDisk,
		OSPartition:  requestPayload.OSPartition,
		Initrd:       requestPayload.Initrd,
		Cmdline:      requestPayload.Cmdline,
		Kernel:       requestPayload.Kernel,
		BootloaderID: requestPayload.BootloaderID,
	}
	m.Allocation.Reinstall = false

	err = r.ds.UpdateMachine(&old, m)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	vrf := ""
	imgs, err := r.ds.ListImages()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	if m.IsFirewall(imgs.ByID()) {
		// if a machine has multiple networks, it serves as firewall, so it can not be enslaved into the tenant vrf
		vrf = "default"
	} else {
		for _, mn := range m.Allocation.MachineNetworks {
			if mn.Private {
				vrf = fmt.Sprintf("vrf%d", mn.Vrf)
				break
			}
		}
	}

	_, err = setVrfAtSwitches(r.ds, m, vrf)
	if err != nil {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("the machine %q could not be enslaved into the vrf %s", id, vrf)) {
			return
		}
	}

	err = response.WriteHeaderAndEntity(http.StatusOK, makeMachineResponse(m, r.ds, utils.Logger(request).Sugar()))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r machineResource) freeMachine(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	m, err := r.ds.FindMachineByID(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	logger := utils.Logger(request).Sugar()

	err = r.actor.freeMachine(r.Publisher, m)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = response.WriteHeaderAndEntity(http.StatusOK, makeMachineResponse(m, r.ds, logger))
	if err != nil {
		logger.Error("Failed to send response", zap.Error(err))
	}

	event := string(metal.ProvisioningEventPlannedReboot)
	_, err = r.provisioningEventForMachine(id, v1.MachineProvisioningEvent{Time: time.Now(), Event: event, Message: "freeMachine"})
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
}

func (r machineResource) deleteMachine(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	m, err := r.ds.FindMachineByID(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	logger := utils.Logger(request).Sugar()

	if m.Allocation != nil {
		checkError(request, response, utils.CurrentFuncName(), errors.New("cannot delete machine that is allocated"))
		return
	}

	ec, err := r.ds.FindProvisioningEventContainer(id)

	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	if !ec.Liveliness.Is(string(metal.MachineLivelinessDead)) {
		checkError(request, response, utils.CurrentFuncName(), errors.New("can only delete dead machines"))
		return
	}

	switches, err := r.ds.SearchSwitchesConnectedToMachine(m)
	if checkError(request, response, utils.CurrentFuncName(), err) {
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
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
	}

	err = r.ds.DeleteMachine(m)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = response.WriteHeaderAndEntity(http.StatusOK, makeMachineResponse(m, r.ds, logger))
	if err != nil {
		logger.Error("Failed to send response", zap.Error(err))
	}
}

// reinstallMachine reinstalls the requested machine with given image by either allocating
// the machine if not yet allocated or not modifying any other allocation parameter than 'ImageID'
// and 'Reinstall' set to true.
// If the given image ID is nil, it deletes the machine instead.
func (r machineResource) reinstallMachine(request *restful.Request, response *restful.Response) {
	var requestPayload v1.MachineReinstallRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	id := request.PathParameter("id")
	m, err := r.ds.FindMachineByID(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	logger := utils.Logger(request)

	if m.Allocation != nil && m.State.Value != metal.LockedState {
		old := *m

		m.Allocation.Reinstall = true
		m.Allocation.ImageID = requestPayload.ImageID

		resp := makeMachineResponse(m, r.ds, logger.Sugar())
		if resp.Allocation.Image != nil {
			err = r.ds.UpdateMachine(&old, m)
			if checkError(request, response, utils.CurrentFuncName(), err) {
				return
			}
			logger.Info("marked machine to get reinstalled", zap.String("machineID", m.ID))

			err = deleteVRFSwitches(r.ds, m, logger)
			if checkError(request, response, utils.CurrentFuncName(), err) {
				return
			}

			err = publishDeleteEvent(r.Publisher, m, logger)
			if checkError(request, response, utils.CurrentFuncName(), err) {
				return
			}

			err = publishMachineCmd(logger.Sugar(), m, r.Publisher, metal.MachineReinstallCmd)
			if err != nil {
				logger.Error("unable to publish machine command", zap.String("command", string(metal.MachineReinstallCmd)), zap.String("machineID", m.ID), zap.Error(err))
			}

			err = response.WriteHeaderAndEntity(http.StatusOK, resp)
			if err != nil {
				logger.Error("Failed to send response", zap.Error(err))
			}

			return
		}
	}

	err = response.WriteHeaderAndEntity(http.StatusBadRequest, httperrors.NewHTTPError(http.StatusBadRequest, errors.New("machine either locked, not allocated yet or invalid image ID specified")))
	if err != nil {
		logger.Error("Failed to send response", zap.Error(err))
	}
}

func (r machineResource) abortReinstallMachine(request *restful.Request, response *restful.Response) {
	var requestPayload v1.MachineAbortReinstallRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	id := request.PathParameter("id")
	m, err := r.ds.FindMachineByID(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	logger := utils.Logger(request).Sugar()

	var bootInfo *v1.BootInfo

	if m.Allocation != nil && !requestPayload.PrimaryDiskWiped {
		old := *m

		m.Allocation.Reinstall = false
		if m.Allocation.MachineSetup != nil {
			m.Allocation.ImageID = m.Allocation.MachineSetup.ImageID
		}

		err = r.ds.UpdateMachine(&old, m)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
		logger.Infow("removed reinstall mark", "machineID", m.ID)

		if m.Allocation.MachineSetup != nil {
			bootInfo = &v1.BootInfo{
				ImageID:      m.Allocation.MachineSetup.ImageID,
				PrimaryDisk:  m.Allocation.MachineSetup.PrimaryDisk,
				OSPartition:  m.Allocation.MachineSetup.OSPartition,
				Initrd:       m.Allocation.MachineSetup.Initrd,
				Cmdline:      m.Allocation.MachineSetup.Cmdline,
				Kernel:       m.Allocation.MachineSetup.Kernel,
				BootloaderID: m.Allocation.MachineSetup.BootloaderID,
			}
		}
	}

	err = response.WriteHeaderAndEntity(http.StatusOK, bootInfo)
	if err != nil {
		logger.Error("Failed to send response", zap.Error(err))
	}
}

func deleteVRFSwitches(ds *datastore.RethinkStore, m *metal.Machine, logger *zap.Logger) error {
	logger.Info("set VRF at switch", zap.String("machineID", m.ID))
	_, err := setVrfAtSwitches(ds, m, "")
	if err != nil {
		logger.Error("cannot delete vrf switches", zap.String("machineID", m.ID), zap.Error(err))
		return fmt.Errorf("cannot delete vrf switches: %w", err)
	}
	return nil
}

func publishDeleteEvent(publisher bus.Publisher, m *metal.Machine, logger *zap.Logger) error {
	logger.Info("publish machine delete event", zap.String("machineID", m.ID))
	deleteEvent := metal.MachineEvent{Type: metal.DELETE, OldMachineID: m.ID}
	err := publisher.Publish(metal.TopicMachine.GetFQN(m.PartitionID), deleteEvent)
	if err != nil {
		logger.Error("cannot publish delete event", zap.String("machineID", m.ID), zap.Error(err))
		return fmt.Errorf("cannot publish delete event: %w", err)
	}
	return nil
}

func (r machineResource) getProvisioningEventContainer(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	// check for existence of the machine
	_, err := r.ds.FindMachineByID(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	ec, err := r.ds.FindProvisioningEventContainer(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, v1.NewMachineRecentProvisioningEvents(ec))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r machineResource) addProvisioningEvent(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	m, err := r.ds.FindMachineByID(id)
	if err != nil && !metal.IsNotFound(err) {
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
	}

	// an event can actually create an empty machine. This enables us to also catch the very first PXE Booting event
	// in a machine lifecycle
	if m == nil {
		m = &metal.Machine{
			Base: metal.Base{
				ID: id,
			},
		}
		err = r.ds.CreateMachine(m)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
	}

	var requestPayload v1.MachineProvisioningEvent
	err = request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	ok := metal.AllProvisioningEventTypes[metal.ProvisioningEventType(requestPayload.Event)]
	if !ok {
		if checkError(request, response, utils.CurrentFuncName(), errors.New("unknown provisioning event")) {
			return
		}
	}

	ec, err := r.provisioningEventForMachine(id, requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = response.WriteHeaderAndEntity(http.StatusOK, v1.NewMachineRecentProvisioningEvents(ec))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r machineResource) provisioningEventForMachine(machineID string, e v1.MachineProvisioningEvent) (*metal.ProvisioningEventContainer, error) {
	ec, err := r.ds.FindProvisioningEventContainer(machineID)
	if err != nil && !metal.IsNotFound(err) {
		return nil, err
	}

	if ec == nil {
		ec = &metal.ProvisioningEventContainer{
			Base: metal.Base{
				ID: machineID,
			},
			Liveliness: metal.MachineLivelinessAlive,
		}
	}
	now := time.Now()
	ec.LastEventTime = &now

	event := metal.ProvisioningEvent{
		Time:    now,
		Event:   metal.ProvisioningEventType(e.Event),
		Message: e.Message,
	}
	if event.Event == metal.ProvisioningEventAlive {
		zapup.MustRootLogger().Sugar().Debugw("received provisioning alive event", "id", ec.ID)
		ec.Liveliness = metal.MachineLivelinessAlive
	} else if event.Event == metal.ProvisioningEventPhonedHome && len(ec.Events) > 0 && ec.Events[0].Event == metal.ProvisioningEventPhonedHome {
		zapup.MustRootLogger().Sugar().Debugw("swallowing repeated phone home event", "id", ec.ID)
		ec.Liveliness = metal.MachineLivelinessAlive
	} else {
		ec.Events = append([]metal.ProvisioningEvent{event}, ec.Events...)
		ec.IncompleteProvisioningCycles = ec.CalculateIncompleteCycles(zapup.MustRootLogger().Sugar())
		ec.Liveliness = metal.MachineLivelinessAlive
	}
	ec.TrimEvents(metal.ProvisioningEventsInspectionLimit)

	err = r.ds.UpsertProvisioningEventContainer(ec)
	return ec, err
}

// MachineLiveliness evaluates whether machines are still alive or if they have died
func MachineLiveliness(ds *datastore.RethinkStore, logger *zap.SugaredLogger) error {
	logger.Info("machine liveliness was requested")

	machines, err := ds.ListMachines()
	if err != nil {
		return err
	}

	liveliness := make(metrics.PartitionLiveliness)

	unknown := 0
	alive := 0
	dead := 0
	errs := 0
	for _, m := range machines {
		p := liveliness[m.PartitionID]
		lvlness, err := evaluateMachineLiveliness(ds, m)
		if err != nil {
			logger.Errorw("cannot update liveliness", "error", err, "machine", m)
			errs++
			// fall through, so the rest of the machines is getting evaluated
		}
		switch lvlness {
		case metal.MachineLivelinessAlive:
			alive++
			p.Alive++
		case metal.MachineLivelinessDead:
			dead++
			p.Dead++
		case metal.MachineLivelinessUnknown:
			unknown++
			p.Unknown++
		}
		liveliness[m.PartitionID] = p
	}

	metrics.ProvideLiveliness(liveliness)

	logger.Infow("machine liveliness evaluated", "alive", alive, "dead", dead, "unknown", unknown, "errors", errs)

	return nil
}

func evaluateMachineLiveliness(ds *datastore.RethinkStore, m metal.Machine) (metal.MachineLiveliness, error) {
	provisioningEvents, err := ds.FindProvisioningEventContainer(m.ID)
	if err != nil {
		// we have no provisioning events... we cannot tell
		return metal.MachineLivelinessUnknown, fmt.Errorf("no provisioningEvents found for ID: %s", m.ID)
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
func ResurrectMachines(ds *datastore.RethinkStore, publisher bus.Publisher, ep *bus.Endpoints, ipamer ipam.IPAMer, logger *zap.SugaredLogger) error {
	logger.Info("machine resurrection was requested")

	machines, err := ds.ListMachines()
	if err != nil {
		return err
	}

	act, err := newAsyncActor(logger.Desugar(), ep, ds, ipamer)
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
			logger.Debugw("no provisioningEvents found for resurrection", "machineID", m.ID, "error", err)
			continue
		}

		if provisioningEvents.Liveliness != metal.MachineLivelinessDead {
			continue
		}

		if provisioningEvents.LastEventTime == nil {
			continue
		}

		if time.Since(*provisioningEvents.LastEventTime) < metal.MachineResurrectAfter {
			continue
		}

		logger.Infow("resurrecting dead machine", "machineID", m.ID, "liveliness", provisioningEvents.Liveliness, "since", time.Since(*provisioningEvents.LastEventTime).String())
		err = act.freeMachine(publisher, &m)
		if err != nil {
			logger.Errorw("error during machine resurrection", "machineID", m.ID, "error", err)
		}
	}

	logger.Info("finished machine resurrection")

	return nil
}

func (r machineResource) machineOn(request *restful.Request, response *restful.Response) {
	r.machineCmd("machineOn", metal.MachineOnCmd, request, response)
}

func (r machineResource) machineOff(request *restful.Request, response *restful.Response) {
	r.machineCmd("machineOff", metal.MachineOffCmd, request, response)
}

func (r machineResource) machineReset(request *restful.Request, response *restful.Response) {
	r.machineCmd("machineReset", metal.MachineResetCmd, request, response)
}

func (r machineResource) machineBios(request *restful.Request, response *restful.Response) {
	r.machineCmd("machineBios", metal.MachineBiosCmd, request, response)
}

func (r machineResource) machineDisk(request *restful.Request, response *restful.Response) {
	r.machineCmd("machineDisk", metal.MachineDiskCmd, request, response)
}

func (r machineResource) machinePxe(request *restful.Request, response *restful.Response) {
	r.machineCmd("machinePxe", metal.MachinePxeCmd, request, response)
}

func (r machineResource) chassisIdentifyLEDOn(request *restful.Request, response *restful.Response) {
	r.machineCmd("chassisIdentifyLEDOn", metal.ChassisIdentifyLEDOnCmd, request, response, request.QueryParameter("description"))
}

func (r machineResource) chassisIdentifyLEDOff(request *restful.Request, response *restful.Response) {
	r.machineCmd("chassisIdentifyLEDOff", metal.ChassisIdentifyLEDOffCmd, request, response, request.QueryParameter("description"))
}

func (r machineResource) updateFirmware(request *restful.Request, response *restful.Response) {
	if r.s3Client == nil && checkError(request, response, utils.CurrentFuncName(), featureDisabledErr) {
		return
	}

	var p v1.MachineUpdateFirmwareRequest
	err := request.ReadEntity(&p)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	id := request.PathParameter("id")
	f, err := getFirmware(r.ds, id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	alreadyInstalled := false
	switch p.Kind {
	case bmc:
		alreadyInstalled = f.BiosVersion == p.Revision
	case bios:
		alreadyInstalled = f.BmcVersion == p.Revision
	}
	if alreadyInstalled && checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("machine's %s version is already equal %s", p.Kind, p.Revision)) {
		return
	}

	rr, err := getFirmwareRevisions(r.s3Client, p.Kind, f.Vendor, f.Board)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	notAvailable := true
	for _, rev := range rr {
		if rev == p.Revision {
			notAvailable = false
			break
		}
	}
	if notAvailable && checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("machine's %s firmware in version %s is not available", p.Kind, p.Revision)) {
		return
	}

	r.machineCmd("updateFirmware", metal.UpdateFirmwareCmd, request, response, p.Kind, p.Revision, p.Description, r.s3Client.Url, r.s3Client.Key, r.s3Client.Secret, r.s3Client.FirmwareBucket)
}

func (r machineResource) machineCmd(op string, cmd metal.MachineCommand, request *restful.Request, response *restful.Response, params ...string) {
	logger := utils.Logger(request).Sugar()
	id := request.PathParameter("id")

	m, err := r.ds.FindMachineByID(id)
	if checkError(request, response, op, err) {
		return
	}

	switch op {
	case "machineReset", "machineOff":
		event := string(metal.ProvisioningEventPlannedReboot)
		_, err = r.provisioningEventForMachine(id, v1.MachineProvisioningEvent{Time: time.Now(), Event: event, Message: op})
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
	}

	err = publishMachineCmd(logger, m, r.Publisher, cmd, params...)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = response.WriteHeaderAndEntity(http.StatusOK, makeMachineResponse(m, r.ds, utils.Logger(request).Sugar()))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func publishMachineCmd(logger *zap.SugaredLogger, m *metal.Machine, publisher bus.Publisher, cmd metal.MachineCommand, params ...string) error {
	pp := []string{}
	for _, p := range params {
		if len(p) > 0 {
			pp = append(pp, p)
		}
	}
	evt := metal.MachineEvent{
		Type: metal.COMMAND,
		Cmd: &metal.MachineExecCommand{
			Command:         cmd,
			Params:          pp,
			TargetMachineID: m.ID,
		},
	}

	logger.Infow("publish event", "event", evt, "command", *evt.Cmd)
	err := publisher.Publish(metal.TopicMachine.GetFQN(m.PartitionID), evt)
	if err != nil {
		return err
	}

	return nil
}

func machineHasIssues(m *v1.MachineResponse) bool {
	if m.Partition == nil {
		return true
	}
	if !metal.MachineLivelinessAlive.Is(m.Liveliness) {
		return true
	}
	if m.Allocation == nil && len(m.RecentProvisioningEvents.Events) > 0 && metal.ProvisioningEventPhonedHome.Is(m.RecentProvisioningEvents.Events[0].Event) {
		// not allocated, but phones home
		return true
	}
	if m.RecentProvisioningEvents.IncompleteProvisioningCycles != "" && m.RecentProvisioningEvents.IncompleteProvisioningCycles != "0" {
		// Machines with incomplete cycles but in "Waiting" state are considered available
		if len(m.RecentProvisioningEvents.Events) > 0 && !metal.ProvisioningEventWaiting.Is(m.RecentProvisioningEvents.Events[0].Event) {
			return true
		}
	}

	return false
}

func makeMachineResponse(m *metal.Machine, ds *datastore.RethinkStore, logger *zap.SugaredLogger) *v1.MachineResponse {
	s, p, i, ec := findMachineReferencedEntities(m, ds, logger)
	return v1.NewMachineResponse(m, s, p, i, ec)
}

func makeMachineResponseList(ms metal.Machines, ds *datastore.RethinkStore, logger *zap.SugaredLogger) []*v1.MachineResponse {
	sMap, pMap, iMap, ecMap := getMachineReferencedEntityMaps(ds, logger)

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

	return result
}

func makeMachineIPMIResponse(m *metal.Machine, ds *datastore.RethinkStore, logger *zap.SugaredLogger) *v1.MachineIPMIResponse {
	s, p, i, ec := findMachineReferencedEntities(m, ds, logger)
	return v1.NewMachineIPMIResponse(m, s, p, i, ec)
}

func makeMachineIPMIResponseList(ms metal.Machines, ds *datastore.RethinkStore, logger *zap.SugaredLogger) []*v1.MachineIPMIResponse {
	sMap, pMap, iMap, ecMap := getMachineReferencedEntityMaps(ds, logger)

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

	return result
}

func findMachineReferencedEntities(m *metal.Machine, ds *datastore.RethinkStore, logger *zap.SugaredLogger) (*metal.Size, *metal.Partition, *metal.Image, *metal.ProvisioningEventContainer) {
	var err error

	var s *metal.Size
	if m.SizeID != "" {
		if m.SizeID == metal.UnknownSize.GetID() {
			s = metal.UnknownSize
		} else {
			s, err = ds.FindSize(m.SizeID)
			if err != nil {
				logger.Errorw("machine references size, but size cannot be found in database", "machineID", m.ID, "sizeID", m.SizeID, "error", err)
			}
		}
	}

	var p *metal.Partition
	if m.PartitionID != "" {
		p, err = ds.FindPartition(m.PartitionID)
		if err != nil {
			logger.Errorw("machine references partition, but partition cannot be found in database", "machineID", m.ID, "partitionID", m.PartitionID, "error", err)
		}
	}

	var i *metal.Image
	if m.Allocation != nil {
		if m.Allocation.ImageID != "" {
			i, err = ds.GetImage(m.Allocation.ImageID)
			if err != nil {
				logger.Errorw("machine references image, but image cannot be found in database", "machineID", m.ID, "imageID", m.Allocation.ImageID, "error", err)
			}
		}
	}

	var ec *metal.ProvisioningEventContainer
	try, err := ds.FindProvisioningEventContainer(m.ID)
	if err != nil {
		logger.Errorw("machine has no provisioning event container in the database", "machineID", m.ID, "error", err)
	} else {
		ec = try
	}

	return s, p, i, ec
}

func getMachineReferencedEntityMaps(ds *datastore.RethinkStore, logger *zap.SugaredLogger) (metal.SizeMap, metal.PartitionMap, metal.ImageMap, metal.ProvisioningEventContainerMap) {
	s, err := ds.ListSizes()
	if err != nil {
		logger.Errorw("sizes could not be listed", "error", err)
	}

	p, err := ds.ListPartitions()
	if err != nil {
		logger.Errorw("partitions could not be listed", "error", err)
	}

	i, err := ds.ListImages()
	if err != nil {
		logger.Errorw("images could not be listed", "error", err)
	}

	ec, err := ds.ListProvisioningEventContainers()
	if err != nil {
		logger.Errorw("provisioning event containers could not be listed", "error", err)
	}

	return s.ByID(), p.ByID(), i.ByID(), ec.ByID()
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
