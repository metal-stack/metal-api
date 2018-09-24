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
	dummySizes = []*maas.Size{
		&maas.Size{
			ID:          "t1.small.x86",
			Name:        "t1.small.x86",
			Description: "The Tiny But Mighty!",
			Created:     time.Now(),
			Changed:     time.Now(),
		},
		&maas.Size{
			ID:          "m2.xlarge.x86",
			Name:        "m2.xlarge.x86",
			Description: "The Latest and Greatest",
			Created:     time.Now(),
			Changed:     time.Now(),
		},
		&maas.Size{
			ID:          "c1.large.arm",
			Name:        "c1.large.arm",
			Description: "The Armv8 Beast!",
			Created:     time.Now(),
			Changed:     time.Now(),
		},
	}
)

func NewSize() *restful.WebService {
	sr := sizeResource{
		sizes: make(map[string]*maas.Size),
	}
	for _, ds := range dummySizes {
		sr.sizes[ds.ID] = ds
	}
	return sr.webService()
}

type sizeResource struct {
	// dummy as long we do not have a database
	sizes map[string]*maas.Size
}

func (sr sizeResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path("/size").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"size"}

	ws.Route(ws.GET("/").To(sr.getSize).
		Doc("get sizes").
		Param(ws.QueryParameter("id", "identifier of the size").AllowMultiple(true).DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]maas.Size{}).
		Returns(http.StatusOK, "OK", []maas.Size{}))

	ws.Route(ws.DELETE("/{id}").To(sr.deleteSize).
		Doc("deletes an size and returns the deleted entity").
		Param(ws.PathParameter("id", "identifier of the size").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(maas.Size{}).
		Returns(http.StatusOK, "OK", maas.Size{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	ws.Route(ws.PUT("/").To(sr.createSize).
		Doc("create a size. if the given ID already exists a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(maas.Size{}).
		Returns(http.StatusCreated, "Created", maas.Size{}).
		Returns(http.StatusConflict, "Conflict", nil))

	ws.Route(ws.POST("/").To(sr.updateSize).
		Doc("updates a size. if the size was changed since this one was read, a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(maas.Size{}).
		Returns(http.StatusOK, "OK", maas.Size{}).
		Returns(http.StatusNotFound, "Not Found", nil).
		Returns(http.StatusConflict, "Conflict", nil))

	return ws
}

func (sr sizeResource) getSize(request *restful.Request, response *restful.Response) {
	request.Request.ParseForm()
	ids := request.Request.Form["id"]
	res := []*maas.Size{}
	for _, s := range sr.sizes {
		if len(ids) == 0 {
			res = append(res, s)
		} else {
			if utils.StringInSlice(s.ID, ids) {
				res = append(res, s)
			}
		}
	}
	response.WriteEntity(res)
}

func (sr sizeResource) deleteSize(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	s, ok := sr.sizes[id]
	if ok {
		delete(sr.sizes, id)
		response.WriteEntity(s)
	} else {
		response.WriteErrorString(http.StatusNotFound, fmt.Sprintf("size with id %q not found", id))
	}
}

func (sr sizeResource) createSize(request *restful.Request, response *restful.Response) {
	var s maas.Size
	err := request.ReadEntity(&s)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, fmt.Errorf("cannot read size srom request: %v", err))
		return
	}
	// well, check if this id already exist ... but
	// we do not have a database, so this is ok here :-)
	s.Created = time.Now()
	s.Changed = s.Created
	sr.sizes[s.ID] = &s
	response.WriteHeaderAndEntity(http.StatusCreated, s)
}

func (sr sizeResource) updateSize(request *restful.Request, response *restful.Response) {
	var s maas.Size
	err := request.ReadEntity(&s)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, fmt.Errorf("cannot read size from request: %v", err))
		return
	}
	old, ok := sr.sizes[s.ID]
	if !ok {
		response.WriteErrorString(http.StatusNotFound, fmt.Sprintf("size with id %q not found", s.ID))
		return
	}
	if !s.Changed.Equal(old.Changed) {
		response.WriteErrorString(http.StatusConflict, fmt.Sprintf("size with id %q was changed in the meantime", s.ID))
		return
	}

	s.Created = old.Created
	s.Changed = time.Now()

	sr.sizes[s.ID] = &s
	response.WriteHeaderAndEntity(http.StatusOK, s)
}
