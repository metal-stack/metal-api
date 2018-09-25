package service

import (
	"fmt"
	"net/http"
	"time"

	"git.f-i-ts.de/ize0h88/maas-service/cmd/maas-api/internal/utils"
	"git.f-i-ts.de/ize0h88/maas-service/pkg/maas"
	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
)

var (
	// only to have something to test
	dummyFacilities = []*maas.Facility{
		&maas.Facility{
			ID:          "NBG1",
			Name:        "Nuernberg 1",
			Description: "Location number one in NBG",
			Created:     time.Now(),
			Changed:     time.Now(),
		},
		&maas.Facility{
			ID:          "NBG2",
			Name:        "Nuernberg 2",
			Description: "Location number two in NBG",
			Created:     time.Now(),
			Changed:     time.Now(),
		},
		&maas.Facility{
			ID:          "FRA",
			Name:        "Frankfurt",
			Description: "A location in frankfurt",
			Created:     time.Now(),
			Changed:     time.Now(),
		},
	}
)

// NewFacility returns a new Facility endpoint
func NewFacility() *restful.WebService {
	fr := facilityResource{
		facilities: make(map[string]*maas.Facility),
	}
	for _, df := range dummyFacilities {
		fr.facilities[df.ID] = df
	}
	return fr.webService()
}

// The facilityResource ist the entrypoint for the whole facility endpoints
type facilityResource struct {
	// dummy as long we do not have a database
	facilities map[string]*maas.Facility
}

// webService creates the webservice endpoint
func (fr facilityResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path("/facility").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"facility"}

	ws.Route(ws.GET("/").To(fr.getFacility).
		Doc("get facilities").
		Param(ws.QueryParameter("id", "identifier of the facility").DataType("string").AllowMultiple(true)).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(maas.Facility{}).
		Returns(http.StatusOK, "OK", maas.Facility{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	ws.Route(ws.DELETE("/{id}").To(fr.deleteFacility).
		Doc("delete a facility and returns the deleted entity").
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

func (fr facilityResource) getFacility(request *restful.Request, response *restful.Response) {
	request.Request.ParseForm()
	ids := request.Request.Form["id"]
	res := []*maas.Facility{}
	for _, f := range fr.facilities {
		if ids == nil {
			res = append(res, f)
		} else {
			if utils.StringInSlice(f.ID, ids) {
				res = append(res, f)
			}
		}
	}
	response.WriteEntity(res)
}

func (fr facilityResource) deleteFacility(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	f, ok := fr.facilities[id]
	if ok {
		delete(fr.facilities, id)
		response.WriteEntity(f)
	} else {
		response.WriteErrorString(http.StatusNotFound, fmt.Sprintf("Facility with id %q not found", id))
	}
}

func (fr facilityResource) createFacility(request *restful.Request, response *restful.Response) {
	var f maas.Facility
	err := request.ReadEntity(&f)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, fmt.Errorf("cannot read facility from request: %v", err))
		return
	}
	// well, check if this id already exist ... but
	// we do not have a database, so this is ok here :-)
	f.Created = time.Now()
	f.Changed = f.Created
	fr.facilities[f.ID] = &f
	response.WriteHeaderAndEntity(http.StatusCreated, f)
}

func (fr facilityResource) updateFacility(request *restful.Request, response *restful.Response) {
	var f maas.Facility
	err := request.ReadEntity(&f)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, fmt.Errorf("cannot read facility from request: %v", err))
		return
	}
	old, ok := fr.facilities[f.ID]
	if !ok {
		response.WriteErrorString(http.StatusNotFound, fmt.Sprintf("Facility with id %q not found", f.ID))
		return
	}
	if !f.Changed.Equal(old.Changed) {
		response.WriteErrorString(http.StatusConflict, fmt.Sprintf("Facility with id %q was changed in the meantime", f.ID))
		return
	}

	f.Created = old.Created
	f.Changed = time.Now()

	fr.facilities[f.ID] = &f
	response.WriteHeaderAndEntity(http.StatusOK, f)
}
