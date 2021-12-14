package service

import (
	"errors"
	"net/http"
	"time"

	"github.com/metal-stack/security"

	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/metal-lib/zapup"
	"go.uber.org/zap"

	mdm "github.com/metal-stack/masterdata-api/pkg/client"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/utils"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
)

type eventResource struct {
	webResource
	mdc        mdm.Client
	userGetter security.UserGetter
}

// NewMachine returns a webservice for machine specific endpoints.
func NewEvent(
	ds *datastore.RethinkStore,
	mdc mdm.Client,
	userGetter security.UserGetter,
) (*restful.WebService, error) {
	r := eventResource{
		webResource: webResource{
			ds: ds,
		},
		mdc:        mdc,
		userGetter: userGetter,
	}

	return r.webService(), nil
}

// webService creates the webservice endpoint
func (r eventResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path(BasePath + "v1/event").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"event"}

	ws.Route(ws.GET("/machine/{id}").
		To(viewer(r.getProvisioningEventContainer)).
		Operation("getProvisioningEventContainer").
		Doc("get the current machine provisioning event container").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.MachineRecentProvisioningEvents{}).
		Returns(http.StatusOK, "OK", v1.MachineRecentProvisioningEvents{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.POST("/machine/{id}").
		To(editor(r.addProvisioningEvent)).
		Operation("addProvisioningEvent").
		Doc("adds a machine provisioning event").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.MachineProvisioningEvent{}).
		Writes(v1.MachineRecentProvisioningEvents{}).
		Returns(http.StatusOK, "OK", v1.MachineRecentProvisioningEvents{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r eventResource) getProvisioningEventContainer(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	// check for existence of the machine
	_, err := r.ds.FindMachineByID(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	ec, err := r.ds.FindProvisioningEventContainer(id)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	err = response.WriteHeaderAndEntity(http.StatusOK, v1.NewMachineRecentProvisioningEvents(ec))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}

func (r eventResource) addProvisioningEvent(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	m, err := r.ds.FindMachineByID(id)
	if err != nil && !metal.IsNotFound(err) {
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
	}

	// an event can actually create an empty machine. This enables us to also catch the very first PXE Booting event
	// in a machine lifecycle
	if m == nil {
		m = &metal.Machine{
			Base: metal.Base{
				ID: id,
			},
		}
		err = r.ds.CreateMachine(m)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
	}

	var requestPayload v1.MachineProvisioningEvent
	err = request.ReadEntity(&requestPayload)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	ok := metal.AllProvisioningEventTypes[metal.ProvisioningEventType(requestPayload.Event)]
	if !ok {
		if checkError(request, response, utils.CurrentFuncName(), errors.New("unknown provisioning event")) {
			return
		}
	}

	event := metal.ProvisioningEvent{
		Time:    time.Now(),
		Event:   metal.ProvisioningEventType(requestPayload.Event),
		Message: requestPayload.Message,
	}
	ec, err := r.ds.ProvisioningEventForMachine(id, event)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	err = response.WriteHeaderAndEntity(http.StatusOK, v1.NewMachineRecentProvisioningEvents(ec))
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response", zap.Error(err))
		return
	}
}
