package service

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"golang.org/x/crypto/ssh"

	"git.f-i-ts.de/cloud-native/metallib/httperrors"
	"go.uber.org/zap"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/ipam"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	v1 "git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/service/v1"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/utils"

	"git.f-i-ts.de/cloud-native/metallib/bus"
	"github.com/dustin/go-humanize"
	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
)

const (
	waitForServerTimeout = 30 * time.Second
)

type machineResource struct {
	webResource
	bus.Publisher
	ipamer ipam.IPAMer
}

type machineAllocationSpec struct {
	UUID        string
	Name        string
	Description string
	Tenant      string
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

type allocationNetwork struct {
	network         *metal.Network
	ips             []metal.IP
	auto            bool
	isTenantNetwork bool
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
	ipamer ipam.IPAMer) *restful.WebService {
	r := machineResource{
		webResource: webResource{
			ds: ds,
		},
		Publisher: pub,
		ipamer:    ipamer,
	}
	return r.webService()
}

// webService creates the webservice endpoint
func (r machineResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path("/v1/machine").
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
		Doc("search machines").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.FindMachinesRequest{}).
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

	ws.Route(ws.DELETE("/{id}/free").
		To(editor(r.freeMachine)).
		Operation("freeMachine").
		Doc("free a machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/{id}/ipmi").
		To(viewer(r.ipmiData)).
		Operation("ipmiData").
		Doc("returns the IPMI connection data for a machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(http.StatusOK, "OK", v1.MachineIPMI{}).
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

	ws.Route(ws.POST("/liveliness").
		To(r.checkMachineLiveliness).
		Operation("checkMachineLiveliness").
		Doc("external trigger for evaluating machine liveliness").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.EmptyBody{}).
		Returns(http.StatusOK, "OK", v1.MachineLivelinessReport{}).
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

	return ws
}

func (r machineResource) listMachines(request *restful.Request, response *restful.Response) {
	ms, err := r.ds.ListMachines()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, makeMachineResponseList(ms, r.ds, utils.Logger(request).Sugar()))
}

func (r machineResource) findMachine(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	m, err := r.ds.FindMachine(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, makeMachineResponse(m, r.ds, utils.Logger(request).Sugar()))
}

func (r machineResource) findMachines(request *restful.Request, response *restful.Response) {
	var requestPayload v1.FindMachinesRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	ms, err := r.ds.FindMachines(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, makeMachineResponseList(ms, r.ds, utils.Logger(request).Sugar()))
}

func (r machineResource) waitForAllocation(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	ctx := request.Request.Context()
	log := utils.Logger(request)

	err := r.wait(id, log.Sugar(), func(alloc Allocation) error {
		select {
		case <-time.After(waitForServerTimeout):
			response.WriteHeaderAndEntity(http.StatusGatewayTimeout, httperrors.NewHTTPError(http.StatusGatewayTimeout, fmt.Errorf("server timeout")))
		case a := <-alloc:
			if a.Err != nil {
				log.Sugar().Errorw("allocation returned an error", "error", a.Err)
				return a.Err
			}

			s, p, i, ec := findMachineReferencedEntites(a.Machine, r.ds, log.Sugar())
			response.WriteHeaderAndEntity(http.StatusOK, v1.NewMachineResponse(a.Machine, s, p, i, ec))
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
func (r machineResource) wait(id string, logger *zap.SugaredLogger, alloc Allocator) error {
	m, err := r.ds.FindMachine(id)
	if err != nil {
		return err
	}
	a := make(chan MachineAllocation)

	// the machine IS already allocated, so notify this allocation back.
	if m.Allocation != nil {
		go func() {
			a <- MachineAllocation{Machine: m}
		}()
		return alloc(a)
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
		changedMachine, err := r.ds.WaitForMachineAllocation(m)
		if err != nil {
			a <- MachineAllocation{Err: err}
		} else {
			a <- MachineAllocation{Machine: changedMachine}
		}
		close(a)
	}()

	return alloc(a)
}

func (r machineResource) setMachineState(request *restful.Request, response *restful.Response) {
	var requestPayload v1.MachineState
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	machineStateType, err := metal.MachineStateFrom(requestPayload.Value)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	if machineStateType != metal.AvailableState && requestPayload.Description == "" {
		// we want a cause why this machine is not available
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("you must supply a description")) {
			return
		}
	}

	id := request.PathParameter("id")
	oldMachine, err := r.ds.FindMachine(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	newMachine := *oldMachine

	newMachine.State = metal.MachineState{
		Value:       machineStateType,
		Description: requestPayload.Description,
	}

	err = r.ds.UpdateMachine(oldMachine, &newMachine)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, makeMachineResponse(&newMachine, r.ds, utils.Logger(request).Sugar()))
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

	m, err := r.ds.FindMachine(requestPayload.UUID)
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
			State: metal.MachineState{
				Value:       metal.AvailableState,
				Description: "",
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
		m.Tags = requestPayload.Tags
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

	response.WriteHeaderAndEntity(returnCode, makeMachineResponse(m, r.ds, utils.Logger(request).Sugar()))
}

func (r machineResource) ipmiData(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	m, err := r.ds.FindMachine(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, v1.NewMachineIPMI(&m.IPMI))
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
		Tenant:      requestPayload.Tenant,
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

	m, err := allocateMachine(r.ds, r.ipamer, &spec)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		// TODO: Trigger network garbage collection
		utils.Logger(request).Sugar().Errorf("machine allocation went wrong, triggered network garbage collection", "error", err)
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, makeMachineResponse(m, r.ds, utils.Logger(request).Sugar()))
}

func allocateMachine(ds *datastore.RethinkStore, ipamer ipam.IPAMer, allocationSpec *machineAllocationSpec) (*metal.Machine, error) {
	err := validateAllocationSpec(allocationSpec)
	if err != nil {
		return nil, err
	}

	machineCandidate, err := findMachineCandidate(ds, allocationSpec)
	if err != nil {
		return nil, err
	}
	// as some fields in the allocation spec are optional, they will now be clearly defined by the machine candidate
	allocationSpec.UUID = machineCandidate.ID
	allocationSpec.PartitionID = machineCandidate.PartitionID
	allocationSpec.SizeID = machineCandidate.SizeID

	machineNetworks, err := makeMachineNetworks(ds, ipamer, allocationSpec)
	if err != nil {
		return nil, err
	}

	alloc := &metal.MachineAllocation{
		Created:         time.Now(),
		Name:            allocationSpec.Name,
		Description:     allocationSpec.Description,
		Hostname:        allocationSpec.Hostname,
		Tenant:          allocationSpec.Tenant,
		Project:         allocationSpec.ProjectID,
		ImageID:         allocationSpec.Image.ID,
		SSHPubKeys:      allocationSpec.SSHPubKeys,
		UserData:        allocationSpec.UserData,
		MachineNetworks: machineNetworks,
	}

	// refetch the machine to catch possible updates after dealing with the network...
	machine, err := ds.FindMachine(machineCandidate.ID)
	if err != nil {
		return nil, err
	}

	old := *machine
	machine.Allocation = alloc
	tags := allocationSpec.Tags
	additionalTags := additionalTags(machine)
	tags = append(tags, additionalTags...)
	machine.Tags = uniqueTags(tags)

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

func additionalTags(machine *metal.Machine) []string {
	const tagSuffix = "machine.metal-pod.io"
	tags := []string{}
	for _, n := range machine.Allocation.MachineNetworks {
		if n.Primary {
			if n.ASN != 0 {
				tags = append(tags, fmt.Sprintf("asn.primary.network.%s=%d", tagSuffix, n.ASN))
			}
			if len(n.IPs) < 1 {
				continue
			}
			ip, _, error := net.ParseCIDR(n.IPs[0])
			if error != nil {
				continue
			}
			// Set the last octet to "0" regardles of version
			ip[len(ip)-1] = 0
			tags = append(tags, fmt.Sprintf("ip.localbgp.primary.network.%s=%s/32", tagSuffix, ip))
		}
	}
	if machine.RackID != "" {
		tags = append(tags, fmt.Sprintf("rack.%s=%s", tagSuffix, machine.RackID))
	}
	if machine.IPMI.Fru.ChassisPartSerial != "" {
		tags = append(tags, fmt.Sprintf("chassis.%s=%s", tagSuffix, machine.IPMI.Fru.ChassisPartSerial))
	}
	return tags
}

func validateAllocationSpec(allocationSpec *machineAllocationSpec) error {
	if allocationSpec.Tenant == "" {
		return fmt.Errorf("no tenant given")
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
		machine, err = ds.FindMachine(allocationSpec.UUID)
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

func makeMachineNetworks(ds *datastore.RethinkStore, ipamer ipam.IPAMer, allocationSpec *machineAllocationSpec) ([]*metal.MachineNetwork, error) {
	networks, err := gatherNetworks(ds, ipamer, allocationSpec)
	if err != nil {
		return nil, err
	}

	machineNetworks := []*metal.MachineNetwork{}
	for _, n := range networks {
		machineNetwork, err := makeMachineNetwork(ds, ipamer, allocationSpec, n)
		if err != nil {
			return nil, err
		}
		machineNetworks = append(machineNetworks, machineNetwork)
	}

	// TODO: It's duplicated information, but the ASN of the tenant network must be present on all other networks...
	// This needs to be done more elegantly, this is a quick fix:
	var asn int64
	for _, n := range machineNetworks {
		if n.ASN != 0 {
			asn = n.ASN
			break
		}
	}
	fmt.Printf("Halo: %v", asn)
	for _, n := range machineNetworks {
		n.ASN = asn
	}

	return machineNetworks, nil
}

func gatherNetworks(ds *datastore.RethinkStore, ipamer ipam.IPAMer, allocationSpec *machineAllocationSpec) (map[string]*allocationNetwork, error) {
	partition, err := ds.FindPartition(allocationSpec.PartitionID)
	if err != nil {
		return nil, fmt.Errorf("partition cannot be found: %v", err)
	}

	userNetworks, err := gatherUserNetworks(ds, allocationSpec, partition)
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

	projectNetwork, err := gatherOrCreateProjectNetwork(ds, ipamer, allocationSpec, partition)
	if err != nil {
		return nil, err
	}

	gatheredNetworks := make(map[string]*allocationNetwork)
	for k, v := range userNetworks {
		gatheredNetworks[k] = v
	}
	if underlayNetwork != nil {
		gatheredNetworks[underlayNetwork.network.ID] = underlayNetwork
	}
	gatheredNetworks[projectNetwork.network.ID] = projectNetwork

	return gatheredNetworks, nil
}

func gatherUserNetworks(ds *datastore.RethinkStore, allocationSpec *machineAllocationSpec, partition *metal.Partition) (map[string]*allocationNetwork, error) {
	isPrimary := true
	primaryNetworks, err := ds.FindNetworks(&v1.FindNetworksRequest{Primary: &isPrimary})
	if err != nil {
		return nil, err
	}

	userNetworks := make(map[string]*allocationNetwork)

	for _, n := range allocationSpec.Networks {
		network, err := ds.FindNetwork(n.NetworkID)
		if err != nil {
			return nil, err
		}

		if network.Underlay {
			return nil, fmt.Errorf("underlay networks are not allowed to be set explicitly: %s", network.ID)
		}
		if network.Primary {
			return nil, fmt.Errorf("primary networks are not allowed to be set explicitly: %s", network.ID)
		}
		for _, primaryNetwork := range primaryNetworks {
			if network.ParentNetworkID == primaryNetwork.ID {
				return nil, fmt.Errorf("given network %q must not be a child of a primary network: %s", network.ID, primaryNetwork.ID)
			}
		}

		auto := true
		if n.AutoAcquireIP != nil {
			auto = *n.AutoAcquireIP
		}

		userNetworks[network.ID] = &allocationNetwork{
			network:         network,
			auto:            auto,
			ips:             []metal.IP{},
			isTenantNetwork: false,
		}
	}

	if len(userNetworks) != len(allocationSpec.Networks) {
		return nil, fmt.Errorf("given network ids are not unique: %v", allocationSpec.Networks)
	}

	for _, ipString := range allocationSpec.IPs {
		ip, err := ds.FindIP(ipString)
		if err != nil {
			return nil, err
		}
		if ip.ProjectID != allocationSpec.ProjectID {
			return nil, fmt.Errorf("given ip %q with project %q does not belong to the project of this allocation: %s", ip.IPAddress, ip.ProjectID, allocationSpec.ProjectID)
		}
		network, ok := userNetworks[ip.NetworkID]
		if !ok {
			return nil, fmt.Errorf("given ip %q is not in any of the given networks, which is required", ip.IPAddress)
		}
		network.ips = append(network.ips, *ip)
	}

	return userNetworks, nil
}

func gatherUnderlayNetwork(ds *datastore.RethinkStore, allocationSpec *machineAllocationSpec, partition *metal.Partition) (*allocationNetwork, error) {
	underlays, err := ds.FindUnderlayNetworks(partition.ID)
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
		network:         underlay,
		auto:            true,
		isTenantNetwork: false,
	}, nil
}

func gatherOrCreateProjectNetwork(ds *datastore.RethinkStore, ipamer ipam.IPAMer, allocationSpec *machineAllocationSpec, partition *metal.Partition) (*allocationNetwork, error) {
	// TODO: The vrf should become an attribute of a project and created on project creation
	vrfID, err := ensureProjectVrf(ds, allocationSpec)
	if err != nil {
		return nil, err
	}

	projectNetwork, err := ds.FindProjectNetwork(allocationSpec.ProjectID)
	if err != nil && !metal.IsNotFound(err) {
		return nil, err
	}

	if projectNetwork == nil {
		primaryNetwork, err := ds.FindPrimaryNetwork(partition.ID)
		if err != nil {
			return nil, fmt.Errorf("could not get primary network: %v", err)
		}

		projectPrefix, err := createChildPrefix(primaryNetwork.Prefixes, partition.ProjectNetworkPrefixLength, ipamer)
		if err != nil {
			return nil, err
		}
		if projectPrefix == nil {
			return nil, fmt.Errorf("could not allocate child prefix in network: %s", primaryNetwork.ID)
		}

		projectNetwork = &metal.Network{
			Prefixes:            metal.Prefixes{*projectPrefix},
			DestinationPrefixes: metal.Prefixes{},
			PartitionID:         allocationSpec.PartitionID,
			ProjectID:           allocationSpec.ProjectID,
			Nat:                 primaryNetwork.Nat,
			Primary:             false,
			Underlay:            false,
			Vrf:                 vrfID,
			ParentNetworkID:     primaryNetwork.ID,
		}

		err = ds.CreateNetwork(projectNetwork)
		if err != nil {
			return nil, err
		}
	}

	return &allocationNetwork{
		network:         projectNetwork,
		auto:            true,
		isTenantNetwork: true,
	}, nil
}

func ensureProjectVrf(ds *datastore.RethinkStore, allocationSpec *machineAllocationSpec) (uint, error) {
	vrf, err := ds.FindVrfByProject(allocationSpec.ProjectID, allocationSpec.Tenant)
	if err != nil {
		if metal.IsNotFound(err) {
			vrf, err = ds.ReserveNewVrf(allocationSpec.Tenant, allocationSpec.ProjectID)
			if err != nil {
				return 0, fmt.Errorf("cannot reserve new vrf for tenant project: %v", err)
			}
		} else {
			return 0, fmt.Errorf("cannot find vrf for tenant project: %v", err)
		}
	}

	vrfID, err := vrf.ToUint()
	if err != nil {
		return 0, fmt.Errorf("cannot convert vrfid to uint: %v", err)
	}

	return vrfID, nil
}

func makeMachineNetwork(ds *datastore.RethinkStore, ipamer ipam.IPAMer, allocationSpec *machineAllocationSpec, n *allocationNetwork) (*metal.MachineNetwork, error) {
	var asn int64
	primary := false

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
			MachineID:        allocationSpec.UUID,
			ProjectID:        allocationSpec.ProjectID,
		}

		err = ds.CreateIP(ip)
		if err != nil {
			return nil, err
		}

		if n.isTenantNetwork {
			asn, err = ip.ASN()
			if err != nil {
				return nil, err
			}
			primary = true
		}

		n.ips = append(n.ips, *ip)
	}

	ipAddresses := []string{}
	for _, ip := range n.ips {
		ipAddresses = append(ipAddresses, ip.IPAddress)
	}

	machineNetwork := metal.MachineNetwork{
		NetworkID:           n.network.ID,
		Prefixes:            n.network.Prefixes.String(),
		IPs:                 ipAddresses,
		DestinationPrefixes: n.network.DestinationPrefixes.String(),
		ASN:                 asn,
		Primary:             primary,
		Underlay:            n.network.Underlay,
		Nat:                 n.network.Nat,
		Vrf:                 n.network.Vrf,
	}

	return &machineNetwork, nil
}

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
	m, err := r.ds.FindMachine(id)
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

	err = r.ds.UpdateMachine(&old, m)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	var sws []metal.Switch
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
			if mn.Primary {
				vrf = fmt.Sprintf("vrf%d", mn.Vrf)
				break
			}
		}
	}

	sws, err = setVrfAtSwitches(r.ds, m, vrf)
	if err != nil {
		if m.Allocation == nil {
			if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("the machine %q could not be enslaved into the vrf %s", id, vrf)) {
				return
			}
		}
	}

	if len(sws) > 0 {
		// Push out events to signal switch configuration change
		evt := metal.SwitchEvent{Type: metal.UPDATE, Machine: *m, Switches: sws}
		err = r.Publish(metal.TopicSwitch.GetFQN(m.PartitionID), evt)
		utils.Logger(request).Sugar().Infow("published switch update event", "event", evt, "error", err)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
	}

	response.WriteHeaderAndEntity(http.StatusOK, makeMachineResponse(m, r.ds, utils.Logger(request).Sugar()))
}

