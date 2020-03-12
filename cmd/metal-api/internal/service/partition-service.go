package service

import (
	"net/http"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/utils"
	"go.uber.org/zap"

	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"

	"fmt"

	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/metal-lib/zapup"
	"github.com/prometheus/client_golang/prometheus"
)

type TopicCreater interface {
	CreateTopic(partitionID, topicFQN string) error
}

type partitionResource struct {
	webResource
	topicCreater TopicCreater
}

// NewPartition returns a webservice for partition specific endpoints.
func NewPartition(ds *datastore.RethinkStore, tc TopicCreater) *restful.WebService {
	r := partitionResource{
		webResource: webResource{
			ds: ds,
		},
		topicCreater: tc,
	}
	pcc := partitionCapacityCollector{r: &r}
	err := prometheus.Register(pcc)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to register prometheus", zap.Error(err))
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
	prefixLength := 22
	if requestPayload.PrivateNetworkPrefixLength != nil {
		prefixLength = *requestPayload.PrivateNetworkPrefixLength
		if prefixLength < 16 || prefixLength > 30 {
			if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("private network prefix length is out of range")) {
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
		PrivateNetworkPrefixLength: prefixLength,
		BootConfiguration: metal.BootConfiguration{
			ImageURL:    imageURL,
			KernelURL:   kernelURL,
			CommandLine: commandLine,
		},
	}

	fqns := []string{metal.TopicMachine.GetFQN(p.GetID()), metal.TopicSwitch.GetFQN(p.GetID())}
	for _, fqn := range fqns {
		if err := r.topicCreater.CreateTopic(p.GetID(), fqn); err != nil {
			if checkError(request, response, utils.CurrentFuncName(), err) {
				return
			}
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
	err = response.WriteHeaderAndEntity(http.StatusOK, v1.NewPartitionResponse(&newPartition))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r partitionResource) partitionCapacity(request *restful.Request, response *restful.Response) {
	partitionCapacities, err := r.calcPartitionCapacity()

	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, partitionCapacities)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r partitionResource) calcPartitionCapacity() ([]v1.PartitionCapacity, error) {
	// FIXME bad workaround to be able to run make spec
	if r.ds == nil {
		return nil, nil
	}
	ps, err := r.ds.ListPartitions()
	if err != nil {
		return nil, err
	}
	ms, err := r.ds.ListMachines()
	if err != nil {
		return nil, err
	}
	machines := makeMachineResponseList(ms, r.ds, zapup.MustRootLogger().Sugar())

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
			if len(m.RecentProvisioningEvents.Events) > 0 {
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
			} else if available {
				cap.Free++
			}

			cap.Total++
		}

		sc := []v1.ServerCapacity{}
		for i := range capacities {
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

// partitionCapacityCollector implements the Collector interface.
type partitionCapacityCollector struct {
	r *partitionResource
}

var (
	capacityTotalDesc = prometheus.NewDesc(
		"metal_partition_capacity_total",
		"The total capacity of machines in the partition",
		[]string{"partition", "size"}, nil,
	)
	capacityFreeDesc = prometheus.NewDesc(
		"metal_partition_capacity_free",
		"The capacity of free machines in the partition",
		[]string{"partition", "size"}, nil,
	)
	capacityAllocatedDesc = prometheus.NewDesc(
		"metal_partition_capacity_allocated",
		"The capacity of allocated machines in the partition",
		[]string{"partition", "size"}, nil,
	)
	capacityFaultyDesc = prometheus.NewDesc(
		"metal_partition_capacity_faulty",
		"The capacity of faulty machines in the partition",
		[]string{"partition", "size"}, nil,
	)
)

func (pcc partitionCapacityCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(pcc, ch)
}

func (pcc partitionCapacityCollector) Collect(ch chan<- prometheus.Metric) {
	pcs, err := pcc.r.calcPartitionCapacity()
	if err != nil {
		zapup.MustRootLogger().Error("Failed to get partition capacity", zap.Error(err))
		return
	}

	for _, pc := range pcs {
		for _, sc := range pc.ServerCapacities {
			metric, err := prometheus.NewConstMetric(
				capacityTotalDesc,
				prometheus.CounterValue,
				float64(sc.Total),
				pc.ID,
				sc.Size,
			)
			if err != nil {
				zapup.MustRootLogger().Error("Failed to create metric for totalCapacity", zap.Error(err))
				return
			}
			ch <- metric

			metric, err = prometheus.NewConstMetric(
				capacityFreeDesc,
				prometheus.CounterValue,
				float64(sc.Free),
				pc.ID,
				sc.Size,
			)
			if err != nil {
				zapup.MustRootLogger().Error("Failed to create metric for freeCapacity", zap.Error(err))
				return
			}
			ch <- metric
			metric, err = prometheus.NewConstMetric(
				capacityAllocatedDesc,
				prometheus.CounterValue,
				float64(sc.Allocated),
				pc.ID,
				sc.Size,
			)
			if err != nil {
				zapup.MustRootLogger().Error("Failed to create metric for allocatedCapacity", zap.Error(err))
				return
			}
			ch <- metric
			metric, err = prometheus.NewConstMetric(
				capacityFaultyDesc,
				prometheus.CounterValue,
				float64(sc.Faulty),
				pc.ID,
				sc.Size,
			)
			if err != nil {
				zapup.MustRootLogger().Error("Failed to create metric for faultyCapacity", zap.Error(err))
				return
			}
			ch <- metric
		}
	}
}
