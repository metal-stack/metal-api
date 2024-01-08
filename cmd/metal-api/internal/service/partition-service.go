package service

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/issues"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-lib/auditing"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"go.uber.org/zap"

	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-lib/httperrors"
)

// TopicCreator creates a topic for messaging.
type TopicCreator interface {
	CreateTopic(topicFQN string) error
}

type partitionResource struct {
	webResource
	topicCreator TopicCreator
}

// NewPartition returns a webservice for partition specific endpoints.
func NewPartition(log *zap.SugaredLogger, ds *datastore.RethinkStore, tc TopicCreator) *restful.WebService {
	r := partitionResource{
		webResource: webResource{
			log: log,
			ds:  ds,
		},
		topicCreator: tc,
	}

	return r.webService()
}

func (r *partitionResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path(BasePath + "v1/partition").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"Partition"}

	ws.Route(ws.GET("/{id}").
		To(r.findPartition).
		Operation("findPartition").
		Doc("get Partition by id").
		Param(ws.PathParameter("id", "identifier of the Partition").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.PartitionResponse{}).
		Returns(http.StatusOK, "OK", v1.PartitionResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/").
		To(r.listPartitions).
		Operation("listPartitions").
		Doc("get all Partitions").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]v1.PartitionResponse{}).
		Returns(http.StatusOK, "OK", []v1.PartitionResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.DELETE("/{id}").
		To(admin(r.deletePartition)).
		Operation("deletePartition").
		Doc("deletes a Partition and returns the deleted entity").
		Param(ws.PathParameter("id", "identifier of the Partition").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.PartitionResponse{}).
		Returns(http.StatusOK, "OK", v1.PartitionResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.PUT("/").
		To(admin(r.createPartition)).
		Operation("createPartition").
		Doc("create a Partition. if the given ID already exists a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.PartitionCreateRequest{}).
		Returns(http.StatusCreated, "Created", v1.PartitionResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/").
		To(admin(r.updatePartition)).
		Operation("updatePartition").
		Doc("updates a Partition. if the Partition was changed since this one was read, a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.PartitionUpdateRequest{}).
		Returns(http.StatusOK, "OK", v1.PartitionResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/capacity").
		To(r.partitionCapacity).
		Operation("partitionCapacity").
		Doc("get partition capacity").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Metadata(auditing.Exclude, true).
		Reads(v1.PartitionCapacityRequest{}).
		Writes([]v1.PartitionCapacity{}).
		Returns(http.StatusOK, "OK", []v1.PartitionCapacity{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r *partitionResource) findPartition(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	p, err := r.ds.FindPartition(id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, v1.NewPartitionResponse(p))
}

func (r *partitionResource) listPartitions(request *restful.Request, response *restful.Response) {
	ps, err := r.ds.ListPartitions()
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	result := []*v1.PartitionResponse{}
	for i := range ps {
		result = append(result, v1.NewPartitionResponse(&ps[i]))
	}

	r.send(request, response, http.StatusOK, result)
}

func (r *partitionResource) createPartition(request *restful.Request, response *restful.Response) {
	var requestPayload v1.PartitionCreateRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	if requestPayload.ID == "" {
		r.sendError(request, response, httperrors.BadRequest(errors.New("id should not be empty")))
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
	var mgmtServiceAddress string
	if requestPayload.MgmtServiceAddress != nil {
		mgmtServiceAddress = *requestPayload.MgmtServiceAddress
	}
	prefixLength := uint8(22)
	if requestPayload.PrivateNetworkPrefixLength != nil {
		prefixLength = uint8(*requestPayload.PrivateNetworkPrefixLength)
		if prefixLength < 16 || prefixLength > 30 {
			r.sendError(request, response, httperrors.BadRequest(errors.New("private network prefix length is out of range")))
			return
		}
	}
	var imageURL string
	if requestPayload.PartitionBootConfiguration.ImageURL != nil {
		imageURL = *requestPayload.PartitionBootConfiguration.ImageURL
	}

	err = checkImageURL("image", imageURL)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	var kernelURL string
	if requestPayload.PartitionBootConfiguration.KernelURL != nil {
		kernelURL = *requestPayload.PartitionBootConfiguration.KernelURL
	}

	err = checkImageURL("kernel", kernelURL)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	var commandLine string
	if requestPayload.PartitionBootConfiguration.CommandLine != nil {
		commandLine = *requestPayload.PartitionBootConfiguration.CommandLine
	}

	p := &metal.Partition{
		Base: metal.Base{
			ID:          requestPayload.ID,
			Name:        name,
			Description: description,
		},
		MgmtServiceAddress:         mgmtServiceAddress,
		PrivateNetworkPrefixLength: prefixLength,
		BootConfiguration: metal.BootConfiguration{
			ImageURL:    imageURL,
			KernelURL:   kernelURL,
			CommandLine: commandLine,
		},
	}

	fqn := metal.TopicMachine.GetFQN(p.GetID())
	if err := r.topicCreator.CreateTopic(fqn); err != nil {
		r.sendError(request, response, httperrors.InternalServerError(err))
		return
	}

	err = r.ds.CreatePartition(p)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusCreated, v1.NewPartitionResponse(p))
}

func (r *partitionResource) deletePartition(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	p, err := r.ds.FindPartition(id)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	err = r.ds.DeletePartition(p)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	r.send(request, response, http.StatusOK, v1.NewPartitionResponse(p))
}

func (r *partitionResource) updatePartition(request *restful.Request, response *restful.Response) {
	var requestPayload v1.PartitionUpdateRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	oldPartition, err := r.ds.FindPartition(requestPayload.ID)
	if err != nil {
		r.sendError(request, response, defaultError(err))
		return
	}

	newPartition := *oldPartition

	if requestPayload.Name != nil {
		newPartition.Name = *requestPayload.Name
	}
	if requestPayload.Description != nil {
		newPartition.Description = *requestPayload.Description
	}
	if requestPayload.MgmtServiceAddress != nil {
		newPartition.MgmtServiceAddress = *requestPayload.MgmtServiceAddress
	}
	if requestPayload.PartitionBootConfiguration.ImageURL != nil {
		err = checkImageURL("image", *requestPayload.PartitionBootConfiguration.ImageURL)
		if err != nil {
			r.sendError(request, response, httperrors.BadRequest(err))
			return
		}

		newPartition.BootConfiguration.ImageURL = *requestPayload.PartitionBootConfiguration.ImageURL
	}

	if requestPayload.PartitionBootConfiguration.KernelURL != nil {
		err = checkImageURL("kernel", *requestPayload.PartitionBootConfiguration.KernelURL)
		if err != nil {
			r.sendError(request, response, httperrors.BadRequest(err))
			return
		}
		newPartition.BootConfiguration.KernelURL = *requestPayload.PartitionBootConfiguration.KernelURL
	}
	if requestPayload.PartitionBootConfiguration.CommandLine != nil {
		newPartition.BootConfiguration.CommandLine = *requestPayload.PartitionBootConfiguration.CommandLine
	}

	err = r.ds.UpdatePartition(oldPartition, &newPartition)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	r.send(request, response, http.StatusOK, v1.NewPartitionResponse(&newPartition))
}

func (r *partitionResource) partitionCapacity(request *restful.Request, response *restful.Response) {
	var requestPayload v1.PartitionCapacityRequest
	err := request.ReadEntity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	partitionCapacities, err := r.calcPartitionCapacity(&requestPayload)
	if err != nil {
		r.sendError(request, response, httperrors.BadRequest(err))
		return
	}

	r.send(request, response, http.StatusOK, partitionCapacities)
}

func (r *partitionResource) calcPartitionCapacity(pcr *v1.PartitionCapacityRequest) ([]v1.PartitionCapacity, error) {
	var (
		ps metal.Partitions
		ms metal.Machines

		pcs = map[string]*v1.PartitionCapacity{}

		machineQuery = datastore.MachineSearchQuery{}
	)

	if pcr != nil && pcr.ID != nil {
		p, err := r.ds.FindPartition(*pcr.ID)
		if err != nil {
			return nil, err
		}
		ps = metal.Partitions{*p}

		machineQuery.PartitionID = pcr.ID
	} else {
		var err error
		ps, err = r.ds.ListPartitions()
		if err != nil {
			return nil, err
		}
	}

	if pcr != nil && pcr.Size != nil {
		machineQuery.SizeID = pcr.Size
	}

	err := r.ds.SearchMachines(&machineQuery, &ms)
	if err != nil {
		return nil, err
	}

	ecs, err := r.ds.ListProvisioningEventContainers()
	if err != nil {
		return nil, fmt.Errorf("unable to fetch provisioning event containers: %w", err)
	}

	sizes, err := r.ds.ListSizes()
	if err != nil {
		return nil, fmt.Errorf("unable to list sizes: %w", err)
	}

	machinesWithIssues, err := issues.Find(&issues.Config{
		Machines:        ms,
		EventContainers: ecs,
		Only:            issues.NotAllocatableIssueTypes(),
	})
	if err != nil {
		return nil, fmt.Errorf("unable to calculate machine issues: %w", err)
	}

	var (
		partitionsByID    = ps.ByID()
		ecsByID           = ecs.ByID()
		sizesByID         = sizes.ByID()
		machinesByProject = ms.ByProjectID()
	)

	for _, m := range ms {
		m := m

		ec, ok := ecsByID[m.ID]
		if !ok {
			continue
		}

		p, ok := partitionsByID[m.PartitionID]
		if !ok {
			continue
		}

		pc, ok := pcs[m.PartitionID]
		if !ok {
			pc = &v1.PartitionCapacity{
				Common: v1.Common{
					Identifiable: v1.Identifiable{
						ID: p.ID,
					},
					Describable: v1.Describable{
						Name:        &p.Name,
						Description: &p.Description,
					},
				},
				ServerCapacities: v1.ServerCapacities{},
			}
		}
		pcs[m.PartitionID] = pc

		size, ok := sizesByID[m.SizeID]
		if !ok {
			size = *metal.UnknownSize()
		}

		cap := pc.ServerCapacities.FindBySize(size.ID)
		if cap == nil {
			cap = &v1.ServerCapacity{
				Size: size.ID,
			}
			pc.ServerCapacities = append(pc.ServerCapacities, cap)
		}

		cap.Total++

		for _, reservation := range size.Reservations {
			reservation := reservation

			cap.Reservations += reservation.Amount
			cap.UnusedReservations += max(reservation.Amount-len(machinesByProject[reservation.ProjectID]), 0)
		}

		if m.Allocation != nil {
			cap.Allocated++
			continue
		}

		if _, ok := machinesWithIssues[m.ID]; ok {
			cap.Faulty++
			cap.FaultyMachines = append(cap.FaultyMachines, m.ID)
			continue
		}

		if m.State.Value == metal.AvailableState && metal.ProvisioningEventWaiting == pointer.FirstOrZero(ec.Events).Event {
			cap.Free++
			continue
		}

		cap.Other++
		cap.OtherMachines = append(cap.OtherMachines, m.ID)
	}

	res := []v1.PartitionCapacity{}
	for _, pc := range pcs {
		pc := pc
		res = append(res, *pc)
	}

	return res, nil
}
