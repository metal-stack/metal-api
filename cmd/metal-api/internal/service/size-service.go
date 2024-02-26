package service

import (
	"errors"
	"fmt"
	"net/http"

	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	mdm "github.com/metal-stack/masterdata-api/pkg/client"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-lib/auditing"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"go.uber.org/zap"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-lib/httperrors"
)

type sizeResource struct {
	webResource
	mdc mdm.Client
}

// NewSize returns a webservice for size specific endpoints.
func NewSize(log *zap.SugaredLogger, ds *datastore.RethinkStore, mdc mdm.Client) *restful.WebService {
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

	ws.Route(ws.POST("/reservations").
		To(r.listSizeReservations).
		Operation("listSizeReservations").
		Doc("get all size reservations").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Metadata(auditing.Exclude, true).
		Reads(v1.EmptyBody{}).
		Writes([]v1.SizeReservationResponse{}).
		Returns(http.StatusOK, "OK", []v1.SizeReservationResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/suggest").
		To(r.suggestSize).
		Operation("suggest").
		Doc("from a given machine id returns the appropriate size").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Metadata(auditing.Exclude, true).
		Reads(v1.SizeSuggestRequest{}).
		Returns(http.StatusOK, "OK", []v1.SizeConstraint{}).
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

	ws.Route(ws.POST("/from-hardware").
		To(r.fromHardware).
		Operation("fromHardware").
		Doc("Searches all sizes for one to match the given hardwarespecs. If nothing is found, a list of entries is returned which describe the constraint which did not match").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Metadata(auditing.Exclude, true).
		Reads(v1.MachineHardware{}).
		Returns(http.StatusOK, "OK", v1.SizeMatchingLog{}).
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

	r.send(request, response, http.StatusOK, []v1.SizeConstraint{
		{
			Type: metal.CoreConstraint,
			Min:  uint64(m.Hardware.CPUCores),
			Max:  uint64(m.Hardware.CPUCores),
		},
		{
			Type: metal.MemoryConstraint,
			Min:  m.Hardware.Memory,
			Max:  m.Hardware.Memory,
		},
		{
			Type: metal.StorageConstraint,
			Min:  m.Hardware.DiskCapacity(),
			Max:  m.Hardware.DiskCapacity(),
		},
	})

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
			Type: c.Type,
			Min:  c.Min,
			Max:  c.Max,
		}
		constraints = append(constraints, constraint)
	}
	var reservations metal.Reservations
	for _, r := range requestPayload.SizeReservations {
		reservations = append(reservations, metal.Reservation{
			Amount:       r.Amount,
			Description:  r.Description,
			ProjectID:    r.ProjectID,
			PartitionIDs: r.PartitionIDs,
		})
	}

	s := &metal.Size{
		Base: metal.Base{
			ID:          requestPayload.ID,
			Name:        name,
			Description: description,
		},
		Constraints:  constraints,
		Reservations: reservations,
		Labels:       labels,
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
		r.sendError(request, response, httperrors.BadRequest(fmt.Errorf("size overlaps with %q", so.GetID())))
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
				Type: sizeConstraints[i].Type,
				Min:  sizeConstraints[i].Min,
				Max:  sizeConstraints[i].Max,
			}
			constraints = append(constraints, constraint)
		}
		newSize.Constraints = constraints
	}
	var reservations metal.Reservations
	if requestPayload.SizeReservations != nil {
		for _, r := range requestPayload.SizeReservations {
			reservations = append(reservations, metal.Reservation{
				Amount:       r.Amount,
				Description:  r.Description,
				ProjectID:    r.ProjectID,
				PartitionIDs: r.PartitionIDs,
			})
		}
		newSize.Reservations = reservations
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
		r.sendError(request, response, httperrors.BadRequest(fmt.Errorf("size overlaps with %q", so.GetID())))
		return
	}

	err = r.ds.UpdateSize(oldSize, &newSize)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, v1.NewSizeResponse(&newSize))
}

func (r *sizeResource) fromHardware(request *restful.Request, response *restful.Response) {
	var requestPayload v1.MachineHardware
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	hw := v1.NewMetalMachineHardware(&requestPayload)
	_, lg, err := r.ds.FromHardware(hw)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	if len(lg) < 1 {
		r.sendError(request, response, httperrors.UnprocessableEntity(errors.New("size matching log is empty")))
		return
	}

	r.send(request, response, http.StatusOK, v1.NewSizeMatchingLog(lg[0]))
}

func (r *sizeResource) listSizeReservations(request *restful.Request, response *restful.Response) {
	ss, err := r.ds.ListSizes()
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	projects, err := r.mdc.Project().Find(request.Request.Context(), &mdmv1.ProjectFindRequest{})
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	ms, err := r.ds.ListMachines()
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	var (
		result              []*v1.SizeReservationResponse
		projectsByID        = projectsByID(projects.Projects)
		machinesByProjectID = ms.ByProjectID()
	)

	for _, size := range ss {
		size := size

		for _, reservation := range size.Reservations {
			reservation := reservation

			for _, partitionID := range reservation.PartitionIDs {
				project := pointer.SafeDeref(projectsByID[reservation.ProjectID])
				allocations := len(machinesByProjectID[reservation.ProjectID].WithPartition(partitionID).WithSize(size.ID))

				result = append(result, &v1.SizeReservationResponse{
					SizeID:             size.ID,
					PartitionID:        partitionID,
					Tenant:             project.TenantId,
					ProjectID:          reservation.ProjectID,
					ProjectName:        project.Name,
					Reservations:       reservation.Amount,
					UsedReservations:   min(reservation.Amount, allocations),
					ProjectAllocations: allocations,
				})
			}
		}
	}

	r.send(request, response, http.StatusOK, result)
}
