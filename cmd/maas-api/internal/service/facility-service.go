package service

import (
	"fmt"
	"net/http"
	"time"

	"git.f-i-ts.de/cloud-native/maas/maas-service/cmd/maas-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/maas/maas-service/pkg/maas"
	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
)

type facilityResource struct {
	ds datastore.Datastore
}

func NewFacility(ds datastore.Datastore) *restful.WebService {
	fr := facilityResource{
		ds: ds,
	}
	return fr.webService()
}

func (fr facilityResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path("/facility").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"facility"}

	ws.Route(ws.GET("/{id}").To(fr.findFacility).
		Doc("get facility by id").
		Param(ws.PathParameter("id", "identifier of the facility").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(maas.Facility{}).
		Returns(http.StatusOK, "OK", maas.Facility{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	ws.Route(ws.GET("/").To(fr.listFacilities).
		Doc("get all facilities").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]maas.Facility{}).
		Returns(http.StatusOK, "OK", []maas.Facility{}))

	ws.Route(ws.DELETE("/{id}").To(fr.deleteFacility).
		Doc("deletes a facility and returns the deleted entity").
		Param(ws.PathParameter("id", "identifier of the facility").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(maas.Facility{}).
		Returns(http.StatusOK, "OK", maas.Facility{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	ws.Route(ws.PUT("/").To(fr.createFacility).
		Doc("create a facility. if the given ID already exists a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(maas.Facility{}).
		Returns(http.StatusCreated, "Created", maas.Facility{}).
		Returns(http.StatusConflict, "Conflict", nil))

	ws.Route(ws.POST("/").To(fr.updateFacility).
		Doc("updates a facility. if the facility was changed since this one was read, a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(maas.Facility{}).
		Returns(http.StatusOK, "OK", maas.Facility{}).
		Returns(http.StatusNotFound, "Not Found", nil).
		Returns(http.StatusConflict, "Conflict", nil))

	return ws
}

func (fr facilityResource) findFacility(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	facility, err := fr.ds.FindFacility(id)
	if err != nil {
		response.WriteError(http.StatusNotFound, err)
	}
	response.WriteEntity(facility)
}

func (fr facilityResource) listFacilities(request *restful.Request, response *restful.Response) {
	res := fr.ds.ListFacilities()
	response.WriteEntity(res)
}

func (fr facilityResource) deleteFacility(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	facility, err := fr.ds.DeleteFacility(id)
	if err != nil {
		response.WriteError(http.StatusNotFound, err)
	} else {
		response.WriteEntity(facility)
	}
}

func (fr facilityResource) createFacility(request *restful.Request, response *restful.Response) {
	var s maas.Facility
	err := request.ReadEntity(&s)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, fmt.Errorf("cannot read facility from request: %v", err))
		return
	}
	s.Created = time.Now()
	s.Changed = s.Created
	err = fr.ds.CreateFacility(&s)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, fmt.Errorf("cannot create facility: %v", err))
	} else {
		response.WriteHeaderAndEntity(http.StatusCreated, s)
	}
}

func (fr facilityResource) updateFacility(request *restful.Request, response *restful.Response) {
	var newFacility maas.Facility
	err := request.ReadEntity(&newFacility)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, fmt.Errorf("cannot read facility from request: %v", err))
		return
	}

	oldFacility, err := fr.ds.FindFacility(newFacility.ID)
	if err != nil {
		response.WriteError(http.StatusNotFound, err)
		return
	}

	err = fr.ds.UpdateFacility(oldFacility, &newFacility)

	if err != nil {
		response.WriteError(http.StatusConflict, err)
	}
	response.WriteHeaderAndEntity(http.StatusOK, newFacility)
}
