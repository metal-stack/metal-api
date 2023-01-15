package service

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/avast/retry-go/v4"
	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
	"go.uber.org/zap"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-lib/httperrors"
)

type switchResource struct {
	webResource
}

// NewSwitch returns a webservice for switch specific endpoints.
func NewSwitch(log *zap.SugaredLogger, ds *datastore.RethinkStore) *restful.WebService {
	r := switchResource{
		webResource: webResource{
			log: log,
			ds:  ds,
		},
	}
	return r.webService()
}

func (r *switchResource) webService() *restful.WebService {
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

func (r *switchResource) findSwitch(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	s, err := r.ds.FindSwitch(id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	resp, err := makeSwitchResponse(s, r.ds)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, resp)
}

func (r *switchResource) listSwitches(request *restful.Request, response *restful.Response) {
	ss, err := r.ds.ListSwitches()
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	resp, err := makeSwitchResponseList(ss, r.ds)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, resp)
}

func (r *switchResource) deleteSwitch(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	s, err := r.ds.FindSwitch(id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	err = r.ds.DeleteSwitch(s)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	resp, err := makeSwitchResponse(s, r.ds)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, resp)
}

// notifySwitch is called periodically from every switch to report last duration and error if ocurred
func (r *switchResource) notifySwitch(request *restful.Request, response *restful.Response) {
	var requestPayload v1.SwitchNotifyRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	id := request.PathParameter("id")
	s, err := r.ds.FindSwitch(id)
	if err != nil && !metal.IsNotFound(err) {
		r.sendError(request, response, defaultError(err))
		return
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
	err = r.ds.UpdateSwitch(&old, s)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	resp, err := makeSwitchResponse(s, r.ds)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, resp)
}

func (r *switchResource) updateSwitch(request *restful.Request, response *restful.Response) {
	var requestPayload v1.SwitchUpdateRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	oldSwitch, err := r.ds.FindSwitch(requestPayload.ID)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	newSwitch := *oldSwitch

	if requestPayload.Description != nil {
		newSwitch.Description = *requestPayload.Description
	}
	newSwitch.ConsoleCommand = requestPayload.ConsoleCommand

	newSwitch.Mode = metal.SwitchModeFrom(requestPayload.Mode)

	err = retry.Do(
		func() error {
			err := r.ds.UpdateSwitch(oldSwitch, &newSwitch)
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
		r.sendError(request, response, defaultError(err))
		return
	}

	resp, err := makeSwitchResponse(&newSwitch, r.ds)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, resp)
}

func (r *switchResource) registerSwitch(request *restful.Request, response *restful.Response) {
	var requestPayload v1.SwitchRegisterRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	if requestPayload.ID == "" {
		r.sendError(request, response, httperrors.BadRequest(errors.New("uuid cannot be empty")))
		return
	}

	_, err = r.ds.FindPartition(requestPayload.PartitionID)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	s, err := r.ds.FindSwitch(requestPayload.ID)
	if err != nil && !metal.IsNotFound(err) {
		r.sendError(request, response, defaultError(err))
		return
	}

	returnCode := http.StatusOK
	if s == nil {
		s = v1.NewSwitch(requestPayload)
		if len(requestPayload.Nics) != len(s.Nics.ByIdentifier()) {
			r.sendError(request, response, httperrors.BadRequest(errors.New("duplicate identifier found in nics")))
			return
		}

		err = r.ds.CreateSwitch(s)
		if err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}

		returnCode = http.StatusCreated
	} else if s.Mode == metal.SwitchReplace {
		spec := v1.NewSwitch(requestPayload)
		err = r.replaceSwitch(s, spec)
		if err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}

		s = spec
	} else {
		old := *s
		spec := v1.NewSwitch(requestPayload)

		uniqueNewNics := spec.Nics.ByIdentifier()
		if len(requestPayload.Nics) != len(uniqueNewNics) {
			r.sendError(request, response, httperrors.BadRequest(errors.New("duplicate identifier found in nics")))
			return
		}

		nics, err := updateSwitchNics(old.Nics.ByIdentifier(), uniqueNewNics, old.MachineConnections)
		if err != nil {
			r.sendError(request, response, defaultError(err))
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
		if spec.OS != nil {
			if s.OS == nil {
				s.OS = &metal.SwitchOS{}
			}
			s.OS.Vendor = spec.OS.Vendor
			s.OS.Version = spec.OS.Version
		}
		s.ManagementIP = spec.ManagementIP
		s.ManagementUser = spec.ManagementUser

		s.Nics = nics
		// Do not replace connections here: We do not want to loose them!

		err = retry.Do(
			func() error {
				err := r.ds.UpdateSwitch(&old, s)
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
			r.sendError(request, response, defaultError(err))
			return
		}

	}

	resp, err := makeSwitchResponse(s, r.ds)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, returnCode, resp)
}

