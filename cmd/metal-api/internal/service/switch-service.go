package service

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
	"go4.org/netipx"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-lib/auditing"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/metal-lib/pkg/pointer"
)

type switchResource struct {
	webResource
}

// NewSwitch returns a webservice for switch specific endpoints.
func NewSwitch(log *slog.Logger, ds *datastore.RethinkStore) *restful.WebService {
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

	ws.Route(ws.POST("/find").
		To(viewer(r.findSwitches)).
		Operation("findSwitches").
		Doc("get all switches that match given properties").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Metadata(auditing.Exclude, true).
		Reads(v1.SwitchFindRequest{}).
		Writes([]v1.SwitchResponse{}).
		Returns(http.StatusOK, "OK", []v1.SwitchResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.DELETE("/{id}").
		To(editor(r.deleteSwitch)).
		Operation("deleteSwitch").
		Doc("deletes an switch and returns the deleted entity").
		Param(ws.PathParameter("id", "identifier of the switch").DataType("string")).
		Param(ws.QueryParameter("force", "if true switch is deleted with no validation").DataType("boolean").DefaultValue("false")).
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

	ws.Route(ws.POST("/{id}/port").
		To(admin(r.toggleSwitchPort)).
		Operation("toggleSwitchPort").
		Param(ws.PathParameter("id", "identifier of the switch").DataType("string")).
		Doc("toggles the port of the switch with a nicname to the given state").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.SwitchPortToggleRequest{}).
		Returns(http.StatusOK, "OK", v1.SwitchResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		Returns(http.StatusBadRequest, "Bad input data", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/{id}/notify").
		To(editor(r.notifySwitch)).
		Doc("notify the metal-api about a configuration change of a switch").
		Operation("notifySwitch").
		Param(ws.PathParameter("id", "identifier of the switch").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Metadata(auditing.Exclude, true).
		Reads(v1.SwitchNotifyRequest{}).
		Returns(http.StatusOK, "OK", v1.SwitchNotifyResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/migrate").
		To(admin(r.migrate)).
		Doc("migrates machine connections from one switch to another").
		Operation("migrateSwitch").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.SwitchMigrateRequest{}).
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

	resp, err := r.makeSwitchResponse(s)
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

	resp, err := r.makeSwitchResponseList(ss)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, resp)
}

func (r *switchResource) findSwitches(request *restful.Request, response *restful.Response) {
	var requestPayload datastore.SwitchSearchQuery
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	var ss metal.Switches
	err = r.ds.SearchSwitches(&requestPayload, &ss)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	resp, err := r.makeSwitchResponseList(ss)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, resp)
}

func (r *switchResource) deleteSwitch(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	forceParam := request.QueryParameter("force")
	if forceParam == "" {
		forceParam = "false"
	}

	force, err := strconv.ParseBool(forceParam)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	s, err := r.ds.FindSwitch(id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	if !force && len(s.MachineConnections) > 0 {
		r.sendError(request, response, httperrors.BadRequest(fmt.Errorf("cannot delete switch %s while it still has machines connected to it", id)))
		return
	}

	err = r.ds.DeleteSwitch(s)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	status, err := r.ds.GetSwitchStatus(id)
	if err == nil {
		err = r.ds.DeleteSwitchStatus(status)
		if err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}
	} else if !metal.IsNotFound(err) {
		r.sendError(request, response, defaultError(err))
		return
	}

	resp, err := r.makeSwitchResponse(s)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, resp)
}

// notifySwitch is called periodically from every switch to report last duration and error if occurred
func (r *switchResource) notifySwitch(request *restful.Request, response *restful.Response) {
	var requestPayload v1.SwitchNotifyRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	id := request.PathParameter("id")

	ss, err := r.ds.GetSwitchStatus(id)
	if err != nil {
		if !metal.IsNotFound(err) {
			r.sendError(request, response, defaultError(err))
			return
		}

		ss = &metal.SwitchStatus{
			Base: metal.Base{ID: id},
		}
	}

	newSS := *ss

	if requestPayload.Error == nil {
		newSS.LastSync = &metal.SwitchSync{
			Time:     time.Now(),
			Duration: requestPayload.Duration,
		}
	} else {
		newSS.LastSyncError = &metal.SwitchSync{
			Time:     time.Now(),
			Duration: requestPayload.Duration,
			Error:    requestPayload.Error,
		}
	}

	err = r.ds.SetSwitchStatus(&newSS)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	oldSwitch, err := r.ds.FindSwitch(id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	newSwitch := *oldSwitch
	switchUpdated := false

	// old versions of metal-core do not send this field, so make sure we do not crash here
	if requestPayload.PortStates != nil {
		for i, nic := range newSwitch.Nics {
			state, has := requestPayload.PortStates[nic.Name]
			if has {
				reported := metal.SwitchPortStatus(state)
				newstate, changed := nic.State.SetState(reported)
				if changed {
					newSwitch.Nics[i].State = &newstate
					switchUpdated = true
				}
			} else {
				// This should NEVER happen; the switch does not know the given NIC.
				// We log this and ignore it, but something is REALLY wrong in this case
				r.log.Error("unknown switch port", "id", id, "nic", nic.Name)
			}
		}
	}

	if requestPayload.BGPPortStates != nil {
		r.log.Debug("bgp port states received", "id", id, "states", requestPayload.BGPPortStates)
		for i, nic := range newSwitch.Nics {
			bpsnew, has := requestPayload.BGPPortStates[nic.Name]
			if !has && nic.BGPPortState == nil {
				continue
			}
			if !has {
				newSwitch.Nics[i].BGPPortState = nil
				switchUpdated = true
				continue
			}
			if nic.BGPPortState != nil &&
				nic.BGPPortState.Neighbor == bpsnew.Neighbor &&
				nic.BGPPortState.PeerGroup == bpsnew.PeerGroup &&
				nic.BGPPortState.VrfName == bpsnew.VrfName &&
				nic.BGPPortState.BgpState == bpsnew.BgpState &&
				nic.BGPPortState.BgpTimerUpEstablished == bpsnew.BgpTimerUpEstablished &&
				nic.BGPPortState.SentPrefixCounter == bpsnew.SentPrefixCounter &&
				nic.BGPPortState.AcceptedPrefixCounter == bpsnew.AcceptedPrefixCounter {
				continue
			}

			r.log.Debug("bgp port state updated", "id", id, "nic", nic.Name, "state", bpsnew)
			newSwitch.Nics[i].BGPPortState = &metal.SwitchBGPPortState{
				Neighbor:              bpsnew.Neighbor,
				PeerGroup:             bpsnew.PeerGroup,
				VrfName:               bpsnew.VrfName,
				BgpState:              bpsnew.BgpState,
				BgpTimerUpEstablished: bpsnew.BgpTimerUpEstablished,
				SentPrefixCounter:     bpsnew.SentPrefixCounter,
				AcceptedPrefixCounter: bpsnew.AcceptedPrefixCounter,
			}
			switchUpdated = true
		}
	}

	if switchUpdated {
		if err := r.ds.UpdateSwitch(oldSwitch, &newSwitch); err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}
	}

	r.send(request, response, http.StatusOK, v1.NewSwitchNotifyResponse(&newSS))
}

// toggleSwitchPort handles a request to toggle the state of a port on a switch. It reads the request body, finds the switch, updates its NIC state if needed, and returns the updated switch on success.
// If the given port is not found or the given status is not concrete, a 400 error is returned. Another requirement is that there must be a machine connected to the port.
func (r *switchResource) toggleSwitchPort(request *restful.Request, response *restful.Response) {
	var requestPayload v1.SwitchPortToggleRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	desired := metal.SwitchPortStatus(requestPayload.Status)

	if !desired.IsConcrete() {
		r.sendError(request, response, httperrors.BadRequest(fmt.Errorf("the status %q must be concrete", requestPayload.Status)))
		return
	}

	id := request.PathParameter("id")
	oldSwitch, err := r.ds.FindSwitch(id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	newSwitch := *oldSwitch
	updated := false
	found := false

	// Updates the state of a NIC on the switch if the requested state change is valid
	//
	// Loops through each NIC on the switch and checks if the name matches the
	// requested NIC. If a match is found, it tries to update the NIC's state to
	// the requested state. If the state change is valid, it sets the new state on
	// the NIC and marks that an update was made.
	for i, nic := range newSwitch.Nics {
		// compare nic-names case-insensitive
		if strings.EqualFold(nic.Name, requestPayload.NicName) {
			found = true
			newstate, changed := nic.State.WantState(desired)
			if changed {
				newSwitch.Nics[i].State = &newstate
				updated = true
			}
			break
		}
	}
	if !found {
		r.sendError(request, response, httperrors.NotFound(fmt.Errorf("the nic %q does not exist in this switch", requestPayload.NicName)))
		return
	}

	// now check if there is something connected at the given nic.
	machineConnection := false

	for _, mcs := range newSwitch.MachineConnections {
		for _, mc := range mcs {
			if strings.EqualFold(mc.Nic.Name, requestPayload.NicName) {
				machineConnection = true
				break
			}
		}
	}
	if !machineConnection {
		r.sendError(request, response, httperrors.BadRequest(fmt.Errorf("switch %q does not have a connected machine at port %q", id, requestPayload.NicName)))
		return
	}

	if updated {
		if err := r.ds.UpdateSwitch(oldSwitch, &newSwitch); err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}
	}

	resp, err := r.makeSwitchResponse(&newSwitch)
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

	resp, err := r.makeSwitchResponse(&newSwitch)
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

		if err := metal.ValidateSwitchOSVendor(s.OS.Vendor); err != nil {
			r.sendError(request, response, defaultError(err))
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

		if err := metal.ValidateSwitchOSVendor(spec.OS.Vendor); err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}

		err = r.replaceSwitch(s, spec)
		if err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}

		s = spec
	} else {
		old := *s
		spec := v1.NewSwitch(requestPayload)

		if err := metal.ValidateSwitchOSVendor(spec.OS.Vendor); err != nil {
			r.sendError(request, response, defaultError(err))
			return
		}

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
			s.OS.MetalCoreVersion = spec.OS.MetalCoreVersion
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

	resp, err := r.makeSwitchResponse(s)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, returnCode, resp)
}

