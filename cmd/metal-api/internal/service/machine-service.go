package service

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metrics"
	"github.com/metal-stack/metal-lib/bus"
	"github.com/pkg/errors"
)

const (
	waitForServerTimeout = 30 * time.Second
)

type machineResource struct {
	webResource
	bus.Publisher
	ipamer ipam.IPAMer
	mdc    mdm.Client
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
	network        *metal.Network
	machineNetwork *metal.MachineNetwork
	ips            []metal.IP
	auto           bool
	isPrivate      bool
}

// allocationNetworkMap is a map of allocationNetworks with the network id as the key
type allocationNetworkMap map[string]*allocationNetwork

// getPrivateNetwork extracts the private network from an allocationNetworkMap
func getPrivateNetwork(networks allocationNetworkMap) (*allocationNetwork, error) {
	var privateNetwork *allocationNetwork
	for _, n := range networks {
		if n.isPrivate {
			privateNetwork = n
			break
		}
	}
	if privateNetwork == nil {
		return nil, fmt.Errorf("no private network contained")
	}
	return privateNetwork, nil
}

// getMachineNetworks extracts the machines networks from an allocationNetworkMap
func getMachineNetworks(networks allocationNetworkMap) []*metal.MachineNetwork {
	machineNetworks := []*metal.MachineNetwork{}
	for _, n := range networks {
		machineNetworks = append(machineNetworks, n.machineNetwork)
	}
	return machineNetworks
}

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
	ipamer ipam.IPAMer,
	mdc mdm.Client) *restful.WebService {
	r := machineResource{
		webResource: webResource{
			ds: ds,
		},
		Publisher: pub,
		ipamer:    ipamer,
		mdc:       mdc,
	}
	return r.webService()
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
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/finalize-allocation").
		To(editor(r.finalizeAllocation)).
		Operation("finalizeAllocation").
		Doc("finalize the allocation of the machine by reconfiguring the switch, sent on successful image installation").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.MachineFinalizeAllocationRequest{}).
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
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/ipmi").
		To(editor(r.ipmiReport)).
		Operation("ipmiReport").
		Doc("reports IPMI ip addresses leased by a management server for machines").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.MachineIpmiReport{}).
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

	ws.Route(ws.GET("/{id}/wait").
		To(editor(r.waitForAllocation)).
		Operation("waitForAllocation").
		Doc("wait for an allocation of this machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		Returns(http.StatusGatewayTimeout, "Timeout", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/reinstall").
		To(editor(r.reinstallMachine)).
		Operation("reinstallMachine").
		Doc("reinstall this machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.MachineReinstallRequest{}).
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
		Returns(http.StatusOK, "OK", v1.MachineRecentProvisioningEvents{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/event").
		To(editor(r.addProvisioningEvent)).
		Operation("addProvisioningEvent").
		Doc("adds a machine provisioning event").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.MachineProvisioningEvent{}).
		Returns(http.StatusOK, "OK", v1.MachineRecentProvisioningEvents{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/power/on").
		To(editor(r.machineOn)).
		Operation("machineOn").
		Doc("sends a power-on to the machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.EmptyBody{}).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/power/off").
		To(editor(r.machineOff)).
		Operation("machineOff").
		Doc("sends a power-off to the machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.EmptyBody{}).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/power/reset").
		To(editor(r.machineReset)).
		Operation("machineReset").
		Doc("sends a reset to the machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.EmptyBody{}).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/power/bios").
		To(editor(r.machineBios)).
		Operation("machineBios").
		Doc("boots machine into BIOS on next reboot").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.EmptyBody{}).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/power/chassis-identify-led-on").
		To(editor(r.chassisIdentifyLEDOn)).
		Operation("chassisIdentifyLEDOn").
		Doc("sends a power-on to the chassis identify LED").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.EmptyBody{}).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/power/chassis-identify-led-on/{description}").
		To(editor(r.chassisIdentifyLEDOn)).
		Operation("chassisIdentifyLEDOn").
		Doc("sends a power-on to the chassis identify LED").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Param(ws.PathParameter("description", "reason why the chassis identify LED has been turned on").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.EmptyBody{}).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/power/chassis-identify-led-off/{description}").
		To(editor(r.chassisIdentifyLEDOff)).
		Operation("chassisIdentifyLEDOff").
		Doc("sends a power-off to the chassis identify LED").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Param(ws.PathParameter("description", "reason why the chassis identify LED has been turned off").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.EmptyBody{}).
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

func (r machineResource) waitForAllocation(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	ctx, cancel := context.WithCancel(request.Request.Context())
	log := utils.Logger(request)

	// after leaving waiting, stop listening for machine table changes in the background
	defer cancel()

	err := r.wait(ctx, id, log.Sugar(), func(alloc Allocation) error {
		select {
		case <-time.After(waitForServerTimeout):
			err := response.WriteHeaderAndEntity(http.StatusGatewayTimeout, httperrors.NewHTTPError(http.StatusGatewayTimeout, fmt.Errorf("server timeout")))
			if err != nil {
				zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
				return nil
			}
		case a := <-alloc:
			if a.Err != nil {
				log.Sugar().Errorw("allocation returned an error", "error", a.Err)
				return a.Err
			}

			s, p, i, ec := findMachineReferencedEntities(a.Machine, r.ds, log.Sugar())
			err := response.WriteHeaderAndEntity(http.StatusOK, v1.NewMachineResponse(a.Machine, s, p, i, ec))
			if err != nil {
				zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
				return nil
			}
		case <-ctx.Done():
			return fmt.Errorf("client timeout")
		}
		return nil
	})
	if err != nil {
		sendError(log, response, utils.CurrentFuncName(), httperrors.InternalServerError(err))
	}
}

// Wait inserts the machine with the given ID in the waittable, so
// this machine is ready for allocation. After this, this function waits
// for an update of this record in the waittable, which is a signal that
// this machine is allocated. This allocation will be signaled via the
// given allocator in a separate goroutine. The allocator is a function
// which will receive a channel and the caller has to select on this
// channel to get a result. Using a channel allows the caller of this
// function to implement timeouts to not wait forever.
// The user of this function will block until this machine is allocated.
func (r machineResource) wait(ctx context.Context, id string, logger *zap.SugaredLogger, allocator Allocator) error {
	m, err := r.ds.FindMachineByID(id)
	if err != nil {
		return err
	}
	a := make(chan MachineAllocation, 1)

	// the machine IS already allocated, so notify this allocation back.
	if m.Allocation != nil {
		go func() {
			a <- MachineAllocation{Machine: m}
		}()
		return allocator(a)
	}

	err = r.ds.InsertWaitingMachine(m)
	if err != nil {
		return err
	}
	defer func() {
		err := r.ds.RemoveWaitingMachine(m)
		if err != nil {
			logger.Errorw("could not remove machine from wait table", "error", err)
		}
	}()

	go func() {
		changedMachine, err := r.ds.WaitForMachineAllocation(ctx, m)
		if err != nil {
			logger.Errorw("WaitForMachineAllocation returned an error", "error", err)
			a <- MachineAllocation{Err: err}
		} else {
			a <- MachineAllocation{Machine: changedMachine}
		}
		close(a)
	}()

	return allocator(a)
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
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("you must supply a description")) {
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
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("you must supply a description")) {
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
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("uuid cannot be empty")) {
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
			RackID:      requestPayload.RackID,
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
		m.RackID = requestPayload.RackID
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
			IncompleteProvisioningCycles: "0"},
		)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
	}

	err = connectMachineWithSwitches(r.ds, m)
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
	var requestPayload v1.MachineIpmiReport
	log := utils.Logger(request)
	logger := log.Sugar()
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	if requestPayload.PartitionID == "" {
		err := fmt.Errorf("given partition id was not found")
		checkError(request, response, utils.CurrentFuncName(), err)
		return
	}

	var ms metal.Machines
	err = r.ds.SearchMachines(&datastore.MachineSearchQuery{}, &ms)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	known := v1.Leases{}
	for _, m := range ms {
		uuid := m.ID
		if uuid == "" {
			continue
		}
		known[uuid] = m.IPMI.Address
	}
	resp := v1.MachineIpmiReportResponse{
		Updated: v1.Leases{},
		Created: v1.Leases{},
	}
	// create empty machines for uuids that are not yet known to the metal-api
	const defaultIPMIPort = "623"
	for uuid, ip := range requestPayload.Leases {
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
			PartitionID: requestPayload.PartitionID,
			IPMI: metal.IPMI{
				Address: ip + ":" + defaultIPMIPort,
			},
		}
		err = r.ds.CreateMachine(m)
		if err != nil {
			logger.Errorf("could not create machine", "id", uuid, "ipmi-ip", ip, "m", m, "err", err)
			continue
		}
		resp.Created[uuid] = ip
	}
	// update machine ipmi data if ipmi ip changed
	for _, oldMachine := range ms {
		uuid := oldMachine.ID
		if uuid == "" {
			continue
		}
		// if oldmachine.uuid is not part of this update cycle skip it
		ip, ok := requestPayload.Leases[uuid]
		if !ok {
			continue
		}
		newMachine := oldMachine

		// Replace host part of ipmi address with the ip from the ipmicatcher
		hostAndPort := strings.Split(oldMachine.IPMI.Address, ":")
		if len(hostAndPort) == 2 {
			newMachine.IPMI.Address = ip + ":" + hostAndPort[1]
		} else if len(hostAndPort) < 2 {
			newMachine.IPMI.Address = ip + ":" + defaultIPMIPort
		} else {
			logger.Errorf("not updating ipmi, address is garbage", "id", uuid, "ip", ip, "machine", newMachine, "address", newMachine.IPMI.Address)
			continue
		}

		if newMachine.IPMI.Address == oldMachine.IPMI.Address {
			continue
		}
		// machine was created by a PXE boot event and has no partition set.
		if oldMachine.PartitionID == "" {
			newMachine.PartitionID = requestPayload.PartitionID
		}

		if newMachine.PartitionID != requestPayload.PartitionID {
			logger.Errorf("could not update machine because overlapping id found", "id", uuid, "machine", newMachine, "partition", requestPayload.PartitionID)
			continue
		}

		err = r.ds.UpdateMachine(&oldMachine, &newMachine)
		if err != nil {
			logger.Errorf("could not update machine", "id", uuid, "ip", ip, "machine", newMachine, "err", err)
			continue
		}
		resp.Updated[uuid] = ip
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, resp)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
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

	m, err := allocateMachine(r.ds, r.ipamer, &spec, r.mdc)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		// TODO: Trigger network garbage collection
		utils.Logger(request).Sugar().Errorf("machine allocation went wrong, triggered network garbage collection", "error", err)
		return
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, makeMachineResponse(m, r.ds, utils.Logger(request).Sugar()))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func allocateMachine(ds *datastore.RethinkStore, ipamer ipam.IPAMer, allocationSpec *machineAllocationSpec, mdc mdm.Client) (*metal.Machine, error) {
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

	machineCandidate, err := findMachineCandidate(ds, allocationSpec)
	if err != nil {
		return nil, err
	}
	// as some fields in the allocation spec are optional, they will now be clearly defined by the machine candidate
	allocationSpec.UUID = machineCandidate.ID
	allocationSpec.PartitionID = machineCandidate.PartitionID
	allocationSpec.SizeID = machineCandidate.SizeID

	networks, err := makeNetworks(ds, ipamer, allocationSpec)
	if err != nil {
		return nil, err
	}

	alloc := &metal.MachineAllocation{
		Created:         time.Now(),
		Name:            allocationSpec.Name,
		Description:     allocationSpec.Description,
		Hostname:        allocationSpec.Hostname,
		Project:         projectID,
		ImageID:         allocationSpec.Image.ID,
		UserData:        allocationSpec.UserData,
		SSHPubKeys:      allocationSpec.SSHPubKeys,
		MachineNetworks: getMachineNetworks(networks),
	}

	// refetch the machine to catch possible updates after dealing with the network...
	machine, err := ds.FindMachineByID(machineCandidate.ID)
	if err != nil {
		return nil, err
	}
	if machine.Allocation != nil {
		return nil, fmt.Errorf("machine %q already allocated", machine.ID)
	}

	old := *machine
	machine.Allocation = alloc
	machine.Tags = makeMachineTags(machine, networks, allocationSpec.Tags)

	err = ds.UpdateMachine(&old, machine)
	if err != nil {
		return nil, fmt.Errorf("error when allocating machine %q, %v", machine.ID, err)
	}

	err = ds.UpdateWaitingMachine(machine)
	if err != nil {
		updateErr := ds.UpdateMachine(machine, &old) // try rollback allocation
		if updateErr != nil {
			return nil, fmt.Errorf("during update rollback due to an error (%v), another error occurred: %v", err, updateErr)
		}
		return nil, fmt.Errorf("cannot allocate machine in DB: %v", err)
	}

	return machine, nil
}

