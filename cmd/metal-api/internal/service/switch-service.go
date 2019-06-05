package service

import (
	"fmt"
	"net/http"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	v1 "git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/service/v1"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/utils"
	"git.f-i-ts.de/cloud-native/metallib/httperrors"
	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"go.uber.org/zap"
)

type switchResource struct {
	webResource
}

// NewSwitch returns a webservice for switch specific endpoints.
func NewSwitch(ds *datastore.RethinkStore) *restful.WebService {
	r := switchResource{
		webResource: webResource{
			ds: ds,
		},
	}
	return r.webService()
}

func (r switchResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path("/v1/switch").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"switch"}

	ws.Route(ws.GET("/{id}").
		To(r.findSwitch).
		Operation("findSwitch").
		Doc("get switch by id").
		Param(ws.PathParameter("id", "identifier of the switch").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.SwitchResponse{}).
		Returns(http.StatusOK, "OK", v1.SwitchResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/").
		To(r.listSwitches).
		Operation("listSwitches").
		Doc("get all switches").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]v1.SwitchResponse{}).
		Returns(http.StatusOK, "OK", []v1.SwitchResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.DELETE("/{id}").
		To(editor(r.deleteSwitch)).
		Operation("deleteSwitch").
		Doc("deletes an switch and returns the deleted entity").
		Param(ws.PathParameter("id", "identifier of the switch").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.SwitchResponse{}).
		Returns(http.StatusOK, "OK", v1.SwitchResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/register").
		To(editor(r.registerSwitch)).
		Doc("register a switch").
		Operation("registerSwitch").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.SwitchRegisterRequest{}).
		Returns(http.StatusOK, "OK", v1.SwitchResponse{}).
		Returns(http.StatusCreated, "Created", v1.SwitchResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r switchResource) findSwitch(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	s, err := r.ds.FindSwitch(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, makeSwitchResponse(s, r.ds, utils.Logger(request).Sugar()))
}

func (r switchResource) listSwitches(request *restful.Request, response *restful.Response) {
	ss, err := r.ds.ListSwitches()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, makeSwitchResponseList(ss, r.ds, utils.Logger(request).Sugar()))
}

func (r switchResource) deleteSwitch(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	s, err := r.ds.FindSwitch(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = r.ds.DeleteSwitch(s)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, makeSwitchResponse(s, r.ds, utils.Logger(request).Sugar()))
}

func (r switchResource) registerSwitch(request *restful.Request, response *restful.Response) {
	var requestPayload v1.SwitchRegisterRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	if requestPayload.ID == "" {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("uuid cannot be empty")) {
			return
		}
	}

	_, err = r.ds.FindPartition(requestPayload.PartitionID)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	s, err := r.ds.FindSwitch(requestPayload.ID)
	if err != nil && !metal.IsNotFound(err) {
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
	}

	returnCode := http.StatusOK

	if s == nil {
		s = v1.NewSwitch(requestPayload)

		if len(requestPayload.Nics) != len(s.Nics.ByMac()) {
			if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("duplicate mac addresses found in nics")) {
				return
			}
		}

		err = r.ds.CreateSwitch(s)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}

		// TODO: Broken switch: A machine was registered before this new switch is getting registered
		// It needs to take over the existing connections from the broken switch or something?
		// https://git.f-i-ts.de/cloud-native/metal/metal/issues/28

		returnCode = http.StatusCreated
	} else {
		old := *s

		spec := v1.NewSwitch(requestPayload)

		if len(requestPayload.Nics) != len(spec.Nics.ByMac()) {
			if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("duplicate mac addresses found in nics")) {
				return
			}
		}

		nics, err := updateSwitchNics(old.Nics.ByMac(), spec.Nics.ByMac(), old.MachineConnections)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}

		if requestPayload.Name != nil {
			s.Name = *requestPayload.Name
		}
		if requestPayload.Description != nil {
			s.Description = *requestPayload.Description
		}
		s.RackID = spec.RackID
		s.PartitionID = spec.PartitionID

		s.Nics = nics
		// Do not replace connections here: We do not want to loose them!

		err = r.ds.UpdateSwitch(&old, s)

		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
	}

	response.WriteHeaderAndEntity(returnCode, makeSwitchResponse(s, r.ds, utils.Logger(request).Sugar()))
}

