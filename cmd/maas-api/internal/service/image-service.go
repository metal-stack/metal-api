package service

import (
	"fmt"
	"net/http"
	"time"

	"git.f-i-ts.de/cloud-native/maas/maas-service/cmd/maas-api/internal/utils"
	"git.f-i-ts.de/cloud-native/maas/maas-service/pkg/maas"
	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	uuid "github.com/satori/go.uuid"
)

var (
	// only to have something to test
	dummyImages = []*maas.Image{
		&maas.Image{
			ID:          uuid.Must(uuid.NewV4()).String(),
			Name:        "Discovery",
			Description: "Image for initial discovery",
			Url:         "https://registry.maas/discovery/dicoverer:latest",
			Created:     time.Now(),
			Changed:     time.Now(),
		},
		&maas.Image{
			ID:          uuid.Must(uuid.NewV4()).String(),
			Name:        "Alpine 3.8",
			Description: "Alpine 3.8",
			Url:         "https://registry.maas/alpine/alpine:3.8",
			Created:     time.Now(),
			Changed:     time.Now(),
		},
	}
)

func NewImage() *restful.WebService {
	ir := imageResource{
		images: make(map[string]*maas.Image),
	}
	for _, di := range dummyImages {
		ir.images[di.ID] = di
	}
	return ir.webService()
}

type imageResource struct {
	// dummy as long we do not have a database
	images map[string]*maas.Image
}

func (ir imageResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path("/image").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"image"}

	ws.Route(ws.GET("/").To(ir.getImage).
		Doc("get images").
		Param(ws.QueryParameter("id", "identifier of the image").AllowMultiple(true).DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]maas.Image{}).
		Returns(http.StatusOK, "OK", []maas.Image{}))

	ws.Route(ws.DELETE("/{id}").To(ir.deleteImage).
		Doc("deletes an image and returns the deleted entity").
		Param(ws.PathParameter("id", "identifier of the image").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(maas.Image{}).
		Returns(http.StatusOK, "OK", maas.Image{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	ws.Route(ws.PUT("/").To(ir.createImage).
		Doc("create an image. if the given ID already exists a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(maas.Image{}).
		Returns(http.StatusCreated, "Created", maas.Image{}).
		Returns(http.StatusConflict, "Conflict", nil))

	ws.Route(ws.POST("/").To(ir.updateImage).
		Doc("updates an image. if the image was changed since this one was read, a conflict is returned").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(maas.Image{}).
		Returns(http.StatusOK, "OK", maas.Image{}).
		Returns(http.StatusNotFound, "Not Found", nil).
		Returns(http.StatusConflict, "Conflict", nil))

	return ws
}

func (ir imageResource) getImage(request *restful.Request, response *restful.Response) {
	request.Request.ParseForm()
	ids := request.Request.Form["id"]
	res := []*maas.Image{}
	for _, i := range ir.images {
		if len(ids) == 0 {
			res = append(res, i)
		} else {
			if utils.StringInSlice(i.ID, ids) {
				res = append(res, i)
			}
		}
	}
	response.WriteEntity(res)
}

func (ir imageResource) deleteImage(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	i, ok := ir.images[id]
	if ok {
		delete(ir.images, id)
		response.WriteEntity(i)
	} else {
		response.WriteErrorString(http.StatusNotFound, fmt.Sprintf("image with id %q not found", id))
	}
}

func (ir imageResource) createImage(request *restful.Request, response *restful.Response) {
	var i maas.Image
	err := request.ReadEntity(&i)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, fmt.Errorf("cannot read image irom request: %v", err))
		return
	}
	// well, check if this id already exist ... but
	// we do not have a database, so this is ok here :-)
	i.Created = time.Now()
	i.Changed = i.Created
	ir.images[i.ID] = &i
	response.WriteHeaderAndEntity(http.StatusCreated, i)
}

func (ir imageResource) updateImage(request *restful.Request, response *restful.Response) {
	var i maas.Image
	err := request.ReadEntity(&i)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, fmt.Errorf("cannot read image irom request: %v", err))
		return
	}
	old, ok := ir.images[i.ID]
	if !ok {
		response.WriteErrorString(http.StatusNotFound, fmt.Sprintf("image with id %q not found", i.ID))
		return
	}
	if !i.Changed.Equal(old.Changed) {
		response.WriteErrorString(http.StatusConflict, fmt.Sprintf("image with id %q was changed in the meantime", i.ID))
		return
	}

	i.Created = old.Created
	i.Changed = time.Now()

	ir.images[i.ID] = &i
	response.WriteHeaderAndEntity(http.StatusOK, i)
}