func (r *switchResource) migrate(request *restful.Request, response *restful.Response) {
	var requestPayload v1.SwitchMigrateRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	if requestPayload.OldSwitchID == "" {
		r.sendError(request, response, httperrors.BadRequest(errors.New("old switch id cannot be empty")))
		return
	}

	if requestPayload.NewSwitchID == "" {
		r.sendError(request, response, httperrors.BadRequest(errors.New("new switch id cannot be empty")))
		return
	}

	old, err := r.ds.FindSwitch(requestPayload.OldSwitchID)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	new, err := r.ds.FindSwitch(requestPayload.NewSwitchID)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	if err := metal.ValidateSwitchOSVendor(new.OS.Vendor); err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	if old.RackID != new.RackID {
		r.sendError(request, response, httperrors.BadRequest(fmt.Errorf("new switch must be in the same rack as the old one")))
		return
	}

	if len(new.MachineConnections) > 0 {
		r.sendError(request, response, httperrors.BadRequest(fmt.Errorf("target switch already has machine connections")))
		return
	}

	s, err := adoptConfiguration(old, new)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	err = r.migrateMachineConnections(old, s)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	err = r.ds.UpdateSwitch(new, s)
	if err != nil {
		r.sendError(request, response, defaultError(fmt.Errorf("failed to migrate switch %s to %s but partial changes might have been written to the database. undo partial changes by migrating in the opposite direction. %w", old.ID, new.ID, err)))
		return
	}

	resp, err := r.makeSwitchResponse(s)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, resp)
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

	nicMap, err := s.TranslateNicMap(old.OS.Vendor)
	if err != nil {
		return err
	}
	err = r.adjustMachineConnections(old.MachineConnections, nicMap)
	if err != nil {
		return err
	}

	return r.ds.UpdateSwitch(old, s)
}

