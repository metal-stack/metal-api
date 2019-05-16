package service

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"git.f-i-ts.de/cloud-native/metallib/httperrors"
	"go.uber.org/zap"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/ipam"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	v1 "git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/service/v1"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/utils"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/utils/jwt"

	"git.f-i-ts.de/cloud-native/metallib/bus"
	"github.com/dustin/go-humanize"
	restful "github.com/emicklei/go-restful"
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
	NetworkIDs  []string
	IPs         []string
	HA          bool
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
		To(r.findMachine).
		Operation("findMachine").
		Doc("get machine by id").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.MachineResponse{}).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/").
		To(r.listMachines).
		Operation("listMachines").
		Doc("get all known machines").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]v1.MachineResponse{}).
		Returns(http.StatusOK, "OK", []v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/find").
		To(r.searchMachine).
		Doc("search machines").
		Param(ws.QueryParameter("mac", "one of the MAC address of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]v1.MachineResponse{}).
		Returns(http.StatusOK, "OK", []v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/register").
		To(r.registerMachine).
		Doc("register a machine").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.MachineRegisterRequest{}).
		Writes(v1.MachineResponse{}).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		Returns(http.StatusCreated, "Created", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/allocate").
		To(r.allocateMachine).
		Doc("allocate a machine").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.MachineAllocateRequest{}).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/finalize-allocation").
		To(r.finalizeAllocation).
		Doc("finalize the allocation of the machine by reconfiguring the switch, sent on successful image installation").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.MachineFinalizeAllocationRequest{}).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/state").
		To(r.setMachineState).
		Doc("set the state of a machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.MachineState{}).
		Writes(v1.MachineResponse{}).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.DELETE("/{id}/free").
		To(r.freeMachine).
		Doc("free a machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(http.StatusOK, "OK", v1.MachineResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/{id}/ipmi").
		To(r.ipmiData).
		Doc("returns the IPMI connection data for a machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(http.StatusOK, "OK", v1.MachineIPMI{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/{id}/wait").
		To(r.waitForAllocation).
		Doc("wait for an allocation of this machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(http.StatusOK, "OK", v1.MachineWaitResponse{}).
		Returns(http.StatusGatewayTimeout, "Timeout", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/{id}/event").
		To(r.getProvisioningEventContainer).
		Doc("get the current machine provisioning event container").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(http.StatusOK, "OK", v1.MachineRecentProvisioningEvents{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/event").
		To(r.addProvisioningEvent).
		Doc("adds a machine provisioning event").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.MachineProvisioningEvent{}).
		Returns(http.StatusOK, "OK", v1.MachineRecentProvisioningEvents{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/liveliness").
		To(r.checkMachineLiveliness).
		Doc("external trigger for evaluating machine liveliness").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(emptyBody{}).
		Returns(http.StatusOK, "OK", v1.MachineLivelinessReport{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/power/on").
		To(r.machineOn).
		Doc("sends a power-on to the machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(emptyBody{}).
		Returns(http.StatusOK, "OK", metal.MachineAllocation{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/power/off").
		To(r.machineOff).
		Doc("sends a power-off to the machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(emptyBody{}).
		Returns(http.StatusOK, "OK", metal.MachineAllocation{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/power/reset").
		To(r.machineReset).
		Doc("sends a reset to the machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(emptyBody{}).
		Returns(http.StatusOK, "OK", metal.MachineAllocation{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/power/bios").
		To(r.machineBios).
		Doc("boots machine into BIOS on next reboot").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(emptyBody{}).
		Returns(http.StatusOK, "OK", metal.MachineAllocation{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r machineResource) findMachine(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	m, err := r.ds.FindMachine(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, makeMachineResponse(m, r.ds, utils.Logger(request).Sugar()))
}

func (r machineResource) listMachines(request *restful.Request, response *restful.Response) {
	ms, err := r.ds.ListMachines()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, makeMachineResponseList(ms, r.ds, utils.Logger(request).Sugar()))
}

func (r machineResource) searchMachine(request *restful.Request, response *restful.Response) {
	mac := strings.TrimSpace(request.QueryParameter("mac"))

	ms, err := r.ds.SearchMachine(mac)
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
			response.WriteErrorString(http.StatusGatewayTimeout, "server timeout")
			return fmt.Errorf("server timeout")
		case a := <-alloc:
			if a.Err != nil {
				log.Sugar().Errorw("allocation returned an error", "error", a.Err)
				return a.Err
			}

			ka := jwt.NewPhoneHomeClaims(a.Machine)
			token, err := ka.JWT()
			if err != nil {
				return fmt.Errorf("could not create jwt: %v", err)
			}

			s, p, i, ec := findMachineReferencedEntites(a.Machine, r.ds, log.Sugar())
			response.WriteHeaderAndEntity(http.StatusOK, v1.NewMachineWaitResponse(a.Machine, s, p, i, ec, token))
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
		logger.Errorw("could not remove machine from wait table", "error", err)
	}()

	go func() {
		changedMachine, err := r.ds.WaitForMachineAllocation(m)
		if err != nil {
			logger.Errorw("stop waiting for changes", "id", id)
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
			Liveliness: metal.MachineLivelinessAlive,
			Tags:       requestPayload.Tags,
			IPMI:       v1.NewMetalIPMI(&requestPayload.IPMI),
		}

		err = r.ds.CreateMachine(m)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}

	} else {
		// machine has already registered, update it
		old := *m

		m.SizeID = size.ID
		m.PartitionID = partition.ID
		m.RackID = requestPayload.RackID
		m.Hardware = machineHardware
		m.Liveliness = metal.MachineLivelinessAlive
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
		err = r.ds.CreateProvisioningEventContainer(&metal.ProvisioningEventContainer{ID: m.ID})
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
	}

	err = r.ds.UpdateSwitchConnections(m)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, makeMachineResponse(m, r.ds, utils.Logger(request).Sugar()))
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
		NetworkIDs:  []string{},
		IPs:         []string{},
		HA:          false,
	}

	m, err := allocateMachine(r.ds, r.ipamer, &spec)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, makeMachineResponse(m, r.ds, utils.Logger(request).Sugar()))
}

func allocateMachine(ds *datastore.RethinkStore, ipamer ipam.IPAMer, allocationSpec *machineAllocationSpec) (*metal.Machine, error) {
	// FIXME: This is only temporary and needs to be made a little bit more elegant.
	if allocationSpec.Tenant == "" {
		return nil, fmt.Errorf("no tenant given")
	}

	var size *metal.Size
	size, err := ds.FindSize(allocationSpec.SizeID)
	if err != nil {
		return nil, fmt.Errorf("size cannot be found: %v", err)
	}

	var partition *metal.Partition
	partition, err = ds.FindPartition(allocationSpec.PartitionID)
	if err != nil {
		return nil, fmt.Errorf("partition cannot be found: %v", err)
	}

	var machine *metal.Machine
	if allocationSpec.UUID == "" {
		// requesting allocation of an arbitrary machine in partition with given size
		machine, err = ds.FindAvailableMachine(partition.ID, size.ID)
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
		if machine.PartitionID != partition.ID {
			return nil, fmt.Errorf("machine is not in the given partition %s", partition.ID)
		}
	}

	if machine == nil {
		return nil, fmt.Errorf("machine is nil")
	}

	old := *machine

	var vrf *metal.Vrf
	vrf, err = ds.FindVrf(map[string]interface{}{"tenant": allocationSpec.Tenant, "projectid": allocationSpec.ProjectID})
	if err != nil {
		return nil, fmt.Errorf("cannot find vrf for tenant project: %v", err)
	}
	if vrf == nil {
		vrf, err = ds.ReserveNewVrf(allocationSpec.Tenant, allocationSpec.ProjectID)
		if err != nil {
			return nil, fmt.Errorf("cannot reserve new vrf for tenant project: %v", err)
		}
	}

	projectNetwork, err := ds.SearchProjectNetwork(allocationSpec.ProjectID)
	if err != nil {
		return nil, err
	}
	primaryNetwork, err := ds.FindPrimaryNetwork(partition.ID)
	if err != nil {
		return nil, fmt.Errorf("could not get primary network: %v", err)
	}
	if projectNetwork == nil {

		projectPrefix, err := createChildPrefix(primaryNetwork.Prefixes, partition.ProjectNetworkPrefixLength, ipamer)
		if err != nil {
			return nil, err
		}
		if projectPrefix == nil {
			return nil, fmt.Errorf("could not allocate child prefix in network: %s", primaryNetwork.ID)
		}

		projectNetwork = &metal.Network{
			Base: metal.Base{
				Name:        fmt.Sprintf("Child of %s", primaryNetwork.ID),
				Description: "Automatically Created Project Network",
			},
			Prefixes:            metal.Prefixes{*projectPrefix},
			DestinationPrefixes: metal.Prefixes{},
			PartitionID:         partition.ID,
			ProjectID:           allocationSpec.ProjectID,
			Nat:                 primaryNetwork.Nat,
			Primary:             false,
			Underlay:            false,
			Vrf:                 vrf.ID,
			ParentNetworkID:     primaryNetwork.ID,
		}

		err = ds.CreateNetwork(projectNetwork)
		if err != nil {
			return nil, err
		}
	}

	// TODO allocateIP should only return a String
	ip, err := allocateIP(*projectNetwork, ipamer)
	if err != nil {
		return nil, fmt.Errorf("unable to allocate an ip in network:%s %#v", projectNetwork.ID, err)
	}

	ip.Name = allocationSpec.Name
	ip.Description = "autoassigned"
	ip.MachineID = machine.ID
	ip.ProjectID = allocationSpec.ProjectID

	err = ds.CreateIP(ip)
	if err != nil {
		return nil, err
	}
	asn, err := ip.ASN()
	if err != nil {
		return nil, err
	}

	machineNetworks := []metal.MachineNetwork{
		{
			NetworkID:           projectNetwork.ID,
			Prefixes:            projectNetwork.Prefixes.String(),
			DestinationPrefixes: projectNetwork.DestinationPrefixes.String(),
			IPs:                 []string{ip.IPAddress},
			Vrf:                 vrf.ID,
			ASN:                 asn,
			Primary:             true,
			Underlay:            projectNetwork.Underlay,
			Nat:                 projectNetwork.Nat,
		},
	}

	for _, additionalNetworkID := range allocationSpec.NetworkIDs {

		if additionalNetworkID == primaryNetwork.ID {
			// We ignore if by accident this allocation contains a network which is a tenant super network
			continue
		}

		nw, err := ds.FindNetwork(additionalNetworkID)
		if err != nil {
			return nil, fmt.Errorf("no network with networkid:%s found %#v", additionalNetworkID, err)
		}
		// additionalNetwork is a tenant network continue
		if nw.ParentNetworkID == primaryNetwork.ID {
			return nil, fmt.Errorf("additional network:%s cannot be a child of the primary network:%s", additionalNetworkID, primaryNetwork.ID)
		}

		// TODO allocateIP should only return a String
		ip, err := allocateIP(*nw, ipamer)
		if err != nil {
			return nil, fmt.Errorf("unable to allocate an ip in network: %s %#v", nw.ID, err)
		}

		ip.Name = allocationSpec.Name
		ip.Description = machine.ID
		ip.ProjectID = allocationSpec.ProjectID

		err = ds.CreateIP(ip)
		if err != nil {
			return nil, err
		}

		additionalMachineNetwork := metal.MachineNetwork{
			NetworkID:           nw.ID,
			Prefixes:            nw.Prefixes.String(),
			IPs:                 []string{ip.IPAddress},
			DestinationPrefixes: nw.DestinationPrefixes.String(),
			ASN:                 asn,
			Primary:             false,
			Underlay:            nw.Underlay,
			Nat:                 projectNetwork.Nat,
			Vrf:                 nw.Vrf,
		}
		machineNetworks = append(machineNetworks, additionalMachineNetwork)
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
	machine.Allocation = alloc

	tagSet := make(map[string]bool)
	tagList := append(machine.Tags, allocationSpec.Tags...)
	for _, t := range tagList {
		tagSet[t] = true
	}
	newTags := []string{}
	for k := range tagSet {
		newTags = append(newTags, k)
	}
	machine.Tags = newTags

	err = ds.UpdateMachine(&old, machine)
	if err != nil {
		ds.DeleteIP(ip)
		return nil, fmt.Errorf("error when allocating machine %q, %v", machine.ID, err)
	}

	err = ds.UpdateWaitingMachine(machine)
	if err != nil {
		ds.DeleteIP(ip)
		ds.UpdateMachine(machine, &old)
		return nil, fmt.Errorf("cannot allocate machine in DB: %v", err)
	}
	return machine, nil
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
	for _, mn := range m.Allocation.MachineNetworks {
		if mn.Primary {
			vrf := fmt.Sprintf("vrf%d", mn.Vrf)
			sws, err = r.ds.SetVrfAtSwitch(m, vrf)
			if err != nil {
				if m.Allocation == nil {
					if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("the machine %q could not be enslaved into the vrf vrf%d", id, mn.Vrf)) {
						return
					}
				}
			}
		} else {
			// FIXME implement other switch port configuration creation for firewall
			// additional switches must be appended to sws if any.
		}
	}

	// Push out events to signal switch configuration change
	evt := metal.SwitchEvent{Type: metal.UPDATE, Machine: *m, Switches: sws}
	err = r.Publish(string(metal.TopicSwitch), evt)
	utils.Logger(request).Sugar().Infow("published switch update event", "event", evt, "error", err)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, makeMachineResponse(m, r.ds, utils.Logger(request).Sugar()))
}

func (r machineResource) freeMachine(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	m, err := r.ds.FindMachine(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	if m.Allocation != nil {
		// if the machine is allocated, we free it in our database
		err = r.releaseMachineNetworks(m, m.Allocation.MachineNetworks)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}

		// TODO: In the future, it would be nice to have the VRF deleted from the vrftable as well

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

	sw, err := r.ds.SetVrfAtSwitch(m, "")
	utils.Logger(request).Sugar().Infow("set VRF at switch", "machineID", id, "error", err)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	deleteEvent := metal.MachineEvent{Type: metal.DELETE, Old: m}
	err = r.Publish(string(metal.TopicMachine), deleteEvent)
	utils.Logger(request).Sugar().Infow("published machine delete event", "machineID", id, "error", err)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	switchEvent := metal.SwitchEvent{Type: metal.UPDATE, Machine: *m, Switches: sw}
	err = r.Publish(string(metal.TopicSwitch), switchEvent)
	utils.Logger(request).Sugar().Infow("published switch update event", "machineID", id, "error", err)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, makeMachineResponse(m, r.ds, utils.Logger(request).Sugar()))
}

func (r machineResource) releaseMachineNetworks(machine *metal.Machine, machineNetworks []metal.MachineNetwork) error {
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
				err = r.ipamer.DeletePrefix(prefix)
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

func (r machineResource) getProvisioningEventContainer(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	// check for existence of the machine
	_, err := r.ds.FindMachine(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	eventContainer, err := r.ds.FindProvisioningEventContainer(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, v1.NewMachineRecentProvisioningEvents(eventContainer))
}

// FIXME: Move to event endpoint
// func (dr machineResource) phoneHome(request *restful.Request, response *restful.Response) {
// 	op := utils.CurrentFuncName()
// 	var data metal.PhoneHomeRequest
// 	err := request.ReadEntity(&data)
// 	log := utils.Logger(request)
// 	if err != nil {
// 		sendError(log, response, op, httperrors.UnprocessableEntity(fmt.Errorf("Cannot read data from request: %v", err)))
// 		return
// 	}
// 	c, err := jwt.FromJWT(data.PhoneHomeToken)
// 	if err != nil {
// 		sendError(log, response, op, httperrors.UnprocessableEntity(fmt.Errorf("Token is invalid: %v", err)))
// 		return
// 	}
// 	if c.Machine == nil || c.Machine.ID == "" {
// 		sendError(log, response, op, httperrors.UnprocessableEntity(fmt.Errorf("Token contains malformed data")))
// 		return
// 	}
// 	oldMachine, err := dr.ds.FindMachine(c.Machine.ID)
// 	if err != nil {
// 		sendError(log, response, op, httperrors.NotFound(err))
// 		return
// 	}
// 	if oldMachine.Allocation == nil {
// 		log.Sugar().Errorw("unallocated machines sends phoneHome", "machine", *oldMachine)
// 		sendError(log, response, op, httperrors.InternalServerError(fmt.Errorf("this machine is not allocated")))
// 	}
// 	newMachine := *oldMachine
// 	lastPingTime := time.Now()
// 	newMachine.Allocation.LastPing = lastPingTime
// 	newMachine.Liveliness = metal.MachineLivelinessAlive // phone home token is sent consistently, but if customer turns off the service, it could turn to unknown
// 	err = dr.ds.UpdateMachine(oldMachine, &newMachine)
// 	if checkError(request, response, op, err) {
// 		return
// 	}
// 	response.WriteEntity(nil)
// }

func (r machineResource) addProvisioningEvent(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	m, err := r.ds.FindMachine(id)
	if err != nil && !metal.IsNotFound(err) {
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
	}

	if m == nil {
		m = &metal.Machine{
			Base: metal.Base{
				ID: id,
			},
			Liveliness: metal.MachineLivelinessAlive,
		}
		err = r.ds.CreateMachine(m)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
	} else {
		old := *m

		m.Liveliness = metal.MachineLivelinessAlive
		err = r.ds.UpdateMachine(&old, m)
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
			ID: m.ID,
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
	} else if event.Event == metal.ProvisioningEventPhonedHome && len(ec.Events) > 0 && ec.Events[0].Event == metal.ProvisioningEventPhonedHome {
		utils.Logger(request).Sugar().Debugw("swallowing repeated phone home event", "id", ec.ID)
	} else {
		ec.Events = append([]metal.ProvisioningEvent{event}, ec.Events...)
		ec.IncompleteProvisioningCycles = ec.CalculateIncompleteCycles(utils.Logger(request).Sugar())
	}

	err = r.ds.UpsertProvisioningEventContainer(ec)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, v1.NewMachineRecentProvisioningEvents(ec))
}

// EvaluateMachineLiveliness evaluates the liveliness of a given machine
func (r machineResource) evaluateMachineLiveliness(m metal.Machine) *metal.Machine {
	if m.Allocation != nil && !m.Allocation.LastPing.IsZero() {
		if time.Since(m.Allocation.LastPing) > metal.MachineDeadAfter {
			// the machine is either dead or the customer did turn off the phone home service
			m.Liveliness = metal.MachineLivelinessUnknown
		} else {
			m.Liveliness = metal.MachineLivelinessAlive
		}
		return &m
	}

	provisioningEvents, err := r.ds.FindProvisioningEventContainer(m.ID)
	if err != nil {
		// we have no provisioning events... we cannot tell
		m.Liveliness = metal.MachineLivelinessUnknown

		return &m
	}

	if provisioningEvents.LastEventTime != nil {
		if time.Since(*provisioningEvents.LastEventTime) > metal.MachineDeadAfter {
			m.Liveliness = metal.MachineLivelinessDead
		} else {
			m.Liveliness = metal.MachineLivelinessAlive
		}
		return &m
	}

	// we have no provisioning events... we cannot tell
	m.Liveliness = metal.MachineLivelinessUnknown

	return &m
}

func (r machineResource) checkMachineLiveliness(request *restful.Request, response *restful.Response) {
	utils.Logger(request).Sugar().Info("liveliness report was requested")

	machines, err := r.ds.ListMachines()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	unknown := 0
	alive := 0
	dead := 0
	for _, m := range machines {
		evaluatedMachine := r.evaluateMachineLiveliness(m)
		err = r.ds.UpdateMachine(&m, evaluatedMachine)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
		switch evaluatedMachine.Liveliness {
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

	err = r.Publish(string(metal.TopicMachine), evt)
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
		s, err = ds.FindSize(m.SizeID)
		if err != nil {
			logger.Errorw("machine with id %s references size with id %s, but size cannot be found in database", m.ID, m.SizeID)
		}
	}

	var p *metal.Partition
	if m.PartitionID != "" {
		p, err = ds.FindPartition(m.PartitionID)
		if err != nil {
			logger.Errorw("machine with id %s references partition with id %s, but partition cannot be found in database", m.ID, m.PartitionID)
		}
	}

	var i *metal.Image
	if m.Allocation != nil {
		if m.Allocation.ImageID != "" {
			i, err = ds.FindImage(m.Allocation.ImageID)
			if err != nil {
				logger.Errorw("machine with id %s references image with id %s, but image cannot be found in database", m.ID, m.Allocation.ImageID)
			}
		}
	}

	var ec *metal.ProvisioningEventContainer
	try, err := ds.FindProvisioningEventContainer(m.ID)
	if err != nil {
		logger.Errorw("machine with id %s has no provisioning event container in the database", m.ID)
	} else {
		ec = try
	}

	return s, p, i, ec
}

func getMachineReferencedEntityMaps(ds *datastore.RethinkStore, logger *zap.SugaredLogger) (metal.SizeMap, metal.PartitionMap, metal.ImageMap, metal.ProvisioningEventContainerMap) {
	s, err := ds.ListSizes()
	if err != nil {
		logger.Errorw("sizes could not be listed: %v", err)
	}

	p, err := ds.ListPartitions()
	if err != nil {
		logger.Errorw("partitions could not be listed: %v", err)
	}

	i, err := ds.ListImages()
	if err != nil {
		logger.Errorw("images could not be listed: %v", err)
	}

	ec, err := ds.ListProvisioningEventContainers()
	if err != nil {
		logger.Errorw("provisioning event containers could not be listed: %v", err)
	}

	return s.ByID(), p.ByID(), i.ByID(), ec.ByID()
}