func validateAllocationSpec(allocationSpec *machineAllocationSpec) error {
	if allocationSpec.ProjectID == "" {
		return fmt.Errorf("project id must be specified")
	}

	if allocationSpec.UUID == "" && allocationSpec.PartitionID == "" {
		return fmt.Errorf("when no machine id is given, a partition id must be specified")
	}

	if allocationSpec.UUID == "" && allocationSpec.SizeID == "" {
		return fmt.Errorf("when no machine id is given, a size id must be specified")
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
			return fmt.Errorf("when no ip is given at least one auto acquire network must be specified")
		}
	}

	if noautoNetN := allocationSpec.noautoNetworkN(); noautoNetN > len(allocationSpec.IPs) {
		return fmt.Errorf("missing ip(s) for network(s) without automatic ip allocation")
	}

	return nil
}

func findMachineCandidate(ds *datastore.RethinkStore, allocationSpec *machineAllocationSpec) (*metal.Machine, error) {
	var err error
	var machine *metal.Machine
	if allocationSpec.UUID == "" {
		// requesting allocation of an arbitrary machine in partition with given size
		machine, err = findAvailableMachine(ds, allocationSpec.PartitionID, allocationSpec.SizeID)
		if err != nil {
			return nil, err
		}
	} else {
		// requesting allocation of a specific, existing machine
		machine, err = ds.FindMachineByID(allocationSpec.UUID)
		if err != nil {
			return nil, fmt.Errorf("machine cannot be found: %v", err)
		}

		if machine.Allocation != nil {
			return nil, fmt.Errorf("machine is already allocated")
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

func findAvailableMachine(ds *datastore.RethinkStore, partitionID, sizeID string) (*metal.Machine, error) {
	size, err := ds.FindSize(sizeID)
	if err != nil {
		return nil, fmt.Errorf("size cannot be found: %v", err)
	}
	partition, err := ds.FindPartition(partitionID)
	if err != nil {
		return nil, fmt.Errorf("partition cannot be found: %v", err)
	}
	machine, err := ds.FindAvailableMachine(partition.ID, size.ID)
	if err != nil {
		return nil, err
	}
	return machine, nil
}

func makeNetworks(ds *datastore.RethinkStore, ipamer ipam.IPAMer, allocationSpec *machineAllocationSpec) (allocationNetworkMap, error) {
	networks, err := gatherNetworks(ds, ipamer, allocationSpec)
	if err != nil {
		return nil, err
	}

	for _, n := range networks {
		machineNetwork, err := makeMachineNetwork(ds, ipamer, allocationSpec, n)
		if err != nil {
			return nil, err
		}
		n.machineNetwork = machineNetwork
	}

	// the metal-networker expects to have the same unique ASN on all networks of this machine
	asn, err := makeASN(networks)
	if err != nil {
		return nil, err
	}
	for _, n := range networks {
		n.machineNetwork.ASN = asn
	}

	return networks, nil
}

func gatherNetworks(ds *datastore.RethinkStore, ipamer ipam.IPAMer, allocationSpec *machineAllocationSpec) (allocationNetworkMap, error) {
	partition, err := ds.FindPartition(allocationSpec.PartitionID)
	if err != nil {
		return nil, fmt.Errorf("partition cannot be found: %v", err)
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
		underlayNetwork, err = gatherUnderlayNetwork(ds, allocationSpec, partition)
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
	for _, privateSuperNetwork := range privateSuperNetworks {
		if partition.ID == privateSuperNetwork.PartitionID {
			partitionPrivateSuperNetwork = &privateSuperNetwork
			break
		}
	}
	if partitionPrivateSuperNetwork == nil {
		return nil, fmt.Errorf("partition %s does not have a private super network", partition.ID)
	}

	// what do we have to prevent:
	// - user wants to place his machine in a network that does not belong to the project in which the machine is being placed
	// - user wants a machine with a private network that is not in the partition of the machine
	// - user wants to define multiple private networks for his machine
	// - user must define one private network
	// - user specifies administrative networks, i.e. underlay or privatesuper networks
	// - user's private network is specified with noauto, which would make the machine have no ip address

	specNetworks := make(map[string]*allocationNetwork)
	var privateNetwork *allocationNetwork

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
			network:   network,
			auto:      auto,
			ips:       []metal.IP{},
			isPrivate: false,
		}

		for _, privateSuperNetwork := range privateSuperNetworks {
			if network.ParentNetworkID == privateSuperNetwork.ID {
				// this is the user given private network
				if privateNetwork != nil {
					return nil, fmt.Errorf("multiple private networks provided, which is not allowed")
				}
				if network.PartitionID != partitionPrivateSuperNetwork.PartitionID {
					return nil, fmt.Errorf("the private network must be in the partition where the machine is going to be placed")
				}
				n.isPrivate = true
				privateNetwork = n
				break
			}
		}

		specNetworks[network.ID] = n
	}

	if len(specNetworks) != len(allocationSpec.Networks) {
		return nil, fmt.Errorf("given network ids are not unique")
	}

	if privateNetwork == nil {
		return nil, fmt.Errorf("no private network given")
	}

	if privateNetwork.network.ProjectID != allocationSpec.ProjectID {
		return nil, fmt.Errorf("the given private network does not belong to the project, which is not allowed")
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

	if !privateNetwork.auto && len(privateNetwork.ips) == 0 {
		return nil, fmt.Errorf("the private network has no auto ip acquisition, but no suitable IPs were provided, which would lead into a machine having no ip address")
	}

	return specNetworks, nil
}

func gatherUnderlayNetwork(ds *datastore.RethinkStore, allocationSpec *machineAllocationSpec, partition *metal.Partition) (*allocationNetwork, error) {
	boolTrue := true
	var underlays metal.Networks
	err := ds.SearchNetworks(&datastore.NetworkSearchQuery{PartitionID: &partition.ID, Underlay: &boolTrue}, &underlays)
	if err != nil {
		return nil, err
	}
	if len(underlays) == 0 {
		return nil, fmt.Errorf("no underlay found in the given partition: %v", err)
	}
	if len(underlays) > 1 {
		return nil, fmt.Errorf("more than one underlay network in partition %s in the database, which should not be the case", partition.ID)
	}
	underlay := &underlays[0]

	return &allocationNetwork{
		network:   underlay,
		auto:      true,
		isPrivate: false,
	}, nil
}

func makeMachineNetwork(ds *datastore.RethinkStore, ipamer ipam.IPAMer, allocationSpec *machineAllocationSpec, n *allocationNetwork) (*metal.MachineNetwork, error) {
	if n.auto {
		ipAddress, ipParentCidr, err := allocateIP(n.network, "", ipamer)
		if err != nil {
			return nil, fmt.Errorf("unable to allocate an ip in network: %s %#v", n.network.ID, err)
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

	ipAddresses := []string{}
	for _, ip := range n.ips {
		new := ip
		new.AddMachineId(allocationSpec.UUID)
		err := ds.UpdateIP(&ip, &new)
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
		Private:             n.isPrivate,
		Underlay:            n.network.Underlay,
		Nat:                 n.network.Nat,
		Vrf:                 n.network.Vrf,
	}

	return &machineNetwork, nil
}

// makeASN we can use the IP of the private network (which always have to be present and unique)
// for generating a unique ASN.
func makeASN(networks allocationNetworkMap) (int64, error) {
	privateNetwork, err := getPrivateNetwork(networks)
	if err != nil {
		return 0, err
	}

	if len(privateNetwork.ips) == 0 {
		return 0, fmt.Errorf("private network has no IPs, which would result in a machine without an IP")
	}

	asn, err := privateNetwork.ips[0].ASN()
	if err != nil {
		return 0, err
	}

	return asn, nil
}

// makeMachineTags constructs the tags of the machine.
// following tags are added in the following precedence (from lowest to highest in case of duplication):
// - external network labels (concatenated, from all machine networks that this machine belongs to)
// - private network labels (concatenated)
// - user given tags (from allocation spec)
// - system tags (immutable information from the metal-api that are useful for the end user, e.g. machine rack and chassis)
func makeMachineTags(m *metal.Machine, networks allocationNetworkMap, userTags []string) []string {
	labels := make(map[string]string)

	for _, n := range networks {
		if !n.isPrivate {
			for k, v := range n.network.Labels {
				labels[k] = v
			}
		}
	}

	privateNetwork, _ := getPrivateNetwork(networks)
	if privateNetwork != nil {
		for k, v := range privateNetwork.network.Labels {
			labels[k] = v
		}
	}

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
				labels[tag.MachineNetworkPrimaryASN] = strconv.FormatInt(n.ASN, 10)
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

	var vrf = ""
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

	err = freeMachine(r.ds, r, r.ipamer, m, logger)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = response.WriteHeaderAndEntity(http.StatusOK, makeMachineResponse(m, r.ds, logger))
	if err != nil {
		logger.Error("Failed to send response", zap.Error(err))
	}
}

func freeMachine(ds *datastore.RethinkStore, publisher bus.Publisher, ipamer ipam.IPAMer, m *metal.Machine, logger *zap.SugaredLogger) error {
	if m.State.Value == metal.LockedState {
		return fmt.Errorf("machine is locked")
	}

	err := deleteVRFSwitches(ds, m, logger)
	if err != nil {
		return err
	}
	err = publishDeleteEvent(publisher, m, logger)
	if err != nil {
		return err
	}

	// if the machine is allocated, we free it in our database
	if m.Allocation != nil {
		err = releaseMachineNetworks(ds, ipamer, m)
		if err != nil {
			// TODO: Trigger network garbage collection
			// TODO: Check if all IPs in rethinkdb are in the IPAM and vice versa, cleanup if this is not the case
			// TODO: Check if there are network prefixes in the IPAM that are not in any of our networks
			logger.Errorw("an error during releasing machine networks occurred, scheduled network garbage collection", "error", err)
		}
	}

	old := *m

	m.Allocation = nil
	m.Tags = nil

	err = ds.UpdateMachine(&old, m)
	if err != nil {
		return err
	}
	logger.Infow("freed machine", "machineID", m.ID)

	return nil
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

	logger := utils.Logger(request).Sugar()

	if m.Allocation != nil && m.State.Value != metal.LockedState {
		old := *m

		m.Allocation.Reinstall = true
		m.Allocation.ImageID = requestPayload.ImageID

		resp := makeMachineResponse(m, r.ds, logger)
		if resp.Allocation.Image != nil {
			err = r.ds.UpdateMachine(&old, m)
			if checkError(request, response, utils.CurrentFuncName(), err) {
				return
			}
			logger.Infow("marked machine to get reinstalled", "machineID", m.ID)

			err = deleteVRFSwitches(r.ds, m, logger)
			if checkError(request, response, utils.CurrentFuncName(), err) {
				return
			}

			err = publishDeleteEvent(r, m, logger)
			if checkError(request, response, utils.CurrentFuncName(), err) {
				return
			}

			err = publishMachineCmd(logger, m, r, metal.MachineReinstall)
			if err != nil {
				logger.Errorw("unable to publish machine command", "command", metal.MachineReinstall, "machineID", m.ID, "error", err)
			}

			err = response.WriteHeaderAndEntity(http.StatusOK, resp)
			if err != nil {
				logger.Error("Failed to send response", zap.Error(err))
			}

			return
		}
	}

	err = response.WriteHeaderAndEntity(http.StatusBadRequest, httperrors.NewHTTPError(http.StatusBadRequest, fmt.Errorf("machine either locked, not allocated yet or invalid image ID specified")))
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

func deleteVRFSwitches(ds *datastore.RethinkStore, m *metal.Machine, logger *zap.SugaredLogger) error {
	_, err := setVrfAtSwitches(ds, m, "")
	logger.Infow("set VRF at switch", "machineID", m.ID, "error", err)
	return err
}

func publishDeleteEvent(publisher bus.Publisher, m *metal.Machine, logger *zap.SugaredLogger) error {
	deleteEvent := metal.MachineEvent{Type: metal.DELETE, OldMachineID: m.ID}
	err := publisher.Publish(metal.TopicMachine.GetFQN(m.PartitionID), deleteEvent)
	logger.Infow("published machine delete event", "machineID", m.ID, "error", err)
	return err
}

func releaseMachineNetworks(ds *datastore.RethinkStore, ipamer ipam.IPAMer, machine *metal.Machine) error {
	for _, machineNetwork := range machine.Allocation.MachineNetworks {
		for _, ipString := range machineNetwork.IPs {
			ip, err := ds.FindIPByID(ipString)
			if err != nil {
				return err
			}
			// ignore ips that were associated with the machine for allocation but the association is not present anymore at the ip
			if !ip.HasMachineId(machine.GetID()) {
				continue
			}
			// disassociate machine from ip
			newIP := *ip
			newIP.RemoveMachineId(machine.GetID())
			err = ds.UpdateIP(ip, &newIP)
			if err != nil {
				return err
			}
			// static ips should not be released automatically
			if ip.Type == metal.Static {
				continue
			}
			// ips that are associated to other machines will should not be released automatically
			if len(newIP.GetMachineIds()) > 0 {
				continue
			}
			// release and delete
			err = ipamer.ReleaseIP(*ip)
			if err != nil {
				return err
			}
			err = ds.DeleteIP(ip)
			if err != nil {
				return err
			}
		}
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
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("unknown provisioning event")) {
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
	errors := 0
	for _, m := range machines {
		p := liveliness[m.PartitionID]
		lvlness, err := evaluateMachineLiveliness(ds, m)
		if err != nil {
			logger.Errorw("cannot update liveliness", "error", err, "machine", m)
			errors++
			// fall through, so the rest of the machines is getting evaluated
		}
		switch lvlness {
		case metal.MachineLivelinessAlive:
			alive++
			p.Alive++
		case metal.MachineLivelinessDead:
			dead++
			p.Dead++
		default:
			unknown++
			p.Unknown++
		}
		liveliness[m.PartitionID] = p
	}

	metrics.ProvideLiveliness(liveliness)

	logger.Infow("machine liveliness evaluated", "alive", alive, "dead", dead, "unknown", unknown, "errors", errors)

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
func ResurrectMachines(ds *datastore.RethinkStore, publisher bus.Publisher, ipamer ipam.IPAMer, logger *zap.SugaredLogger) error {
	logger.Info("machine resurrection was requested")

	machines, err := ds.ListMachines()
	if err != nil {
		return err
	}

	for _, m := range machines {
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
		err = freeMachine(ds, publisher, ipamer, &m, logger)
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

func (r machineResource) chassisIdentifyLEDOn(request *restful.Request, response *restful.Response) {
	r.machineCmd("chassisIdentifyLEDOn", metal.ChassisIdentifyLEDOnCmd, request, response, request.PathParameter("description"))
}

func (r machineResource) chassisIdentifyLEDOff(request *restful.Request, response *restful.Response) {
	r.machineCmd("chassisIdentifyLEDOff", metal.ChassisIdentifyLEDOffCmd, request, response, request.PathParameter("description"))
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

	err = publishMachineCmd(logger, m, r, cmd, params...)
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
			i, err = ds.FindImage(m.Allocation.ImageID)
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
	for _, net := range s.Networks {
		if net.AutoAcquireIP != nil && !*net.AutoAcquireIP {
			result++
		}
	}
	return result
}

func (s machineAllocationSpec) autoNetworkN() int {
	return len(s.Networks) - s.noautoNetworkN()
}