// findTwinSwitch finds the neighboring twin of a switch for the given partition and rack
func (r *switchResource) findTwinSwitch(newSwitch *metal.Switch) (*metal.Switch, error) {
	var rackSwitches metal.Switches
	err := r.ds.SearchSwitches(&datastore.SwitchSearchQuery{RackID: pointer.Pointer(newSwitch.RackID)}, &rackSwitches)
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
		if len(sw.MachineConnections) == 0 {
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
		s := *new
		s.Mode = metal.SwitchOperational
		return &s, nil
	}

	return adoptConfiguration(twin, new)
}

func adoptConfiguration(existing, new *metal.Switch) (*metal.Switch, error) {
	s := *new
	newNics, err := adoptNics(existing, new)
	if err != nil {
		return nil, fmt.Errorf("could not adopt existing nic configuration, err: %w", err)
	}

	newMachineConnections, err := adoptMachineConnections(existing, new)
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
	newNicMap, err := newSwitch.TranslateNicMap(twin.OS.Vendor)
	if err != nil {
		return nil, err
	}
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
		newNic := *nic
		// check for configuration at twin
		if twinNic, ok := twinNicsByName[name]; ok {
			newNic.Vrf = twinNic.Vrf
		}
		newNics = append(newNics, newNic)
	}

	sort.SliceStable(newNics, func(i, j int) bool {
		return newNics[i].Name < newNics[j].Name
	})
	return newNics, nil
}

