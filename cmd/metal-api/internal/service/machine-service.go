package service

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/netbox"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/utils"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/utils/jwt"
	"git.f-i-ts.de/cloud-native/metallib/bus"
	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"go.uber.org/zap"
)

const (
	waitForServerTimeout = 30 * time.Second
)

type machineResource struct {
	webResource
	bus.Publisher
	netbox *netbox.APIProxy
}

// NewMachine returns a webservice for machine specific endpoints.
func NewMachine(
	ds *datastore.RethinkStore,
	pub bus.Publisher,
	netbox *netbox.APIProxy) *restful.WebService {
	dr := machineResource{
		webResource: webResource{
			ds: ds,
		},
		Publisher: pub,
		netbox:    netbox,
	}
	return dr.webService()
}

// webService creates the webservice endpoint
func (dr machineResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path("/v1/machine").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"machine"}

	ws.Route(ws.GET("/{id}").
		To(dr.restEntityGet(dr.ds.FindMachine)).
		Operation("findMachine").
		Doc("get machine by id").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(metal.Machine{}).
		Returns(http.StatusOK, "OK", metal.Machine{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	ws.Route(ws.GET("/").
		To(dr.restListGet(dr.ds.ListMachines)).
		Operation("listMachines").
		Doc("get all known machines").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]metal.Machine{}).
		Returns(http.StatusOK, "OK", []metal.Machine{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	ws.Route(ws.GET("/find").To(dr.searchMachine).
		Doc("search machines").
		Param(ws.QueryParameter("mac", "one of the MAC address of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]metal.Machine{}).
		Returns(http.StatusOK, "OK", []metal.Machine{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	ws.Route(ws.POST("/register").To(dr.registerMachine).
		Doc("register a machine").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(metal.RegisterMachine{}).
		Writes(metal.Machine{}).
		Returns(http.StatusOK, "OK", metal.Machine{}).
		Returns(http.StatusCreated, "Created", metal.Machine{}).
		Returns(http.StatusNotFound, "one of the given key values was not found", nil))

	ws.Route(ws.POST("/allocate").To(dr.allocateMachine).
		Doc("allocate a machine").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(metal.AllocateMachine{}).
		Returns(http.StatusOK, "OK", metal.Machine{}).
		Returns(http.StatusNotFound, "No free machine for allocation found", nil).
		Returns(http.StatusUnprocessableEntity, "Unprocessable Entity", metal.ErrorResponse{}))

	ws.Route(ws.POST("/{id}/state").To(dr.setMachineState).
		Doc("set the state of a machine").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(metal.MachineState{}).
		Writes(metal.Machine{}).
		Returns(http.StatusOK, "OK", metal.Machine{}).
		Returns(http.StatusNotFound, "one of the given key values was not found", nil).
		Returns(http.StatusUnprocessableEntity, "Unprocessable Entity", metal.ErrorResponse{}))

	ws.Route(ws.DELETE("/{id}/free").To(dr.freeMachine).
		Doc("free a machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(http.StatusOK, "OK", metal.Machine{}).
		Returns(http.StatusUnprocessableEntity, "Unprocessable Entity", metal.ErrorResponse{}))

	ws.Route(ws.GET("/{id}/ipmi").To(dr.ipmiData).
		Doc("returns the IPMI connection data for a machine").
		Operation("ipmiData").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(http.StatusOK, "OK", metal.IPMI{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	ws.Route(ws.GET("/{id}/wait").To(dr.waitForAllocation).
		Doc("wait for an allocation of this machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(http.StatusOK, "OK", metal.MachineWithPhoneHomeToken{}).
		Returns(http.StatusGatewayTimeout, "Timeout", nil).
		Returns(http.StatusNotFound, "Not Found", nil))

	ws.Route(ws.POST("/{id}/report").To(dr.allocationReport).
		Doc("send the allocation report of a given machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(metal.ReportAllocation{}).
		Returns(http.StatusOK, "OK", metal.MachineAllocation{}).
		Returns(http.StatusNotFound, "Not Found", nil).
		Returns(http.StatusUnprocessableEntity, "Unprocessable Entity", metal.ErrorResponse{}))

	ws.Route(ws.POST("/{id}/on").To(dr.machineOn).
		Doc("sends a power-on to the machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads([]string{}).
		Returns(http.StatusOK, "OK", metal.MachineAllocation{}).
		Returns(http.StatusNotFound, "Not Found", nil).
		Returns(http.StatusUnprocessableEntity, "Unprocessable Entity", metal.ErrorResponse{}))

	ws.Route(ws.POST("/{id}/off").To(dr.machineOff).
		Doc("sends a power-off to the machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads([]string{}).
		Returns(http.StatusOK, "OK", metal.MachineAllocation{}).
		Returns(http.StatusNotFound, "Not Found", nil).
		Returns(http.StatusUnprocessableEntity, "Unprocessable Entity", metal.ErrorResponse{}))

	ws.Route(ws.POST("/{id}/reset").To(dr.machineReset).
		Doc("sends a reset to the machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads([]string{}).
		Returns(http.StatusOK, "OK", metal.MachineAllocation{}).
		Returns(http.StatusNotFound, "Not Found", nil).
		Returns(http.StatusUnprocessableEntity, "Unprocessable Entity", metal.ErrorResponse{}))

	ws.Route(ws.POST("/{id}/bios").To(dr.machineBios).
		Doc("sends a bios to the machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads([]string{}).
		Returns(http.StatusOK, "OK", metal.MachineAllocation{}).
		Returns(http.StatusNotFound, "Not Found", nil).
		Returns(http.StatusUnprocessableEntity, "Unprocessable Entity", metal.ErrorResponse{}))

	ws.Route(ws.POST("/phoneHome").To(dr.phoneHome).
		Doc("phone back home from the machine").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(metal.PhoneHomeRequest{}).
		Returns(http.StatusOK, "OK", nil).
		Returns(http.StatusNotFound, "Machine could not be found by id", nil).
		Returns(http.StatusUnprocessableEntity, "Unprocessable Entity", metal.ErrorResponse{}))

	return ws
}

func (dr machineResource) waitForAllocation(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	ctx := request.Request.Context()
	log := utils.Logger(request)
	err := dr.ds.Wait(id, func(alloc datastore.Allocation) error {
		select {
		case <-time.After(waitForServerTimeout):
			response.WriteErrorString(http.StatusGatewayTimeout, "server timeout")
			return fmt.Errorf("server timeout")
		case a := <-alloc:
			if a.Err != nil {
				log.Sugar().Errorw("allocation returned an error", "error", a.Err)
				return a.Err
			}
			log.Sugar().Infow("return allocated machine", "machine", a)
			ka := jwt.NewPhoneHomeClaims(a.Machine)
			token, err := ka.JWT()
			if err != nil {
				return fmt.Errorf("could not create jwt: %v", err)
			}
			err = response.WriteEntity(metal.MachineWithPhoneHomeToken{Machine: a.Machine, PhoneHomeToken: token})
			if err != nil {
				return fmt.Errorf("could not write entity: %v", err)
			}
		case <-ctx.Done():
			return fmt.Errorf("client timeout")
		}
		return nil
	})
	if err != nil {
		sendError(log, response, "waitForAllocation", http.StatusInternalServerError, err)
	}
}

func (dr machineResource) phoneHome(request *restful.Request, response *restful.Response) {
	var data metal.PhoneHomeRequest
	err := request.ReadEntity(&data)
	log := utils.Logger(request)
	if err != nil {
		sendError(log, response, "phoneHome", http.StatusUnprocessableEntity, fmt.Errorf("Cannot read data from request: %v", err))
		return
	}
	c, err := jwt.FromJWT(data.PhoneHomeToken)
	if err != nil {
		sendError(log, response, "phoneHome", http.StatusUnprocessableEntity, fmt.Errorf("Token is invalid: %v", err))
		return
	}
	if c.Machine == nil || c.Machine.ID == "" {
		sendError(log, response, "phoneHome", http.StatusUnprocessableEntity, fmt.Errorf("Token contains malformed data"))
		return
	}
	oldMachine, err := dr.ds.FindMachine(c.Machine.ID)
	if err != nil {
		sendError(log, response, "phoneHome", http.StatusNotFound, err)
		return
	}
	if oldMachine.Allocation == nil {
		log.Sugar().Errorw("unallocated machines sends phoneHome", "machine", *oldMachine)
		sendError(log, response, "phoneHome", http.StatusInternalServerError, fmt.Errorf("this machine is not allocated"))
	}
	newMachine := *oldMachine
	newMachine.Allocation.LastPing = time.Now()
	err = dr.ds.UpdateMachine(oldMachine, &newMachine)
	if checkError(request, response, "phoneHome", err) {
		return
	}
	response.WriteEntity(nil)
}

func (dr machineResource) searchMachine(request *restful.Request, response *restful.Response) {
	mac := strings.TrimSpace(request.QueryParameter("mac"))

	result, err := dr.ds.SearchMachine(mac)
	if checkError(request, response, "searchMachine", err) {
		return
	}

	response.WriteEntity(result)
}

func (dr machineResource) setMachineState(request *restful.Request, response *restful.Response) {
	log := utils.Logger(request).Sugar()
	var data metal.MachineState
	err := request.ReadEntity(&data)
	if checkError(request, response, "setMachineState", err) {
		return
	}
	if data.Value != metal.AvailableState && data.Description == "" {
		// we want a "WHY" if this machine should not be available
		log.Errorw("empty description in state", "state", data)
		sendError(log.Desugar(), response, "setMachineState", http.StatusUnprocessableEntity, fmt.Errorf("you must supply a description"))
	}
	found := false
	for _, s := range metal.AllStates {
		if data.Value == s {
			found = true
			break
		}
	}
	if !found {
		log.Errorw("illegal state sent", "state", data, "allowed", metal.AllStates)
		sendError(log.Desugar(), response, "setMachineState", http.StatusUnprocessableEntity, fmt.Errorf("the state is illegal"))
	}
	id := request.PathParameter("id")
	m, err := dr.ds.FindMachine(id)
	if checkError(request, response, "setMachineState", err) {
		return
	}
	if m.State.Value == data.Value && m.State.Description == data.Description {
		response.WriteEntity(m)
		return
	}
	newmachine := *m
	newmachine.State = data
	err = dr.ds.UpdateMachine(m, &newmachine)
	if checkError(request, response, "setMachineState", err) {
		return
	}
	response.WriteEntity(newmachine)
}

func (dr machineResource) registerMachine(request *restful.Request, response *restful.Response) {
	var data metal.RegisterMachine
	err := request.ReadEntity(&data)
	log := utils.Logger(request).Sugar()
	if checkError(request, response, "registerMachine", err) {
		return
	}
	if data.UUID == "" {
		sendError(utils.Logger(request), response, "registerMachine", http.StatusUnprocessableEntity, fmt.Errorf("No UUID given"))
		return
	}
	part, err := dr.ds.FindPartition(data.PartitionID)
	if checkError(request, response, "registerMachine", err) {
		return
	}

	size, _, err := dr.ds.FromHardware(data.Hardware)
	if err != nil {
		size = metal.UnknownSize
		log.Errorw("no size found for hardware", "hardware", data.Hardware, "error", err)
	}

	err = dr.netbox.Register(part.ID, data.RackID, size.ID, data.UUID, data.Hardware.Nics)
	if checkError(request, response, "registerMachine", err) {
		return
	}

	m, err := dr.ds.RegisterMachine(data.UUID, *part, data.RackID, *size, data.Hardware, data.IPMI, data.Tags)

	if checkError(request, response, "registerMachine", err) {
		return
	}

	err = dr.ds.UpdateSwitchConnections(m)
	if checkError(request, response, "registerMachine", err) {
		return
	}

	response.WriteEntity(m)
}

func (dr machineResource) ipmiData(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	ipmi, err := dr.ds.FindIPMI(id)

	if checkError(request, response, "ipmiData", err) {
		return
	}
	response.WriteEntity(ipmi)
}

func (dr machineResource) allocateMachine(request *restful.Request, response *restful.Response) {
	var allocate metal.AllocateMachine
	err := request.ReadEntity(&allocate)
	log := utils.Logger(request)
	slog := log.Sugar()
	if checkError(request, response, "allocateMachine", err) {
		return
	}
	if allocate.Tenant == "" {
		if checkError(request, response, "allocateMachine", fmt.Errorf("no tenant given")) {
			slog.Errorw("allocate", zap.String("tenant", "missing"))
			return
		}
	}
	image, err := dr.ds.FindImage(allocate.ImageID)
	if checkError(request, response, "allocateMachine", err) {
		return
	}
	var size *metal.Size
	var part *metal.Partition
	if allocate.UUID == "" {
		size, err = dr.ds.FindSize(allocate.SizeID)
		if checkError(request, response, "allocateMachine", err) {
			return
		}
		part, err = dr.ds.FindPartition(allocate.PartitionID)
		if checkError(request, response, "allocateMachine", err) {
			return
		}
	}

	d, err := dr.ds.AllocateMachine(allocate.UUID, allocate.Name, allocate.Description, allocate.Hostname,
		allocate.ProjectID, part, size,
		image, allocate.SSHPubKeys, allocate.Tags,
		allocate.UserData,
		allocate.Tenant,
		dr.netbox)
	if err != nil {
		if err == datastore.ErrNoMachineAvailable {
			sendError(log, response, "allocateMachine", http.StatusNotFound, err)
		} else {
			sendError(log, response, "allocateMachine", http.StatusUnprocessableEntity, err)
		}
		return
	}
	response.WriteEntity(d)
}

func (dr machineResource) freeMachine(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	m, err := dr.ds.FreeMachine(id)
	if checkError(request, response, "freeMachine", err) {
		return
	}

	sw, err := dr.ds.SetVrfAtSwitch(m, "")
	if checkError(request, response, "freeMachine", err) {
		return
	}

	err = dr.netbox.Release(id)
	if checkError(request, response, "freeMachine", err) {
		return
	}

	evt := metal.MachineEvent{Type: metal.DELETE, Old: m}
	dr.Publish(string(metal.TopicMachine), evt)
	utils.Logger(request).Sugar().Infow("publish delete event", "event", evt)

	se := metal.SwitchEvent{Type: metal.UPDATE, Machine: *m, Switches: sw}
	dr.Publish(string(metal.TopicSwitch), se)
	utils.Logger(request).Sugar().Infow("publish switch update event", "event", se)

	response.WriteEntity(m)
}

func (dr machineResource) machineOn(request *restful.Request, response *restful.Response) {
	dr.machineCmd("machineOn", metal.MachineOnCmd, request, response)
}

func (dr machineResource) machineOff(request *restful.Request, response *restful.Response) {
	dr.machineCmd("machineOff", metal.MachineOffCmd, request, response)
}

func (dr machineResource) machineReset(request *restful.Request, response *restful.Response) {
	dr.machineCmd("machineReset", metal.MachineResetCmd, request, response)
}

func (dr machineResource) machineBios(request *restful.Request, response *restful.Response) {
	dr.machineCmd("machineBios", metal.MachineBiosCmd, request, response)
}

func (dr machineResource) machineCmd(op string, cmd metal.MachineCommand, request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	var params []string
	if err := request.ReadEntity(&params); checkError(request, response, op, err) {
		return
	}
	m, err := dr.ds.FindMachine(id)
	if checkError(request, response, op, err) {
		return
	}
	evt := metal.MachineEvent{Type: metal.COMMAND, Cmd: &metal.MachineExecCommand{
		Command: cmd,
		Params:  params,
		Target:  m,
	}}
	if checkError(request, response, op, dr.Publish("machine", evt)) {
		return
	}
	utils.Logger(request).Sugar().Infow("publish event", "event", evt, "command", *evt.Cmd)
	response.WriteEntity(m)
}

func (dr machineResource) allocationReport(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	var report metal.ReportAllocation
	err := request.ReadEntity(&report)
	if checkError(request, response, "allocationReport", err) {
		return
	}

	m, err := dr.ds.FindMachine(id)
	if checkError(request, response, "allocationReport", err) {
		return
	}
	if !report.Success {
		utils.Logger(request).Sugar().Errorw("failed allocation", "id", id, "error-message", report.ErrorMessage)
		response.WriteEntity(m.Allocation)
		return
	}
	if m.Allocation == nil {
		sendError(utils.Logger(request), response, "allocationReport", http.StatusUnprocessableEntity, fmt.Errorf("the machine %q is not allocated", id))
		return
	}
	old := *m
	m.Allocation.ConsolePassword = report.ConsolePassword
	err = dr.ds.UpdateMachine(&old, m)
	if err != nil {
		sendError(utils.Logger(request), response, "allocationReport", http.StatusUnprocessableEntity, fmt.Errorf("the machine %q could not be updated", id))
		return
	}

	vrf := fmt.Sprintf("vrf%d", m.Allocation.Vrf)
	sw, err := dr.ds.SetVrfAtSwitch(m, vrf)
	if err != nil {
		sendError(utils.Logger(request), response, "allocationReport", http.StatusUnprocessableEntity, fmt.Errorf("the machine %q could not be enslaved into the vrf vrf%d", id, m.Allocation.Vrf))
		return
	}

	// Push out events to signal switch configuration change
	evt := metal.SwitchEvent{Type: metal.UPDATE, Machine: *m, Switches: sw}
	dr.Publish(string(metal.TopicSwitch), evt)
	utils.Logger(request).Sugar().Infow("publish switch update event", "event", evt)
	response.WriteEntity(m.Allocation)
}
