package service

import (
	"fmt"
	"log/slog"
	"net/http"
	"sort"

	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	mdm "github.com/metal-stack/masterdata-api/pkg/client"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-lib/auditing"
	"github.com/metal-stack/metal-lib/pkg/pointer"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-lib/httperrors"
)

type sizeResource struct {
	webResource
	mdc mdm.Client
}

// NewSize returns a webservice for size specific endpoints.
func NewSize(log *slog.Logger, ds *datastore.RethinkStore, mdc mdm.Client) *restful.WebService {
	r := sizeResource{
		webResource: webResource{
			log: log,
			ds:  ds,
		},
		mdc: mdc,
	}
	return r.webService()
}

func (r *sizeResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path(BasePath + "v1/size").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"size"}

	ws.Route(ws.GET("/{id}").
		To(r.findSize).
		Operation("findSize").
		Doc("get size by id").
		Param(ws.PathParameter("id", "identifier of the size").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.SizeResponse{}).
		Returns(http.StatusOK, "OK", v1.SizeResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/").
		To(r.listSizes).
		Operation("listSizes").
		Doc("get all sizes").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]v1.SizeResponse{}).
		Returns(http.StatusOK, "OK", []v1.SizeResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.DELETE("/{id}").
		To(admin(r.deleteSize)).
		Operation("deleteSize").
		Doc("deletes an size and returns the deleted entity").
		Param(ws.PathParameter("id", "identifier of the size").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.SizeResponse{}).
		Returns(http.StatusOK, "OK", v1.SizeResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.PUT("/").
		To(admin(r.createSize)).
		Operation("createSize").
		Doc("create a size. if the given ID already exists a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.SizeCreateRequest{}).
		Returns(http.StatusCreated, "Created", v1.SizeResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/").
		To(admin(r.updateSize)).
		Operation("updateSize").
		Doc("updates a size. if the size was changed since this one was read, a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.SizeUpdateRequest{}).
		Returns(http.StatusOK, "OK", v1.SizeResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	// suggest

	ws.Route(ws.POST("/suggest").
		To(r.suggestSize).
		Operation("suggest").
		Doc("from a given machine id returns the appropriate size").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Metadata(auditing.Exclude, true).
		Reads(v1.SizeSuggestRequest{}).
		Returns(http.StatusOK, "OK", []v1.SizeConstraint{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	// size reservations

	ws.Route(ws.GET("/reservations").
		To(r.listSizeReservations).
		Operation("listSizeReservations").
		Doc("get all size reservations").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Metadata(auditing.Exclude, true).
		Writes([]v1.SizeReservationResponse{}).
		Returns(http.StatusOK, "OK", []v1.SizeReservationResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/reservations/find").
		To(r.findSizeReservations).
		Operation("findSizeReservations").
		Doc("get all size reservations").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Metadata(auditing.Exclude, true).
		Reads(v1.SizeReservationListRequest{}).
		Writes([]v1.SizeReservationResponse{}).
		Returns(http.StatusOK, "OK", []v1.SizeReservationResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/reservations/{id}").
		To(r.getSizeReservation).
		Operation("getSizeReservation").
		Doc("get size reservation by id").
		Param(ws.PathParameter("id", "identifier of the size reservation").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.SizeReservationResponse{}).
		Returns(http.StatusOK, "OK", v1.SizeReservationResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.DELETE("/reservations/{id}").
		To(admin(r.deleteSizeReservation)).
		Operation("deleteSizeReservation").
		Doc("deletes an size reservation and returns the deleted entity").
		Param(ws.PathParameter("id", "identifier of the size reservation").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.SizeReservationResponse{}).
		Returns(http.StatusOK, "OK", v1.SizeReservationResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.PUT("/reservations").
		To(admin(r.createSizeReservation)).
		Operation("createSizeReservation").
		Doc("create a size reservation. if the given ID already exists a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.SizeReservationCreateRequest{}).
		Returns(http.StatusCreated, "Created", v1.SizeReservationResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/reservations").
		To(admin(r.updateSizeReservation)).
		Operation("updateSizeReservation").
		Doc("updates a size reservation. if the size reservation was changed since this one was read, a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.SizeReservationUpdateRequest{}).
		Returns(http.StatusOK, "OK", v1.SizeReservationResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/reservations/usage").
		To(r.sizeReservationsUsage).
		Operation("sizeReservationsUsage").
		Doc("get all size reservations").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Metadata(auditing.Exclude, true).
		Reads(v1.SizeReservationListRequest{}).
		Writes([]v1.SizeReservationUsageResponse{}).
		Returns(http.StatusOK, "OK", []v1.SizeReservationUsageResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r *sizeResource) findSize(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	s, err := r.ds.FindSize(id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, v1.NewSizeResponse(s))
}

func (r *sizeResource) suggestSize(request *restful.Request, response *restful.Response) {
	var requestPayload v1.SizeSuggestRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	if requestPayload.MachineID == "" {
		r.sendError(request, response, httperrors.BadRequest(fmt.Errorf("machineID must be given")))
		return
	}

	m, err := r.ds.FindMachineByID(requestPayload.MachineID)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	var (
		gpus           = make(map[string]uint64)
		gpuconstraints []v1.SizeConstraint

		cores       uint64
		coresModels []string

		diskCapacity uint64
		diskNames    []string
	)

	for _, gpu := range m.Hardware.MetalGPUs {
		_, ok := gpus[gpu.Model]
		if !ok {
			gpus[gpu.Model] = 1
		} else {
			gpus[gpu.Model]++
		}
	}
	for model, count := range gpus {
		gpuconstraints = append(gpuconstraints, v1.SizeConstraint{
			Type:       metal.GPUConstraint,
			Min:        count,
			Max:        count,
			Identifier: model,
		})
	}

	for _, cpu := range m.Hardware.MetalCPUs {
		cores += uint64(cpu.Cores)
		coresModels = append(coresModels, cpu.Model)
	}

	for _, d := range m.Hardware.Disks {
		diskCapacity += d.Size
		diskNames = append(diskNames, d.Name)
	}

	constraints := []v1.SizeConstraint{
		{
			Type:       metal.CoreConstraint,
			Min:        cores,
			Max:        cores,
			Identifier: longestCommonPrefix(coresModels),
		},
		{
			Type: metal.MemoryConstraint,
			Min:  m.Hardware.Memory,
			Max:  m.Hardware.Memory,
		},
		{
			Type:       metal.StorageConstraint,
			Min:        diskCapacity,
			Max:        diskCapacity,
			Identifier: longestCommonPrefix(diskNames),
		},
	}

	if len(gpuconstraints) > 0 {
		constraints = append(constraints, gpuconstraints...)
	}

	r.send(request, response, http.StatusOK, constraints)
}

func (r *sizeResource) listSizes(request *restful.Request, response *restful.Response) {
	ss, err := r.ds.ListSizes()
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	result := []*v1.SizeResponse{}
	for i := range ss {
		result = append(result, v1.NewSizeResponse(&ss[i]))
	}

	r.send(request, response, http.StatusOK, result)
}

func (r *sizeResource) createSize(request *restful.Request, response *restful.Response) {
	var requestPayload v1.SizeCreateRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	if requestPayload.ID == "" {
		r.sendError(request, response, httperrors.BadRequest(fmt.Errorf("id should not be empty")))
		return
	}

	if requestPayload.ID == metal.UnknownSize().GetID() {
		r.sendError(request, response, httperrors.BadRequest(fmt.Errorf("id cannot be %q", metal.UnknownSize().GetID())))
		return
	}

	var name string
	if requestPayload.Name != nil {
		name = *requestPayload.Name
	}
	var description string
	if requestPayload.Description != nil {
		description = *requestPayload.Description
	}
	labels := map[string]string{}
	if requestPayload.Labels != nil {
		labels = requestPayload.Labels
	}
	var constraints []metal.Constraint
	for _, c := range requestPayload.SizeConstraints {
		constraint := metal.Constraint{
			Type:       c.Type,
			Min:        c.Min,
			Max:        c.Max,
			Identifier: c.Identifier,
		}
		constraints = append(constraints, constraint)
	}

	s := &metal.Size{
		Base: metal.Base{
			ID:          requestPayload.ID,
			Name:        name,
			Description: description,
		},
		Constraints: constraints,
		Labels:      labels,
	}

	ss, err := r.ds.ListSizes()
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	ps, err := r.ds.ListPartitions()
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	projects, err := r.mdc.Project().Find(request.Request.Context(), &mdmv1.ProjectFindRequest{})
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	err = s.Validate(ps.ByID(), projectsByID(projects.Projects))
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	if so := s.Overlaps(&ss); so != nil {
		r.sendError(request, response, httperrors.BadRequest(fmt.Errorf("size %q overlaps with %q", s.GetID(), so.GetID())))
		return
	}

	err = r.ds.CreateSize(s)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusCreated, v1.NewSizeResponse(s))
}

func (r *sizeResource) deleteSize(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	s, err := r.ds.FindSize(id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	err = r.ds.DeleteSize(s)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, v1.NewSizeResponse(s))
}

func (r *sizeResource) updateSize(request *restful.Request, response *restful.Response) {
	var requestPayload v1.SizeUpdateRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	oldSize, err := r.ds.FindSize(requestPayload.ID)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	newSize := *oldSize

	if requestPayload.Name != nil {
		newSize.Name = *requestPayload.Name
	}
	if requestPayload.Description != nil {
		newSize.Description = *requestPayload.Description
	}
	if requestPayload.Labels != nil {
		newSize.Labels = requestPayload.Labels
	}
	var constraints []metal.Constraint
	if requestPayload.SizeConstraints != nil {
		sizeConstraints := *requestPayload.SizeConstraints
		for i := range sizeConstraints {
			constraint := metal.Constraint{
				Type:       sizeConstraints[i].Type,
				Min:        sizeConstraints[i].Min,
				Max:        sizeConstraints[i].Max,
				Identifier: sizeConstraints[i].Identifier,
			}
			constraints = append(constraints, constraint)
		}
		newSize.Constraints = constraints
	}

	ss, err := r.ds.ListSizes()
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	ps, err := r.ds.ListPartitions()
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	projects, err := r.mdc.Project().Find(request.Request.Context(), &mdmv1.ProjectFindRequest{})
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	err = newSize.Validate(ps.ByID(), projectsByID(projects.Projects))
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	if so := newSize.Overlaps(&ss); so != nil {
		r.sendError(request, response, httperrors.BadRequest(fmt.Errorf("size %q overlaps with %q", newSize.GetID(), so.GetID())))
		return
	}

	err = r.ds.UpdateSize(oldSize, &newSize)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, v1.NewSizeResponse(&newSize))
}

func (r *sizeResource) listSizeReservations(request *restful.Request, response *restful.Response) {
	rvs, err := r.ds.ListSizeReservations()
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	result := []*v1.SizeReservationResponse{}
	for i := range rvs {
		result = append(result, v1.NewSizeReservationResponse(&rvs[i]))
	}

	r.send(request, response, http.StatusOK, result)
}

func (r *sizeResource) findSizeReservations(request *restful.Request, response *restful.Response) {
	var requestPayload v1.SizeReservationListRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	var rvs metal.SizeReservations
	err = r.ds.SearchSizeReservations(&datastore.SizeReservationSearchQuery{
		SizeID:    requestPayload.SizeID,
		Partition: requestPayload.PartitionID,
		Project:   requestPayload.ProjectID,
	}, &rvs)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	result := []*v1.SizeReservationResponse{}
	for i := range rvs {
		result = append(result, v1.NewSizeReservationResponse(&rvs[i]))
	}

	r.send(request, response, http.StatusOK, result)
}

func (r *sizeResource) getSizeReservation(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	rv, err := r.ds.FindSizeReservation(id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, v1.NewSizeReservationResponse(rv))
}

func (r *sizeResource) createSizeReservation(request *restful.Request, response *restful.Response) {
	var requestPayload v1.SizeReservationCreateRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	rv := &metal.SizeReservation{
		Base: metal.Base{
			ID:          requestPayload.ID,
			Name:        pointer.SafeDeref(requestPayload.Name),
			Description: pointer.SafeDeref(requestPayload.Description),
		},
		SizeID:       requestPayload.SizeID,
		Amount:       requestPayload.Amount,
		ProjectID:    requestPayload.ProjectID,
		PartitionIDs: requestPayload.PartitionIDs,
		Labels:       requestPayload.Labels,
	}

	ss, err := r.ds.ListSizes()
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	ps, err := r.ds.ListPartitions()
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	projects, err := r.mdc.Project().Find(request.Request.Context(), &mdmv1.ProjectFindRequest{})
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	err = rv.Validate(ss.ByID(), ps.ByID(), projectsByID(projects.Projects))
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	err = r.ds.CreateSizeReservation(rv)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, v1.NewSizeReservationResponse(rv))
}

func (r *sizeResource) updateSizeReservation(request *restful.Request, response *restful.Response) {
	var requestPayload v1.SizeReservationUpdateRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	oldRv, err := r.ds.FindSizeReservation(requestPayload.ID)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	rv := *oldRv

	if requestPayload.Name != nil {
		rv.Name = *requestPayload.Name
	}
	if requestPayload.Description != nil {
		rv.Description = *requestPayload.Description
	}
	if requestPayload.Labels != nil {
		if len(requestPayload.Labels) == 0 {
			rv.Labels = nil
		} else {
			rv.Labels = requestPayload.Labels
		}
	}
	if requestPayload.Amount != nil {
		rv.Amount = *requestPayload.Amount
	}
	if len(requestPayload.PartitionIDs) > 0 {
		rv.PartitionIDs = requestPayload.PartitionIDs
	}

	ss, err := r.ds.ListSizes()
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	ps, err := r.ds.ListPartitions()
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	projects, err := r.mdc.Project().Find(request.Request.Context(), &mdmv1.ProjectFindRequest{})
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	err = rv.Validate(ss.ByID(), ps.ByID(), projectsByID(projects.Projects))
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	err = r.ds.UpdateSizeReservation(oldRv, &rv)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, v1.NewSizeReservationResponse(&rv))
}

func (r *sizeResource) deleteSizeReservation(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	rv, err := r.ds.FindSizeReservation(id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	err = r.ds.DeleteSizeReservation(rv)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, v1.NewSizeReservationResponse(rv))
}

func (r *sizeResource) sizeReservationsUsage(request *restful.Request, response *restful.Response) {
	var requestPayload v1.SizeReservationListRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	rvs := metal.SizeReservations{}
	err = r.ds.SearchSizeReservations(&datastore.SizeReservationSearchQuery{
		SizeID:    requestPayload.SizeID,
		Partition: requestPayload.PartitionID,
		Project:   requestPayload.ProjectID,
	}, &rvs)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	ms := metal.Machines{}
	err = r.ds.SearchMachines(&datastore.MachineSearchQuery{
		PartitionID: requestPayload.PartitionID,
	}, &ms)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	var (
		result              []*v1.SizeReservationUsageResponse
		machinesByProjectID = ms.ByProjectID()
	)

	for _, reservation := range rvs {
		for _, partitionID := range reservation.PartitionIDs {
			allocations := len(machinesByProjectID[reservation.ProjectID].WithPartition(partitionID).WithSize(reservation.SizeID))

			result = append(result, &v1.SizeReservationUsageResponse{
				Common: v1.Common{
					Identifiable: v1.Identifiable{ID: reservation.ID},
					Describable:  v1.Describable{Name: &reservation.Name, Description: &reservation.Description},
				},
				SizeID:             reservation.SizeID,
				PartitionID:        partitionID,
				ProjectID:          reservation.ProjectID,
				Amount:             reservation.Amount,
				UsedAmount:         min(reservation.Amount, allocations),
				ProjectAllocations: allocations,
				Labels:             reservation.Labels,
			})
		}
	}

	r.send(request, response, http.StatusOK, result)
}

// longestCommonPrefix finds the longest prefix of a slice of strings.
func longestCommonPrefix(strs []string) string {
	longestPrefix := ""

	if len(strs) > 0 {
		// by sorting the strings we can quite efficiently find the longest prefix
		// we just need to compare the first and last element after the sort
		sort.Strings(strs)
		first := strs[0]
		last := strs[len(strs)-1]

		for i := 0; i < len(first); i++ {
			// append as long as chars are equal, stop when different

			if last[i] != first[i] {
				break
			}

			longestPrefix += string(last[i])
		}

		// append * to pattern if not identical
		if len(last) != len(longestPrefix) {
			longestPrefix += "*"
		}
	}

	return longestPrefix
}