// adoptMachineConnections copies machine connections from twin and maps mac addresses based on the nic name
func adoptMachineConnections(twin, newSwitch *metal.Switch) (metal.ConnectionMap, error) {
	newNicMap, err := newSwitch.TranslateNicMap(twin.OS.Vendor)
	if err != nil {
		return nil, err
	}
	newConnectionMap := metal.ConnectionMap{}
	missingNics := []string{}

	for mid, cons := range twin.MachineConnections {
		newConnections := metal.Connections{}
		for _, con := range cons {
			if n, ok := newNicMap[con.Nic.Name]; ok {
				newCon := con
				newCon.Nic.Name = n.Name
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

// migrateMachineConnections removes all machine connections from the old switch and adds them to the new one. this enables switch deletion after migration is completed.
func (r *switchResource) migrateMachineConnections(old, new *metal.Switch) error {
	nicMap, err := new.TranslateNicMap(old.OS.Vendor)
	if err != nil {
		return err
	}

	err = r.adjustMachineConnections(old.MachineConnections, nicMap)
	if err != nil {
		return err
	}

	return r.removeSwitchMachineConnections(old)
}

func (r *switchResource) removeSwitchMachineConnections(sw *metal.Switch) error {
	new := *sw
	new.MachineConnections = make(metal.ConnectionMap)
	return r.ds.UpdateSwitch(sw, &new)
}

// adjustMachineConnections updates the neighbor entries for all machines connected to the switch
func (r *switchResource) adjustMachineConnections(oldConnections metal.ConnectionMap, nicMap metal.NicMap) error {
	for mid, cons := range oldConnections {
		m, err := r.ds.FindMachineByID(mid)
		if err != nil {
			return err
		}
		newNics, err := adjustMachineNics(m.Hardware.Nics, cons, nicMap)
		if err != nil {
			return err
		}
		newMachine := *m
		newMachine.Hardware.Nics = newNics
		err = r.ds.UpdateMachine(m, &newMachine)
		if err != nil {
			return err
		}
	}
	return nil
}

func adjustMachineNics(nics metal.Nics, connections metal.Connections, nicMap metal.NicMap) (metal.Nics, error) {
	newNics := make(metal.Nics, 0)

	for _, nic := range nics {
		newNic := nic
		newNeighbors := make([]metal.Nic, 0)
		for _, neigh := range nic.Neighbors {
			if nicInConnections(neigh.Name, neigh.MacAddress, connections) {
				n, ok := nicMap[neigh.Name]
				if !ok {
					return nil, fmt.Errorf("unable to find corresponding new neighbor nic")
				}
				newNeighbors = append(newNeighbors, *n)
			} else {
				newNeighbors = append(newNeighbors, neigh)
			}
		}
		newNic.Neighbors = newNeighbors
		newNics = append(newNics, newNic)
	}

	return newNics, nil
}

func nicInConnections(name string, mac metal.MacAddress, connections metal.Connections) bool {
	for _, con := range connections {
		if con.Nic.Name == name && con.Nic.MacAddress == mac {
			return true
		}
	}
	return false
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
				if c.Nic.GetIdentifier() == nicThatGetsLost.GetIdentifier() {
					return nil, fmt.Errorf("nic with identifier %s gets removed but the machine with id %q is already connected to this nic, which is currently not supported", nicThatGetsLost.GetIdentifier(), machineID)
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
					if c.Nic.MacAddress == nic.MacAddress && c.Nic.GetIdentifier() == nic.GetIdentifier() && oldNic.Name != nic.Name {
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

func (r *switchResource) makeSwitchResponse(s *metal.Switch) (*v1.SwitchResponse, error) {
	p, nws, ips, machines, ss, err := r.findSwitchReferencedEntities(s)
	if err != nil {
		return nil, err
	}

	nics, err := r.makeSwitchNics(s, nws, ips, machines)
	if err != nil {
		return nil, err
	}
	cons := r.makeSwitchCons(s)

	return v1.NewSwitchResponse(s, ss, p, nics, cons), nil
}

func (r *switchResource) makeBGPFilterFirewall(m metal.Machine) (v1.BGPFilter, error) {
	vnis := []string{}
	cidrs := []string{}

	for _, net := range m.Allocation.MachineNetworks {
		if net.Underlay {
			for _, ip := range net.IPs {
				ipwithMask, err := ipWithMask(ip)
				if err != nil {
					return v1.BGPFilter{}, err
				}
				cidrs = append(cidrs, ipwithMask)
			}
		} else {
			vnis = append(vnis, fmt.Sprintf("%d", net.Vrf))
			// filter for "project" addresses / cidrs is not possible since EVPN Type-5 routes can not be filtered by prefixes
		}
	}

	return v1.NewBGPFilter(vnis, cidrs), nil
}

func (r *switchResource) makeBGPFilterMachine(m metal.Machine, nws metal.NetworkMap, ips metal.IPsMap) (v1.BGPFilter, error) {
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

		privateNetwork, ok := nws[private.NetworkID]
		if !ok {
			return v1.BGPFilter{}, fmt.Errorf("no private network found for ID:%s", private.NetworkID)
		}
		parentNetwork, ok := nws[privateNetwork.ParentNetworkID]
		if !ok {
			return v1.BGPFilter{}, fmt.Errorf("no parent network found for ID:%s", privateNetwork.ParentNetworkID)
		}
		// Only for private networks, AdditionalAnnouncableCIDRs are applied.
		// they contain usually the pod- and service- cidrs in a kubernetes cluster
		if len(parentNetwork.AdditionalAnnouncableCIDRs) > 0 {
			r.log.Debug("makeBGPFilterMachine", "additional cidrs", parentNetwork.AdditionalAnnouncableCIDRs)
			cidrs = append(cidrs, parentNetwork.AdditionalAnnouncableCIDRs...)
		}
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

		// TODO machine BGPFilter must not contain firewall private network IPs

		// Allow all other ip addresses allocated for the project.
		ipwithMask, err := ipWithMask(i.IPAddress)
		if err != nil {
			return v1.BGPFilter{}, err
		}
		cidrs = append(cidrs, ipwithMask)
	}

	compactedCidrs, err := compactCidrs(cidrs)
	if err != nil {
		return v1.BGPFilter{}, err
	}
	return v1.NewBGPFilter(vnis, compactedCidrs), nil
}

func ipWithMask(ip string) (string, error) {
	parsed, err := netip.ParseAddr(ip)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%d", ip, parsed.BitLen()), nil
}

func compactCidrs(cidrs []string) ([]string, error) {
	// compact all cidrs which are used to be added to the route map
	// to find the smallest sorted set of prefixes which covers all cidrs which need to be added.
	var ipsetBuilder netipx.IPSetBuilder
	for _, cidr := range cidrs {
		parsed, err := netip.ParsePrefix(cidr)
		if err != nil {
			return nil, err
		}
		ipsetBuilder.AddPrefix(parsed)
	}
	set, err := ipsetBuilder.IPSet()
	if err != nil {
		return nil, fmt.Errorf("unable to create ipset:%w", err)
	}
	var compactedCidrs []string
	for _, pfx := range set.Prefixes() {
		compactedCidrs = append(compactedCidrs, pfx.String())
	}
	return compactedCidrs, nil
}

func (r *switchResource) makeBGPFilter(m metal.Machine, vrf string, nws metal.NetworkMap, ips metal.IPsMap) (v1.BGPFilter, error) {
	var (
		filter v1.BGPFilter
		err    error
	)

	if m.IsFirewall() {
		// vrf "default" means: the firewall was successfully allocated and the switch port configured
		// otherwise the port is still not configured yet (pxe-setup) and a BGPFilter would break the install routine
		if vrf == "default" {
			filter, err = r.makeBGPFilterFirewall(m)
		}
	} else {
		filter, err = r.makeBGPFilterMachine(m, nws, ips)
	}

	return filter, err
}

func (r *switchResource) makeSwitchNics(s *metal.Switch, nws metal.NetworkMap, ips metal.IPsMap, machines metal.Machines) (v1.SwitchNics, error) {
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
			f, err := r.makeBGPFilter(*m, n.Vrf, nws, ips)
			if err != nil {
				return nil, err
			}
			filter = &f
		}
		nic := v1.SwitchNic{
			MacAddress:   string(n.MacAddress),
			Name:         n.Name,
			Identifier:   n.Identifier,
			Vrf:          n.Vrf,
			BGPFilter:    filter,
			Actual:       v1.SwitchPortStatusUnknown,
			BGPPortState: n.BGPPortState,
		}
		if n.State != nil {
			if n.State.Desired != nil {
				nic.Actual = v1.SwitchPortStatus(*n.State.Desired)
			} else {
				nic.Actual = v1.SwitchPortStatus(n.State.Actual)
			}
		}
		nics = append(nics, nic)
	}

	sort.Slice(nics, func(i, j int) bool {
		return nics[i].Name < nics[j].Name
	})

	return nics, nil
}

func (r *switchResource) makeSwitchCons(s *metal.Switch) []v1.SwitchConnection {
	cons := []v1.SwitchConnection{}

	nicMap := s.Nics.ByName()

	for _, metalConnections := range s.MachineConnections {
		for _, mc := range metalConnections {
			// The connection state is set to the state of the NIC in the database.
			// This state is not necessarily the actual state of the port on the switch.
			// When the port is toggled, the connection state in the DB is updated after
			// the real switch port changed state.
			// So if a client queries the current switch state, it will see the desired
			// state in the global NIC state, but the actual state of the port in the
			// connection map.
			n := nicMap[mc.Nic.Name]
			state := metal.SwitchPortStatusUnknown
			var bps *metal.SwitchBGPPortState
			if n != nil && n.State != nil {
				state = n.State.Actual
			}
			if n != nil && n.BGPPortState != nil {
				bps = n.BGPPortState
			}

			nic := v1.SwitchNic{
				MacAddress:   string(mc.Nic.MacAddress),
				Name:         mc.Nic.Name,
				Identifier:   mc.Nic.Identifier,
				Vrf:          mc.Nic.Vrf,
				Actual:       v1.SwitchPortStatus(state),
				BGPPortState: bps,
			}
			con := v1.SwitchConnection{
				Nic:       nic,
				MachineID: mc.MachineID,
			}
			cons = append(cons, con)
		}
	}

	sort.SliceStable(cons, func(i, j int) bool {
		return cons[i].MachineID < cons[j].MachineID
	})

	return cons
}

func (r *switchResource) findSwitchReferencedEntities(s *metal.Switch) (*metal.Partition, metal.NetworkMap, metal.IPsMap, metal.Machines, *metal.SwitchStatus, error) {
	var (
		err error
		p   *metal.Partition
		m   metal.Machines
	)

	if s.PartitionID != "" {
		p, err = r.ds.FindPartition(s.PartitionID)
		if err != nil {
			return nil, nil, nil, nil, nil, fmt.Errorf("switch %q references partition, but partition %q cannot be found in database: %w", s.ID, s.PartitionID, err)
		}

		err = r.ds.SearchMachines(&datastore.MachineSearchQuery{PartitionID: &s.PartitionID}, &m)
		if err != nil {
			return nil, nil, nil, nil, nil, fmt.Errorf("could not search machines of partition %q for switch %q: %w", s.PartitionID, s.ID, err)
		}
	}

	ips, err := r.ds.ListIPs()
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("ips could not be listed: %w", err)
	}

	nws, err := r.ds.ListNetworks()
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("networks could not be listed: %w", err)
	}

	ss, err := r.ds.GetSwitchStatus(s.ID)
	if err != nil && !metal.IsNotFound(err) {
		return nil, nil, nil, nil, nil, fmt.Errorf("switchStatus could not be listed: %w", err)
	}

	return p, nws.ByID(), ips.ByProjectID(), m, ss, nil
}

func (r *switchResource) makeSwitchResponseList(ss metal.Switches) ([]*v1.SwitchResponse, error) {
	pMap, nws, ips, err := getSwitchReferencedEntityMaps(r.ds)
	if err != nil {
		return nil, err
	}

	result := []*v1.SwitchResponse{}
	m, err := r.ds.ListMachines()
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

		nics, err := r.makeSwitchNics(&sw, nws, ips, m)
		if err != nil {
			return nil, err
		}
		cons := r.makeSwitchCons(&sw)
		ss, err := r.ds.GetSwitchStatus(sw.ID)
		if err != nil && !metal.IsNotFound(err) {
			return nil, err
		}
		result = append(result, v1.NewSwitchResponse(&sw, ss, p, nics, cons))
	}

	return result, nil
}

func getSwitchReferencedEntityMaps(ds *datastore.RethinkStore) (metal.PartitionMap, metal.NetworkMap, metal.IPsMap, error) {
	p, err := ds.ListPartitions()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("partitions could not be listed: %w", err)
	}

	ips, err := ds.ListIPs()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("ips could not be listed: %w", err)
	}
	nws, err := ds.ListNetworks()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("networks could not be listed: %w", err)
	}
	return p.ByID(), nws.ByID(), ips.ByProjectID(), nil
}