func (r machineResource) freeMachine(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	m, err := r.ds.FindMachine(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	if m.State.Value == metal.LockedState {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("machine is locked")) {
			return
		}
	}

	if m.Allocation != nil {
		// if the machine is allocated, we free it in our database
		err = r.releaseMachineNetworks(m, m.Allocation.MachineNetworks)
		if err != nil {
			// TODO: Trigger network garbage collection
			utils.Logger(request).Sugar().Errorf("an error during releasing machine networks occurred, scheduled network garbage collection", "error", err)
		}

		old := *m
		m.Allocation = nil
		err = r.ds.UpdateMachine(&old, m)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
		utils.Logger(request).Sugar().Infow("freed machine", "machineID", id)
	}

	// do the next steps in any case, so a client can call this function multiple times to
	// fire of the needed events

	sw, err := setVrfAtSwitches(r.ds, m, "")
	utils.Logger(request).Sugar().Infow("set VRF at switch", "machineID", id, "error", err)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	deleteEvent := metal.MachineEvent{Type: metal.DELETE, Old: m}
	err = r.Publish(metal.TopicMachine.GetFQN(m.PartitionID), deleteEvent)
	utils.Logger(request).Sugar().Infow("published machine delete event", "machineID", id, "error", err)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	switchEvent := metal.SwitchEvent{Type: metal.UPDATE, Machine: *m, Switches: sw}
	err = r.Publish(metal.TopicSwitch.GetFQN(m.PartitionID), switchEvent)
	utils.Logger(request).Sugar().Infow("published switch update event", "machineID", id, "error", err)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, makeMachineResponse(m, r.ds, utils.Logger(request).Sugar()))
}

