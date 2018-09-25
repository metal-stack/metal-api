package service

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"git.f-i-ts.de/ize0h88/maas-service/cmd/maas-api/internal/utils"
	"git.f-i-ts.de/ize0h88/maas-service/pkg/maas"
	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
)

var (
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

type sizeResource struct {
	// dummy as long we do not have a database
	sizes map[string]*maas.Size
}

func NewSize() *restful.WebService {
	sr := sizeResource{
		sizes: make(map[string]*maas.Size),
	}
	addDummySizes(sr.sizes)
	return sr.webService()
}

func addDummySizes(sizes map[string]*maas.Size) {
	for _, ds := range dummySizes {
		sizes[ds.ID] = ds
	}
}

func deleteSizes(sizes map[string]*maas.Size) {
	for _, size := range sizes {
		delete(sizes, size.ID)
	}
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
	res := getSize(sr, ids)
	response.WriteEntity(res)
}

func getSize(sr sizeResource, want []string) []*maas.Size {
	all := sr.sizes
	var res []*maas.Size

	if len(want) == 0 {
		for _, size := range all {
			res = append(res, size)
		}
	} else {
		for _, size := range all {
			if utils.StringInSlice(size.ID, want) {
				res = append(res, size)
			}
		}
	}
	return res
}

func (sr sizeResource) deleteSize(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	size, err := deleteSize(sr, id)
	if err != nil {
		response.WriteError(http.StatusNotFound, err)
	} else {
		response.WriteEntity(size)
	}
}

func deleteSize(sr sizeResource, id string) (*maas.Size, error) {
	all := sr.sizes
	size, ok := all[id]
	if ok {
		delete(all, id)
	} else {
		return nil, errors.New(fmt.Sprintf("size with id %q not found", id))
	}
	return size, nil
}

func (sr sizeResource) createSize(request *restful.Request, response *restful.Response) {
	var s maas.Size
	err := request.ReadEntity(&s)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, fmt.Errorf("cannot read size from request: %v", err))
		return
	}
	s.Created = time.Now()
	s.Changed = s.Created
	err = createSize(sr, &s)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, fmt.Errorf("cannot create size: %v", err))
	} else {
		response.WriteHeaderAndEntity(http.StatusCreated, s)
	}
}

func createSize(sr sizeResource, s *maas.Size) error {
	all := sr.sizes
	// well, check if this id already exist ... but
	// we do not have a database, so this is ok here :-)
	all[s.ID] = s
	return nil
}

func (sr sizeResource) updateSize(request *restful.Request, response *restful.Response) {
	var s maas.Size
	err := request.ReadEntity(&s)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, fmt.Errorf("cannot read size from request: %v", err))
		return
	}
	returnCode, err := updateSize(sr, &s)

	if err != nil {
		response.WriteError(returnCode, err)
	} else {
		response.WriteHeaderAndEntity(http.StatusOK, s)
	}
}

func updateSize(sr sizeResource, s *maas.Size) (int, error) {
	all := sr.sizes

	old, ok := all[s.ID]
	if !ok {
		return http.StatusNotFound, errors.New(fmt.Sprintf("size with id %q not found", s.ID))
	}
	if !s.Changed.Equal(old.Changed) {
		return http.StatusConflict, errors.New(fmt.Sprintf("size with id %q was changed in the meantime", s.ID))
	}

	s.Created = old.Created
	s.Changed = time.Now()

	all[s.ID] = s
	return http.StatusOK, nil
}