func updateSwitchNics(oldNics metal.NicMap, newNics metal.NicMap, currentConnections metal.ConnectionMap) (metal.Nics, error) {
	// TODO: Broken switch would change nics, but if this happens we would need to repair broken connections:
	// https://git.f-i-ts.de/cloud-native/metal/metal/issues/28

	// To start off we just prevent basic things that can go wrong
	nicsThatGetLost := metal.Nics{}
	for mac, nic := range oldNics {
		_, ok := newNics[mac]
		if !ok {
			nicsThatGetLost = append(nicsThatGetLost, *nic)
		}
	}

	for _, nicThatGetsLost := range nicsThatGetLost {
		for machineID, connections := range currentConnections {
			for _, c := range connections {
				if c.Nic.MacAddress == nicThatGetsLost.MacAddress {
					return nil, fmt.Errorf("nic with mac address %s gets removed but the machine with id %q is already connected to this nic, which is currently not supported", nicThatGetsLost.MacAddress, machineID)
				}
			}
		}
	}

	nicsThatGetAdded := metal.Nics{}
	nicsThatAlreadyExist := metal.Nics{}
	for mac, nic := range newNics {
		oldNic, ok := oldNics[mac]
		if ok {
			updatedNic := *oldNic
			updatedNic.Name = nic.Name
			nicsThatAlreadyExist = append(nicsThatAlreadyExist, updatedNic)
		} else {
			nicsThatGetAdded = append(nicsThatGetAdded, *nic)
		}
	}

	finalNics := metal.Nics{}
	finalNics = append(finalNics, nicsThatGetAdded...)
	finalNics = append(finalNics, nicsThatAlreadyExist...)

	return finalNics, nil

}

// SetVrfAtSwitches finds the switches connected to the given machine and puts the switch ports into the given vrf.
// Returns the updated switches.
func setVrfAtSwitches(ds *datastore.RethinkStore, m *metal.Machine, vrf string) ([]metal.Switch, error) {
	switches, err := ds.SearchSwitchesConnectedToMachine(m)
	if err != nil {
		return nil, err
	}
	newSwitches := make([]metal.Switch, 0)
	for _, sw := range switches {
		oldSwitch := sw
		setVrf(&sw, m.ID, vrf)
		err := ds.UpdateSwitch(&oldSwitch, &sw)
		if err != nil {
			return nil, err
		}
		newSwitches = append(newSwitches, sw)
	}
	return newSwitches, nil
}

func setVrf(s *metal.Switch, mid, vrf string) {
	// gather nics within MachineConnections
	changed := metal.Nics{}
	for _, c := range s.MachineConnections[mid] {
		c.Nic.Vrf = vrf
		changed = append(changed, c.Nic)
	}

	if len(changed) == 0 {
		return
	}

	// update sw.Nics
	currentByMac := s.Nics.ByMac()
	changedByMac := changed.ByMac()
	s.Nics = metal.Nics{}
	for mac, old := range currentByMac {
		e := old
		if new, has := changedByMac[mac]; has {
			e = new
		}
		s.Nics = append(s.Nics, *e)
	}
}

func connectMachineWithSwitches(ds *datastore.RethinkStore, m *metal.Machine) error {
	switches, err := ds.SearchSwitches(m.RackID, nil)
	if err != nil {
		return err
	}
	for _, sw := range switches {
		oldSwitch := sw
		sw.ConnectMachine(m)
		err := ds.UpdateSwitch(&oldSwitch, &sw)
		if err != nil {
			return err
		}
	}
	return nil
}

func makeSwitchResponse(s *metal.Switch, ds *datastore.RethinkStore, logger *zap.SugaredLogger) *v1.SwitchResponse {
	p := findSwitchReferencedEntites(s, ds, logger)
	return v1.NewSwitchResponse(s, p)
}

func findSwitchReferencedEntites(s *metal.Switch, ds *datastore.RethinkStore, logger *zap.SugaredLogger) *metal.Partition {
	var err error

	var p *metal.Partition
	if s.PartitionID != "" {
		p, err = ds.FindPartition(s.PartitionID)
		if err != nil {
			logger.Errorw("switch references partition, but partition cannot be found in database", "switchID", s.ID, "partitionID", s.PartitionID, "error", err)
		}
	}

	return p
}

func makeSwitchResponseList(ss []metal.Switch, ds *datastore.RethinkStore, logger *zap.SugaredLogger) []*v1.SwitchResponse {
	pMap := getSwitchReferencedEntityMaps(ds, logger)

	result := []*v1.SwitchResponse{}

	for index := range ss {
		var p *metal.Partition
		if ss[index].PartitionID != "" {
			partitionEntity := pMap[ss[index].PartitionID]
			p = &partitionEntity
		}
		result = append(result, v1.NewSwitchResponse(&ss[index], p))
	}

	return result
}

func getSwitchReferencedEntityMaps(ds *datastore.RethinkStore, logger *zap.SugaredLogger) metal.PartitionMap {
	p, err := ds.ListPartitions()
	if err != nil {
		logger.Errorw("partitions could not be listed", "error", err)
	}

	return p.ByID()
}