// replaceSwitch replaces a broken switch
//
// assumptions:
//   - cabling btw. old, new and twin-brother switch in the same rack is "the same" (m1 is connected to swp1s0 at every switch)
//
// constraints:
//   - old, new and twin-brother switch must have the same partition and rack
//   - old switch needs to be marked for replacement
//   - twin-brother switch must not be marked for replacement (otherwise there would be no valid source of truth)
//   - new switch needs all the nics of the twin-brother switch
//   - new switch gets the same vrf configuration as the twin-brother switch based on the switch port name
//   - new switch gets the same machine connections as the twin-brother switch based on the switch port name
func (r *switchResource) replaceSwitch(old, new *metal.Switch) error {
	twin, err := r.findTwinSwitch(new)
	if err != nil {
		return fmt.Errorf("could not determine twin brother for switch %s, err: %w", new.Name, err)
	}

	s, err := adoptFromTwin(old, twin, new)
	if err != nil {
		return err
	}

	return r.ds.UpdateSwitch(old, s)
}

// findTwinSwitch finds the neighboring twin of a switch for the given partition and rack
func (r *switchResource) findTwinSwitch(newSwitch *metal.Switch) (*metal.Switch, error) {
	rackSwitches, err := r.ds.SearchSwitches(newSwitch.RackID, nil)
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
		if sw.Mode == metal.SwitchReplace || sw.ID == newSwitch.ID {
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
				newCon.Nic.Identifier = n.Identifier
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

func updateSwitchNics(oldNics, newNics map[string]*metal.Nic, currentConnections metal.ConnectionMap) (metal.Nics, error) {
	// To start off we just prevent basic things that can go wrong
	nicsThatGetLost := metal.Nics{}
	for k, nic := range oldNics {
		_, ok := newNics[k]
		if !ok {
			nicsThatGetLost = append(nicsThatGetLost, *nic)
		}
	}

	// check if nic gets removed but has a connection
	for _, nicThatGetsLost := range nicsThatGetLost {
		for machineID, connections := range currentConnections {
			for _, c := range connections {
				if nicThatGetsLost.Identifier != "" && (c.Nic.Identifier == nicThatGetsLost.Identifier) {
					return nil, fmt.Errorf("nic with identifier %s gets removed but the machine with id %q is already connected to this nic, which is currently not supported", nicThatGetsLost.Identifier, machineID)
				} else if c.Nic.MacAddress == nicThatGetsLost.MacAddress {
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
					if c.Nic.MacAddress == nic.MacAddress && c.Nic.Identifier == nic.Identifier && oldNic.Name != nic.Name {
						return nil, fmt.Errorf("nic with mac address %s and identifier %s wants to be renamed from %q to %q, but already has a connection to machine with id %q, which is currently not supported", nic.MacAddress, nic.Identifier, oldNic.Name, nic.Name, machineID)
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

func makeSwitchResponse(s *metal.Switch, ds *datastore.RethinkStore) (*v1.SwitchResponse, error) {
	p, ips, machines, err := findSwitchReferencedEntites(s, ds)
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
			Identifier: n.Identifier,
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
				Identifier: mc.Nic.Identifier,
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

func findSwitchReferencedEntites(s *metal.Switch, ds *datastore.RethinkStore) (*metal.Partition, metal.IPsMap, metal.Machines, error) {
	var err error
	var p *metal.Partition
	var m metal.Machines

	if s.PartitionID != "" {
		p, err = ds.FindPartition(s.PartitionID)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("switch %q references partition, but partition %q cannot be found in database: %w", s.ID, s.PartitionID, err)
		}

		err = ds.SearchMachines(&datastore.MachineSearchQuery{PartitionID: &s.PartitionID}, &m)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("could not search machines of partition %q for switch %q: %w", s.PartitionID, s.ID, err)
		}
	}

	ips, err := ds.ListIPs()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("ips could not be listed: %w", err)
	}

	return p, ips.ByProjectID(), m, nil
}

func makeSwitchResponseList(ss []metal.Switch, ds *datastore.RethinkStore) ([]*v1.SwitchResponse, error) {
	pMap, ips, err := getSwitchReferencedEntityMaps(ds)
	if err != nil {
		return nil, err
	}

	result := []*v1.SwitchResponse{}
	m, err := ds.ListMachines()
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

func getSwitchReferencedEntityMaps(ds *datastore.RethinkStore) (metal.PartitionMap, metal.IPsMap, error) {
	p, err := ds.ListPartitions()
	if err != nil {
		return nil, nil, fmt.Errorf("partitions could not be listed: %w", err)
	}

	ips, err := ds.ListIPs()
	if err != nil {
		return nil, nil, fmt.Errorf("ips could not be listed: %w", err)
	}

	return p.ByID(), ips.ByProjectID(), nil
}
