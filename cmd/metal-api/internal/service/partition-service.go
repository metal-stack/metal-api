package service

import (
	"net/http"
	"time"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metallib/httperrors"
	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
)

type partitionResource struct {
	webResource
}

// NewPartition returns a webservice for partition specific endpoints.
func NewPartition(ds *datastore.RethinkStore) *restful.WebService {
	fr := partitionResource{
		webResource: webResource{
			ds: ds,
		},
	}
	return fr.webService()
}

func (fr partitionResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path("/v1/partition").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"Partition"}

	ws.Route(ws.GET("/{id}").
		To(fr.restEntityGet(fr.ds.FindPartition)).
		Operation("findPartition").
		Doc("get Partition by id").
		Param(ws.PathParameter("id", "identifier of the Partition").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(metal.Partition{}).
		Returns(http.StatusOK, "OK", metal.Partition{}).
		Returns(http.StatusNotFound, "Not Found", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/").
		To(fr.restListGet(fr.ds.ListPartitions)).
		Operation("listPartitions").
		Doc("get all Partitions").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]metal.Partition{}).
		Returns(http.StatusOK, "OK", []metal.Partition{}))

	ws.Route(ws.DELETE("/{id}").
		To(fr.restEntityGet(fr.ds.DeletePartition)).
		Operation("deletePartition").
		Doc("deletes a Partition and returns the deleted entity").
		Param(ws.PathParameter("id", "identifier of the Partition").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(metal.Partition{}).
		Returns(http.StatusOK, "OK", metal.Partition{}).
		Returns(http.StatusNotFound, "Not Found", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.PUT("/").To(fr.createPartition).
		Doc("create a Partition. if the given ID already exists a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(metal.Partition{}).
		Returns(http.StatusCreated, "Created", metal.Partition{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/").To(fr.updatePartition).
		Doc("updates a Partition. if the Partition was changed since this one was read, a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(metal.Partition{}).
		Returns(http.StatusOK, "OK", metal.Partition{}).
		Returns(http.StatusNotFound, "Not Found", httperrors.HTTPErrorResponse{}).
		Returns(http.StatusConflict, "Conflict", httperrors.HTTPErrorResponse{}))

	return ws
}

func (fr partitionResource) createPartition(request *restful.Request, response *restful.Response) {
	var s metal.Partition
	err := request.ReadEntity(&s)
	if checkError(request, response, "createPartition", err) {
		return
	}
	s.Created = time.Now()
	s.Changed = s.Created
	returnedPartition, err := fr.ds.CreatePartition(&s)
	if checkError(request, response, "createPartition", err) {
		return
	}
	response.WriteHeaderAndEntity(http.StatusCreated, returnedPartition)
}

func (fr partitionResource) updatePartition(request *restful.Request, response *restful.Response) {
	var newPartition metal.Partition
	err := request.ReadEntity(&newPartition)
	if checkError(request, response, "updatePartition", err) {
		return
	}

	oldPartition, err := fr.ds.FindPartition(newPartition.ID)
	if checkError(request, response, "updatePartition", err) {
		return
	}

	err = fr.ds.UpdatePartition(oldPartition, &newPartition)

	if checkError(request, response, "updatePartition", err) {
		return
	}
	response.WriteHeaderAndEntity(http.StatusOK, newPartition)
}
