package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/utils"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/metal-lib/zapup"
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
		Path(BasePath + "v1/switch").
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

	ws.Route(ws.POST("/").
		To(admin(r.updateSwitch)).
		Operation("updateSwitch").
		Doc("updates a switch. if the switch was changed since this one was read, a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.SwitchUpdateRequest{}).
		Returns(http.StatusOK, "OK", v1.SwitchResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/notify").
		To(editor(r.notifySwitch)).
		Doc("notify the metal-api about a configuration change of a switch").
		Operation("notifySwitch").
		Param(ws.PathParameter("id", "identifier of the switch").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.SwitchNotifyRequest{}).
		Returns(http.StatusOK, "OK", v1.SwitchResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r switchResource) findSwitch(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	s, err := r.ds.FindSwitch(request.Request.Context(), id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	resp, err := makeSwitchResponse(request.Request.Context(), s, r.ds)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = response.WriteHeaderAndEntity(http.StatusOK, resp)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r switchResource) listSwitches(request *restful.Request, response *restful.Response) {
	ss, err := r.ds.ListSwitches(request.Request.Context())
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	resp, err := makeSwitchResponseList(request.Request.Context(), ss, r.ds)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = response.WriteHeaderAndEntity(http.StatusOK, resp)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r switchResource) deleteSwitch(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	s, err := r.ds.FindSwitch(request.Request.Context(), id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = r.ds.DeleteSwitch(request.Request.Context(), s)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	resp, err := makeSwitchResponse(request.Request.Context(), s, r.ds)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = response.WriteHeaderAndEntity(http.StatusOK, resp)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

// notifySwitch is called periodically from every switch to report last duration and error if ocurred
func (r switchResource) notifySwitch(request *restful.Request, response *restful.Response) {
	var requestPayload v1.SwitchNotifyRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	id := request.PathParameter("id")
	s, err := r.ds.FindSwitch(request.Request.Context(), id)
	if err != nil && !metal.IsNotFound(err) {
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
	}

	old := *s
	sync := &metal.SwitchSync{
		Time:     time.Now(),
		Duration: requestPayload.Duration,
	}

	if requestPayload.Error == nil {
		s.LastSync = sync
	} else {
		sync.Error = requestPayload.Error
		s.LastSyncError = sync
	}

	// FIXME needs https://github.com/metal-stack/metal-api/issues/263
	err = r.ds.UpdateSwitch(request.Request.Context(), &old, s)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	resp, err := makeSwitchResponse(request.Request.Context(), s, r.ds)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = response.WriteHeaderAndEntity(http.StatusOK, resp)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r switchResource) updateSwitch(request *restful.Request, response *restful.Response) {
	var requestPayload v1.SwitchUpdateRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	oldSwitch, err := r.ds.FindSwitch(request.Request.Context(), requestPayload.ID)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	newSwitch := *oldSwitch

	if requestPayload.Description != nil {
		newSwitch.Description = *requestPayload.Description
	}

	newSwitch.Mode = metal.SwitchModeFrom(requestPayload.Mode)

	err = retry.Do(
		func() error {
			err := r.ds.UpdateSwitch(request.Request.Context(), oldSwitch, &newSwitch)
			return err
		},
		retry.Attempts(10),
		retry.RetryIf(func(err error) bool {
			return strings.Contains(err.Error(), datastore.EntityAlreadyModifiedErrorMessage)
		}),
		retry.DelayType(retry.CombineDelay(retry.BackOffDelay, retry.RandomDelay)),
		retry.LastErrorOnly(true),
	)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	resp, err := makeSwitchResponse(request.Request.Context(), &newSwitch, r.ds)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = response.WriteHeaderAndEntity(http.StatusOK, resp)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r switchResource) registerSwitch(request *restful.Request, response *restful.Response) {
	var requestPayload v1.SwitchRegisterRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	if requestPayload.ID == "" {
		if checkError(request, response, utils.CurrentFuncName(), errors.New("uuid cannot be empty")) {
			return
		}
	}

	_, err = r.ds.FindPartition(request.Request.Context(), requestPayload.PartitionID)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	s, err := r.ds.FindSwitch(request.Request.Context(), requestPayload.ID)
	if err != nil && !metal.IsNotFound(err) {
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
	}

	returnCode := http.StatusOK

	if s == nil {
		s = v1.NewSwitch(requestPayload)

		if len(requestPayload.Nics) != len(s.Nics.ByMac()) {
			if checkError(request, response, utils.CurrentFuncName(), errors.New("duplicate mac addresses found in nics")) {
				return
			}
		}

		err = r.ds.CreateSwitch(request.Request.Context(), s)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}

		returnCode = http.StatusCreated
	} else if s.Mode == metal.SwitchReplace {
		spec := v1.NewSwitch(requestPayload)
		err = r.replaceSwitch(request.Request.Context(), s, spec)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
		s = spec
	} else {
		old := *s
		spec := v1.NewSwitch(requestPayload)
		if len(requestPayload.Nics) != len(spec.Nics.ByMac()) {
			if checkError(request, response, utils.CurrentFuncName(), errors.New("duplicate mac addresses found in nics")) {
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

		err = retry.Do(
			func() error {
				ctx, cancel := context.WithTimeout(context.Background(), datastore.DefaultQueryTimeout)
				defer cancel()

				err := r.ds.UpdateSwitch(ctx, &old, s)
				return err
			},
			retry.Attempts(10),
			retry.RetryIf(func(err error) bool {
				return strings.Contains(err.Error(), datastore.EntityAlreadyModifiedErrorMessage)
			}),
			retry.DelayType(retry.CombineDelay(retry.BackOffDelay, retry.RandomDelay)),
			retry.LastErrorOnly(true),
		)

		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
	}

	resp, err := makeSwitchResponse(request.Request.Context(), s, r.ds)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = response.WriteHeaderAndEntity(returnCode, resp)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

// replaceSwitch replaces a broken switch
//
// assumptions:
//  - cabling btw. old, new and twin-brother switch in the same rack is "the same" (m1 is connected to swp1s0 at every switch)
//
// constraints:
//  - old, new and twin-brother switch must have the same partition and rack
//  - old switch needs to be marked for replacement
//  - twin-brother switch must not be marked for replacement (otherwise there would be no valid source of truth)
//  - new switch needs all the nics of the twin-brother switch
//  - new switch gets the same vrf configuration as the twin-brother switch based on the switch port name
//  - new switch gets the same machine connections as the twin-brother switch based on the switch port name
func (r switchResource) replaceSwitch(ctx context.Context, old, new *metal.Switch) error {
	twin, err := r.findTwinSwitch(ctx, new)
	if err != nil {
		return fmt.Errorf("could not determine twin brother for switch %s, err: %w", new.Name, err)
	}
	s, err := adoptFromTwin(old, twin, new)
	if err != nil {
		return err
	}
	return r.ds.UpdateSwitch(ctx, old, s)
}

// findTwinSwitch finds the neighboring twin of a switch for the given partition and rack
func (r switchResource) findTwinSwitch(ctx context.Context, newSwitch *metal.Switch) (*metal.Switch, error) {
	rackSwitches, err := r.ds.SearchSwitches(ctx, newSwitch.RackID, nil)
	if err != nil {
		return nil, fmt.Errorf("could not search switches in rack: %v", newSwitch.RackID)
	}
	if len(rackSwitches) == 0 {
		return nil, fmt.Errorf("could not find any switch in rack: %v", newSwitch.RackID)
	}
	var twin *metal.Switch
	for i := range rackSwitches {
		sw := rackSwitches[i]
		if sw.PartitionID != newSwitch.PartitionID {
			continue
		}
		if twin == nil {
			twin = &sw
		} else {
			return nil, fmt.Errorf("found multiple twin switches for %v (%v and %v)", newSwitch.ID, twin.ID, sw.ID)
		}
	}
	if twin == nil {
		return nil, fmt.Errorf("no twin brother found for switch %s, partition: %v, rack: %v", newSwitch.ID, newSwitch.PartitionID, newSwitch.RackID)
	}
	return twin, nil
}

// adoptFromTwin adopts the switch configuration found at the neighboring twin switch to a replacement switch.
func adoptFromTwin(old, twin, new *metal.Switch) (*metal.Switch, error) {
	s := *new
	if new.PartitionID != old.PartitionID {
		return nil, fmt.Errorf("old and new switch belong to different partitions, old: %v, new: %v", old.PartitionID, new.PartitionID)
	}
	if new.RackID != old.RackID {
		return nil, fmt.Errorf("old and new switch belong to different racks, old: %v, new: %v", old.RackID, new.RackID)
	}
	if twin.Mode == metal.SwitchReplace {
		return nil, errors.New("twin switch must not be in replace mode")
	}
	if len(twin.MachineConnections) == 0 {
		// twin switch has no machine connections, switch may be used immediately, replace mode is unnecessary
		s.Mode = metal.SwitchOperational
		return &s, nil
	}

	newNics, err := adoptNics(twin, new)
	if err != nil {
		return nil, fmt.Errorf("could not adopt nic configuration from twin, err: %w", err)
	}

	newMachineConnections, err := adoptMachineConnections(twin, new)
	if err != nil {
		return nil, err
	}

	s.MachineConnections = newMachineConnections
	s.Nics = newNics
	s.Mode = metal.SwitchOperational

	return &s, nil
}

// adoptNics checks whether new switch has all nics of its twin switch
// copies vrf configuration and returns the new nics for the replacement switch
func adoptNics(twin, newSwitch *metal.Switch) (metal.Nics, error) {
	newNics := metal.Nics{}
	newNicMap := newSwitch.Nics.ByName()
	missingNics := []string{}
	twinNicsByName := twin.Nics.ByName()
	for name := range twinNicsByName {
		if _, ok := newNicMap[name]; !ok {
			missingNics = append(missingNics, name)
		}
	}
	if len(missingNics) > 0 {
		return nil, fmt.Errorf("new switch misses the nics %v - check the breakout configuration of the switch ports of switch %s", missingNics, newSwitch.Name)
	}

	for name, nic := range newNicMap {
		// check for configuration at twin
		if twinNic, ok := twinNicsByName[name]; ok {
			newNic := *nic
			newNic.Vrf = twinNic.Vrf
			newNics = append(newNics, newNic)
		} else {
			// leave unchanged
			newNics = append(newNics, *nic)
		}
	}

	sort.SliceStable(newNics, func(i, j int) bool {
		return newNics[i].Name < newNics[j].Name
	})
	return newNics, nil
}

// adoptMachineConnections copies machine connections from twin and maps mac addresses based on the nic name
func adoptMachineConnections(twin, newSwitch *metal.Switch) (metal.ConnectionMap, error) {
	newNicMap := newSwitch.Nics.ByName()
	newConnectionMap := metal.ConnectionMap{}
	missingNics := []string{}
	for mid, cons := range twin.MachineConnections {
		newConnections := metal.Connections{}
		for _, con := range cons {
			if n, ok := newNicMap[con.Nic.Name]; ok {
				newCon := con
				newCon.Nic.MacAddress = n.MacAddress
				newConnections = append(newConnections, newCon)
			} else {
				// this should never happen! it means that we have machine connections with nic names that are not present anymore at the twin or the new switch
				missingNics = append(missingNics, con.Nic.Name)
			}
		}
		newConnectionMap[mid] = newConnections
	}
	if len(missingNics) > 0 {
		return nil, fmt.Errorf("twin switch has machine connections with switch ports that are not present at the new switch %v", missingNics)
	}
	return newConnectionMap, nil
}

func updateSwitchNics(oldNics metal.NicMap, newNics metal.NicMap, currentConnections metal.ConnectionMap) (metal.Nics, error) {
	// To start off we just prevent basic things that can go wrong
	nicsThatGetLost := metal.Nics{}
	for mac, nic := range oldNics {
		_, ok := newNics[mac]
		if !ok {
			nicsThatGetLost = append(nicsThatGetLost, *nic)
		}
	}

	// check if nic gets removed but has a connection
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

			// check if connection exists and name changes
			for machineID, connections := range currentConnections {
				for _, c := range connections {
					if c.Nic.MacAddress == nic.MacAddress && oldNic.Name != nic.Name {
						return nil, fmt.Errorf("nic with mac address %s wants to be renamed from %q to %q, but already has a connection to machine with id %q, which is currently not supported", nic.MacAddress, oldNic.Name, nic.Name, machineID)
					}
				}
			}

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
func setVrfAtSwitches(ctx context.Context, ds *datastore.RethinkStore, m *metal.Machine, vrf string) ([]metal.Switch, error) {
	switches, err := ds.SearchSwitchesConnectedToMachine(ctx, m)
	if err != nil {
		return nil, err
	}
	newSwitches := make([]metal.Switch, 0)
	for i := range switches {
		sw := switches[i]
		oldSwitch := sw
		setVrf(&sw, m.ID, vrf)
		err := ds.UpdateSwitch(ctx, &oldSwitch, &sw)
		if err != nil {
			return nil, err
		}
		newSwitches = append(newSwitches, sw)
	}
	return newSwitches, nil
}

func setVrf(s *metal.Switch, mid, vrf string) {
	affectedMacs := map[metal.MacAddress]bool{}
	for _, c := range s.MachineConnections[mid] {
		mac := c.Nic.MacAddress
		affectedMacs[mac] = true
	}

	if len(affectedMacs) == 0 {
		return
	}

	nics := metal.Nics{}
	for mac, old := range s.Nics.ByMac() {
		e := old
		if _, ok := affectedMacs[mac]; ok {
			e.Vrf = vrf
		}
		nics = append(nics, *e)
	}
	s.Nics = nics
}

func connectMachineWithSwitches(ctx context.Context, ds *datastore.RethinkStore, m *metal.Machine) error {
	switches, err := ds.SearchSwitches(ctx, "", nil)
	if err != nil {
		return err
	}
	oldSwitches := []metal.Switch{}
	newSwitches := []metal.Switch{}
	for _, sw := range switches {
		oldSwitch := sw
		if cons := sw.ConnectMachine(m); cons > 0 {
			oldSwitches = append(oldSwitches, oldSwitch)
			newSwitches = append(newSwitches, sw)
		}
	}

	if len(newSwitches) != 2 {
		return fmt.Errorf("machine %v is not connected to exactly two switches, found connections to %d switches", m.ID, len(newSwitches))
	}

	s1 := newSwitches[0]
	s2 := newSwitches[1]
	cons1 := s1.MachineConnections[m.ID]
	cons2 := s2.MachineConnections[m.ID]
	connectionMapError := fmt.Errorf("twin-switches do not have a connection map that is mirrored crosswise for machine %v, switch %v (connections: %v), switch %v (connections: %v)", m.ID, s1.Name, cons1, s2.Name, cons2)
	if len(cons1) != len(cons2) {
		return connectionMapError
	}

	if s1.RackID != s2.RackID {
		return fmt.Errorf("connected switches of a machine must reside in the same rack, rack of switch %s: %s, rack of switch %s: %s, machine: %s", s1.Name, s1.RackID, s2.Name, s2.RackID, m.ID)
	}
	// We detect the rackID of a machine by connections to leaf switches
	m.RackID = s1.RackID

	byNicName, err := s2.MachineConnections.ByNicName()
	if err != nil {
		return err
	}
	for _, con := range s1.MachineConnections[m.ID] {
		if con2, has := byNicName[con.Nic.Name]; has {
			if con.Nic.Name != con2.Nic.Name {
				return connectionMapError
			}
		} else {
			return connectionMapError
		}
	}

	for i := range oldSwitches {
		err = ds.UpdateSwitch(ctx, &oldSwitches[i], &newSwitches[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func makeSwitchResponse(ctx context.Context, s *metal.Switch, ds *datastore.RethinkStore) (*v1.SwitchResponse, error) {
	p, ips, machines, err := findSwitchReferencedEntites(ctx, s, ds)
	if err != nil {
		return nil, err
	}
	nics := makeSwitchNics(s, ips, machines)
	cons := makeSwitchCons(s)
	return v1.NewSwitchResponse(s, p, nics, cons), nil
}

func makeBGPFilterFirewall(m metal.Machine) v1.BGPFilter {
	vnis := []string{}
	cidrs := []string{}

	for _, net := range m.Allocation.MachineNetworks {
		if net.Underlay {
			for _, ip := range net.IPs {
				cidrs = append(cidrs, fmt.Sprintf("%s/32", ip))
			}
		} else {
			vnis = append(vnis, fmt.Sprintf("%d", net.Vrf))
			// filter for "project" addresses / cidrs is not possible since EVPN Type-5 routes can not be filtered by prefixes
		}
	}
	return v1.NewBGPFilter(vnis, cidrs)
}

func makeBGPFilterMachine(m metal.Machine, ips metal.IPsMap) v1.BGPFilter {
	vnis := []string{}
	cidrs := []string{}

	var private *metal.MachineNetwork
	var underlay *metal.MachineNetwork
	for _, net := range m.Allocation.MachineNetworks {
		if net.Private {
			private = net
		} else if net.Underlay {
			underlay = net
		}
	}

	// Allow all prefixes of the private network
	if private != nil {
		cidrs = append(cidrs, private.Prefixes...)
	}
	for _, i := range ips[m.Allocation.Project] {
		// No need to add /32 addresses of the primary network to the whitelist.
		if private != nil && private.ContainsIP(i.IPAddress) {
			continue
		}
		// Do not allow underlay addresses to be announced.
		if underlay != nil && underlay.ContainsIP(i.IPAddress) {
			continue
		}
		// Allow all other ip addresses allocated for the project.
		cidrs = append(cidrs, fmt.Sprintf("%s/32", i.IPAddress))
	}
	return v1.NewBGPFilter(vnis, cidrs)
}

func makeBGPFilter(m metal.Machine, vrf string, ips metal.IPsMap) v1.BGPFilter {
	var filter v1.BGPFilter
	if m.IsFirewall() {
		// vrf "default" means: the firewall was successfully allocated and the switch port configured
		// otherwise the port is still not configured yet (pxe-setup) and a BGPFilter would break the install routine
		if vrf == "default" {
			filter = makeBGPFilterFirewall(m)
		}
	} else {
		filter = makeBGPFilterMachine(m, ips)
	}
	return filter
}

func makeSwitchNics(s *metal.Switch, ips metal.IPsMap, machines metal.Machines) v1.SwitchNics {
	machinesByID := map[string]*metal.Machine{}
	for i, m := range machines {
		machinesByID[m.ID] = &machines[i]
	}
	machinesBySwp := map[string]*metal.Machine{}
	for mid, metalConnections := range s.MachineConnections {
		for _, mc := range metalConnections {
			if mid == mc.MachineID {
				machinesBySwp[mc.Nic.Name] = machinesByID[mid]
				break
			}
		}
	}
	nics := v1.SwitchNics{}
	for _, n := range s.Nics {
		m := machinesBySwp[n.Name]
		var filter *v1.BGPFilter
		if m != nil && m.Allocation != nil {
			f := makeBGPFilter(*m, n.Vrf, ips)
			filter = &f
		}
		nic := v1.SwitchNic{
			MacAddress: string(n.MacAddress),
			Name:       n.Name,
			Vrf:        n.Vrf,
			BGPFilter:  filter,
		}
		nics = append(nics, nic)
	}
	return nics
}

func makeSwitchCons(s *metal.Switch) []v1.SwitchConnection {
	cons := []v1.SwitchConnection{}
	for _, metalConnections := range s.MachineConnections {
		for _, mc := range metalConnections {
			nic := v1.SwitchNic{
				MacAddress: string(mc.Nic.MacAddress),
				Name:       mc.Nic.Name,
				Vrf:        mc.Nic.Vrf,
			}
			con := v1.SwitchConnection{
				Nic:       nic,
				MachineID: mc.MachineID,
			}
			cons = append(cons, con)
		}
	}
	return cons
}

func findSwitchReferencedEntites(ctx context.Context, s *metal.Switch, ds *datastore.RethinkStore) (*metal.Partition, metal.IPsMap, metal.Machines, error) {
	var err error

	var p *metal.Partition
	var m metal.Machines
	if s.PartitionID != "" {
		p, err = ds.FindPartition(ctx, s.PartitionID)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("switch %q references partition, but partition %q cannot be found in database: %w", s.ID, s.PartitionID, err)
		}

		err = ds.SearchMachines(ctx, &datastore.MachineSearchQuery{PartitionID: &s.PartitionID}, &m)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("could not search machines of partition %q for switch %q: %w", s.PartitionID, s.ID, err)
		}
	}

	ips, err := ds.ListIPs(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("ips could not be listed: %w", err)
	}

	return p, ips.ByProjectID(), m, nil
}

func makeSwitchResponseList(ctx context.Context, ss []metal.Switch, ds *datastore.RethinkStore) ([]*v1.SwitchResponse, error) {
	pMap, ips, err := getSwitchReferencedEntityMaps(ctx, ds)
	if err != nil {
		return nil, err
	}

	result := []*v1.SwitchResponse{}
	m, err := ds.ListMachines(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not find machines: %w", err)
	}

	for i := range ss {
		sw := ss[i]
		var p *metal.Partition
		if sw.PartitionID != "" {
			partitionEntity := pMap[sw.PartitionID]
			p = &partitionEntity
		}

		nics := makeSwitchNics(&sw, ips, m)
		cons := makeSwitchCons(&sw)
		result = append(result, v1.NewSwitchResponse(&sw, p, nics, cons))
	}

	return result, nil
}

func getSwitchReferencedEntityMaps(ctx context.Context, ds *datastore.RethinkStore) (metal.PartitionMap, metal.IPsMap, error) {
	p, err := ds.ListPartitions(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("partitions could not be listed: %w", err)
	}

	ips, err := ds.ListIPs(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("ips could not be listed: %w", err)
	}

	return p.ByID(), ips.ByProjectID(), nil
}