func (r machineResource) releaseMachineNetworks(machine *metal.Machine, machineNetworks []*metal.MachineNetwork) error {
	for _, machineNetwork := range machineNetworks {
		for _, ipString := range machineNetwork.IPs {
			ip, err := r.ds.FindIP(ipString)
			if err != nil {
				return err
			}
			err = r.ipamer.ReleaseIP(*ip)
			if err != nil {
				return err
			}
			err = r.ds.DeleteIP(ip)
			if err != nil {
				return err
			}
		}

		network, err := r.ds.FindNetwork(machineNetwork.NetworkID)
		if err != nil {
			return err
		}
		// Only Networks must be deleted which are "owned" by this machine.
		if network.ProjectID != machine.Allocation.Project {
			continue
		}
		deleteNetwork := false
		for _, prefix := range network.Prefixes {
			usage, err := r.ipamer.PrefixUsage(prefix.String())
			if err != nil {
				return err
			}
			if usage.UsedIPs <= 2 { // 2 is for broadcast and network
				err = r.ipamer.ReleaseChildPrefix(prefix)
				if err != nil {
					return err
				}
				deleteNetwork = true
			}
		}
		if deleteNetwork {
			err = r.ds.DeleteNetwork(network)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func networkGarbageCollection(ds *datastore.RethinkStore, ipamer ipam.IPAMer) {
	// TODO: Check if all IPs in rethinkdb are in the IPAM and vice versa, cleanup if this is not the case
	// TODO: Check if there are network prefixes in the IPAM that are not in any of our networks
	// TODO: Check if there are tenant networks that do not have any IPs in use, clean them up
}

func (r machineResource) getProvisioningEventContainer(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	// check for existence of the machine
	_, err := r.ds.FindMachine(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	ec, err := r.ds.FindProvisioningEventContainer(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, v1.NewMachineRecentProvisioningEvents(ec))
}

func (r machineResource) addProvisioningEvent(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	m, err := r.ds.FindMachine(id)
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

	ec, err := r.ds.FindProvisioningEventContainer(m.ID)
	if err != nil && !metal.IsNotFound(err) {
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
	}

	if ec == nil {
		ec = &metal.ProvisioningEventContainer{
			Base: metal.Base{
				ID: m.ID,
			},
			Liveliness: metal.MachineLivelinessAlive,
		}
	}

	now := time.Now()

	ec.LastEventTime = &now

	event := metal.ProvisioningEvent{
		Time:    now,
		Event:   metal.ProvisioningEventType(requestPayload.Event),
		Message: requestPayload.Message,
	}
	if event.Event == metal.ProvisioningEventAlive {
		utils.Logger(request).Sugar().Debugw("received provisioning alive event", "id", ec.ID)
		ec.Liveliness = metal.MachineLivelinessAlive
	} else if event.Event == metal.ProvisioningEventPhonedHome && len(ec.Events) > 0 && ec.Events[0].Event == metal.ProvisioningEventPhonedHome {
		utils.Logger(request).Sugar().Debugw("swallowing repeated phone home event", "id", ec.ID)
		ec.Liveliness = metal.MachineLivelinessAlive
	} else {
		ec.Events = append([]metal.ProvisioningEvent{event}, ec.Events...)
		ec.IncompleteProvisioningCycles = ec.CalculateIncompleteCycles(utils.Logger(request).Sugar())
		ec.Liveliness = metal.MachineLivelinessAlive
	}
	ec.TrimEvents(metal.ProvisioningEventsInspectionLimit)

	err = r.ds.UpsertProvisioningEventContainer(ec)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, v1.NewMachineRecentProvisioningEvents(ec))
}

// EvaluateMachineLiveliness evaluates the liveliness of a given machine
func (r machineResource) evaluateMachineLiveliness(m metal.Machine) (metal.MachineLiveliness, error) {
	provisioningEvents, err := r.ds.FindProvisioningEventContainer(m.ID)
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
		err = r.ds.UpdateProvisioningEventContainer(&old, provisioningEvents)
		if err != nil {
			return provisioningEvents.Liveliness, err
		}
	}

	return provisioningEvents.Liveliness, nil
}

func (r machineResource) checkMachineLiveliness(request *restful.Request, response *restful.Response) {
	logger := utils.Logger(request).Sugar()
	logger.Info("liveliness report was requested")

	machines, err := r.ds.ListMachines()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	unknown := 0
	alive := 0
	dead := 0
	for _, m := range machines {
		lvlness, err := r.evaluateMachineLiveliness(m)
		if err != nil {
			logger.Errorw("cannot update liveliness", "error", err, "machine", m)
			// fall through, so the caller should get the evaulated state, although it is not persistet
		}
		switch lvlness {
		case metal.MachineLivelinessAlive:
			alive++
		case metal.MachineLivelinessDead:
			dead++
		default:
			unknown++
		}
	}

	report := v1.MachineLivelinessReport{
		AliveCount:   alive,
		DeadCount:    dead,
		UnknownCount: unknown,
	}

	response.WriteHeaderAndEntity(http.StatusOK, report)
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

func (r machineResource) machineCmd(op string, cmd metal.MachineCommand, request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	m, err := r.ds.FindMachine(id)
	if checkError(request, response, op, err) {
		return
	}
	evt := metal.MachineEvent{
		Type: metal.COMMAND,
		Cmd: &metal.MachineExecCommand{
			Command: cmd,
			Params:  []string{},
			Target:  m,
		},
	}

	err = r.Publish(metal.TopicMachine.GetFQN(m.PartitionID), evt)
	utils.Logger(request).Sugar().Infow("publish event", "event", evt, "command", *evt.Cmd, "error", err)
	if checkError(request, response, op, err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, makeMachineResponse(m, r.ds, utils.Logger(request).Sugar()))
}

func makeMachineResponse(m *metal.Machine, ds *datastore.RethinkStore, logger *zap.SugaredLogger) *v1.MachineResponse {
	s, p, i, ec := findMachineReferencedEntites(m, ds, logger)
	return v1.NewMachineResponse(m, s, p, i, ec)
}

func makeMachineResponseList(ms []metal.Machine, ds *datastore.RethinkStore, logger *zap.SugaredLogger) []*v1.MachineResponse {
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

func findMachineReferencedEntites(m *metal.Machine, ds *datastore.RethinkStore, logger *zap.SugaredLogger) (*metal.Size, *metal.Partition, *metal.Image, *metal.ProvisioningEventContainer) {
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
	start := time.Now()
	globalStart := start
	logger.Infow("getMachineReferencedEntityMaps", "start", start)
	s, err := ds.ListSizes()
	if err != nil {
		logger.Errorw("sizes could not be listed", "error", err)
	}
	logger.Infow("getMachineReferencedEntityMaps", "sizes", time.Now().Sub(start))

	start = time.Now()
	p, err := ds.ListPartitions()
	if err != nil {
		logger.Errorw("partitions could not be listed", "error", err)
	}
	logger.Infow("getMachineReferencedEntityMaps", "partitions", time.Now().Sub(start))
	start = time.Now()

	i, err := ds.ListImages()
	if err != nil {
		logger.Errorw("images could not be listed", "error", err)
	}
	logger.Infow("getMachineReferencedEntityMaps", "images", time.Now().Sub(start))
	start = time.Now()

	ec, err := ds.ListProvisioningEventContainers()
	if err != nil {
		logger.Errorw("provisioning event containers could not be listed", "error", err)
	}
	logger.Infow("getMachineReferencedEntityMaps", "events", time.Now().Sub(start), "total", time.Now().Sub(globalStart))

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

func (s machineAllocationSpec) isMachine() bool {
	return !s.IsFirewall
}
