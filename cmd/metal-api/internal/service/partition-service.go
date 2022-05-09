package service

import (
	"errors"
	"net/http"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/utils"
	"go.uber.org/zap"

	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/metal-lib/zapup"
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
func NewPartition(ds *datastore.RethinkStore, tc TopicCreator) *restful.WebService {
	r := partitionResource{
		webResource: webResource{
			ds: ds,
		},
		topicCreator: tc,
	}

	return r.webService()
}

func (r partitionResource) webService() *restful.WebService {
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

	// Deprecated, can be removed in the future
	ws.Route(ws.GET("/capacity").
		To(r.partitionCapacityCompat).
		Operation("partitionCapacity").
		Doc("get Partition capacity").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Deprecate().
		Writes([]v1.PartitionCapacity{}).
		Returns(http.StatusOK, "OK", []v1.PartitionCapacity{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/capacity").
		To(r.partitionCapacity).
		Operation("partitionCapacity").
		Doc("get partition capacity").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.PartitionCapacityRequest{}).
		Writes([]v1.PartitionCapacity{}).
		Returns(http.StatusOK, "OK", []v1.PartitionCapacity{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r partitionResource) findPartition(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	p, err := r.ds.FindPartition(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, v1.NewPartitionResponse(p))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r partitionResource) listPartitions(request *restful.Request, response *restful.Response) {
	ps, err := r.ds.ListPartitions()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	result := []*v1.PartitionResponse{}
	for i := range ps {
		result = append(result, v1.NewPartitionResponse(&ps[i]))
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, result)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r partitionResource) createPartition(request *restful.Request, response *restful.Response) {
	var requestPayload v1.PartitionCreateRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	if requestPayload.ID == "" {
		if checkError(request, response, utils.CurrentFuncName(), errors.New("id should not be empty")) {
			return
		}
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
			if checkError(request, response, utils.CurrentFuncName(), errors.New("private network prefix length is out of range")) {
				return
			}
		}
	}
	var imageURL string
	if requestPayload.PartitionBootConfiguration.ImageURL != nil {
		imageURL = *requestPayload.PartitionBootConfiguration.ImageURL
	}
	err = checkImageURL("image", imageURL)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	var kernelURL string
	if requestPayload.PartitionBootConfiguration.KernelURL != nil {
		kernelURL = *requestPayload.PartitionBootConfiguration.KernelURL
	}
	err = checkImageURL("kernel", kernelURL)
	if checkError(request, response, utils.CurrentFuncName(), err) {
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
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
	}

	err = r.ds.CreatePartition(p)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = response.WriteHeaderAndEntity(http.StatusCreated, v1.NewPartitionResponse(p))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r partitionResource) deletePartition(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	p, err := r.ds.FindPartition(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = r.ds.DeletePartition(p)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, v1.NewPartitionResponse(p))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r partitionResource) updatePartition(request *restful.Request, response *restful.Response) {
	var requestPayload v1.PartitionUpdateRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	oldPartition, err := r.ds.FindPartition(requestPayload.ID)
	if checkError(request, response, utils.CurrentFuncName(), err) {
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
		err = checkImageURL("kernel", *requestPayload.PartitionBootConfiguration.ImageURL)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
		newPartition.BootConfiguration.ImageURL = *requestPayload.PartitionBootConfiguration.ImageURL
	}

	if requestPayload.PartitionBootConfiguration.KernelURL != nil {
		err = checkImageURL("kernel", *requestPayload.PartitionBootConfiguration.KernelURL)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
		newPartition.BootConfiguration.KernelURL = *requestPayload.PartitionBootConfiguration.KernelURL
	}
	if requestPayload.PartitionBootConfiguration.CommandLine != nil {
		newPartition.BootConfiguration.CommandLine = *requestPayload.PartitionBootConfiguration.CommandLine
	}

	err = r.ds.UpdatePartition(oldPartition, &newPartition)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, v1.NewPartitionResponse(&newPartition))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r partitionResource) partitionCapacityCompat(request *restful.Request, response *restful.Response) {
	partitionCapacities, err := r.calcPartitionCapacity(nil)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = response.WriteHeaderAndEntity(http.StatusOK, partitionCapacities)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r partitionResource) partitionCapacity(request *restful.Request, response *restful.Response) {
	var requestPayload v1.PartitionCapacityRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	partitionCapacities, err := r.calcPartitionCapacity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = response.WriteHeaderAndEntity(http.StatusOK, partitionCapacities)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r partitionResource) calcPartitionCapacity(pcr *v1.PartitionCapacityRequest) ([]v1.PartitionCapacity, error) {
	// FIXME bad workaround to be able to run make spec
	if r.ds == nil {
		return nil, nil
	}

	var (
		ps  metal.Partitions
		ms  metal.Machines
		err error
	)

	if pcr != nil && pcr.ID != nil {
		p, err := r.ds.FindPartition(*pcr.ID)
		if err != nil {
			return nil, err
		}
		ps = metal.Partitions{*p}
	} else {
		ps, err = r.ds.ListPartitions()
		if err != nil {
			return nil, err
		}
	}

	msq := datastore.MachineSearchQuery{}
	if pcr != nil && pcr.Size != nil {
		msq.SizeID = pcr.Size
	}

	err = r.ds.SearchMachines(&msq, &ms)
	if err != nil {
		return nil, err
	}
	machines, err := makeMachineResponseList(ms, r.ds)
	if err != nil {
		return nil, err
	}

	partitionCapacities := []v1.PartitionCapacity{}
	for _, p := range ps {
		capacities := make(map[string]*v1.ServerCapacity)
		for _, m := range machines {
			if m.Partition == nil {
				continue
			}
			if m.Partition.ID != p.ID {
				continue
			}

			size := metal.UnknownSize.ID
			if m.Size != nil {
				size = m.Size.ID
			}

			available := false
			if m.State.Value == string(metal.AvailableState) && len(m.RecentProvisioningEvents.Events) > 0 {
				events := m.RecentProvisioningEvents.Events
				if metal.ProvisioningEventWaiting.Is(events[0].Event) && metal.ProvisioningEventAlive.Is(m.Liveliness) {
					available = true
				}
			}

			cap, ok := capacities[size]
			if !ok {
				cap = &v1.ServerCapacity{Size: size}
				capacities[size] = cap
			}

			if m.Allocation != nil {
				cap.Allocated++
			} else if machineHasIssues(m) {
				cap.Faulty++
				cap.FaultyMachines = append(cap.FaultyMachines, m.ID)
			} else if available {
				cap.Free++
			} else {
				cap.Other++
				cap.OtherMachines = append(cap.OtherMachines, m.ID)
			}

			cap.Total++
		}
		sc := []v1.ServerCapacity{}
		for i := range capacities {
			if capacities[i] == nil {
				continue
			}
			sc = append(sc, *capacities[i])
		}

		pc := v1.PartitionCapacity{
			Common: v1.Common{
				Identifiable: v1.Identifiable{
					ID: p.ID,
				},
				Describable: v1.Describable{
					Name:        &p.Name,
					Description: &p.Description,
				},
			},
			ServerCapacities: sc,
		}

		partitionCapacities = append(partitionCapacities, pc)
	}

	return partitionCapacities, err
}
