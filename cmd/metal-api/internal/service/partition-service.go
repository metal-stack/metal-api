package service

import (
	"net/http"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/utils"

	v1 "git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/service/v1"

	"fmt"

	"git.f-i-ts.de/cloud-native/metallib/httperrors"
	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
)

type partitionResource struct {
	webResource
}

// NewPartition returns a webservice for partition specific endpoints.
func NewPartition(ds *datastore.RethinkStore) *restful.WebService {
	r := partitionResource{
		webResource: webResource{
			ds: ds,
		},
	}
	return r.webService()
}

func (r partitionResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path("/v1/partition").
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

	ws.Route(ws.GET("/capacity").
		To(r.partitionCapacity).
		Operation("partitionCapacity").
		Doc("get Partition capacity").
		Metadata(restfulspec.KeyOpenAPITags, tags).
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

	response.WriteHeaderAndEntity(http.StatusOK, v1.NewPartitionResponse(p))
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

	response.WriteHeaderAndEntity(http.StatusOK, result)
}

func (r partitionResource) createPartition(request *restful.Request, response *restful.Response) {
	var requestPayload v1.PartitionCreateRequest
	err := request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	if requestPayload.ID == "" {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("id should not be empty")) {
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
	projectNetworkPrefixLength := 22
	if requestPayload.ProjectNetworkPrefixLength != nil {
		projectNetworkPrefixLength = *requestPayload.ProjectNetworkPrefixLength
		if projectNetworkPrefixLength < 16 || projectNetworkPrefixLength > 30 {
			if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("projectNetworkPrefixLength is out of range")) {
				return
			}
		}
	}
	var imageURL string
	if requestPayload.PartitionBootConfiguration.ImageURL != nil {
		imageURL = *requestPayload.PartitionBootConfiguration.ImageURL
	}
	var kernelURL string
	if requestPayload.PartitionBootConfiguration.KernelURL != nil {
		kernelURL = *requestPayload.PartitionBootConfiguration.KernelURL
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
		ProjectNetworkPrefixLength: projectNetworkPrefixLength,
		BootConfiguration: metal.BootConfiguration{
			ImageURL:    imageURL,
			KernelURL:   kernelURL,
			CommandLine: commandLine,
		},
	}

	err = r.ds.CreatePartition(p)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusCreated, v1.NewPartitionResponse(p))
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

	response.WriteHeaderAndEntity(http.StatusOK, v1.NewPartitionResponse(p))
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
		newPartition.BootConfiguration.ImageURL = *requestPayload.PartitionBootConfiguration.ImageURL
	}
	if requestPayload.PartitionBootConfiguration.KernelURL != nil {
		newPartition.BootConfiguration.KernelURL = *requestPayload.PartitionBootConfiguration.KernelURL
	}
	if requestPayload.PartitionBootConfiguration.CommandLine != nil {
		newPartition.BootConfiguration.CommandLine = *requestPayload.PartitionBootConfiguration.CommandLine
	}

	err = r.ds.UpdatePartition(oldPartition, &newPartition)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, v1.NewPartitionResponse(&newPartition))
}

func (r partitionResource) partitionCapacity(request *restful.Request, response *restful.Response) {
	ps, err := r.ds.ListPartitions()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	ms, err := r.ds.ListMachines()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	machines := makeMachineResponseList(ms, r.ds, utils.Logger(request).Sugar())

	partitionCapacities := []v1.PartitionCapacity{}
	for _, p := range ps {

		capacities := make(map[string]v1.ServerCapacity)
		for _, m := range machines {
			if m.Partition == nil {
				continue
			}
			if m.Partition.ID != p.ID {
				continue
			}
			size := "unknown"
			if m.Size != nil {
				size = m.Size.ID
			}
			available := false
			if len(m.RecentProvisioningEvents.Events) > 0 {
				events := m.RecentProvisioningEvents.Events
				if metal.ProvisioningEventWaiting.Is(events[0].Event) && metal.ProvisioningEventAlive.Is(m.Liveliness) {
					available = true
				}
			}
			oldCap, ok := capacities[size]
			total := 1
			free := 0
			if ok {
				total = oldCap.Total + 1
			}
			if available {
				free = 1
			}

			cap := v1.ServerCapacity{
				Size:  size,
				Total: total,
				Free:  oldCap.Free + free,
			}
			capacities[size] = cap
		}
		sc := []v1.ServerCapacity{}
		for _, c := range capacities {
			sc = append(sc, c)
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

	response.WriteHeaderAndEntity(http.StatusOK, partitionCapacities)
}
