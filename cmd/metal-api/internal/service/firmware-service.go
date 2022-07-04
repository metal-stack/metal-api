package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	s3server "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/s3client"

	"github.com/metal-stack/metal-lib/httperrors"
	"go.uber.org/zap"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
)

var featureDisabledErr = errors.New("this feature is currently disabled")

type firmwareResource struct {
	webResource
	s3Client *s3server.Client
}

// NewFirmware returns a webservice for firmware specific endpoints.
func NewFirmware(log *zap.SugaredLogger, ds *datastore.RethinkStore, s3Client *s3server.Client) (*restful.WebService, error) {
	r := firmwareResource{
		webResource: webResource{
			log: log,
			ds:  ds,
		},
		s3Client: s3Client,
	}
	return r.webService(), nil
}

// webService creates the webservice endpoint
func (r *firmwareResource) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path(BasePath + "v1/firmware").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"firmware"}

	ws.Route(ws.PUT("/{kind}/{vendor}/{board}/{revision}").
		To(admin(r.uploadFirmware)).
		Operation("uploadFirmware").
		Doc("upload given firmware").
		Param(ws.PathParameter("kind", "the firmware kind [bios|bmc]").DataType("string")).
		Param(ws.PathParameter("vendor", "the vendor").DataType("string")).
		Param(ws.PathParameter("board", "the board").DataType("string")).
		Param(ws.PathParameter("revision", "the firmware revision").DataType("string")).
		Param(ws.FormParameter("file", "the firmware file").DataType("file")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Consumes("multipart/form-data").
		Returns(http.StatusOK, "OK", nil).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.DELETE("/{kind}/{vendor}/{board}/{revision}").
		To(admin(r.removeFirmware)).
		Operation("removeFirmware").
		Doc("remove given firmware").
		Param(ws.PathParameter("kind", "the firmware kind [bios|bmc]").DataType("string")).
		Param(ws.PathParameter("vendor", "the vendor").DataType("string")).
		Param(ws.PathParameter("board", "the board").DataType("string")).
		Param(ws.PathParameter("revision", "the firmware revision").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.EmptyBody{}).
		Returns(http.StatusOK, "OK", nil).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/").
		To(admin(r.listFirmwares)).
		Operation("listFirmwares").
		Doc("returns all firmwares (for a specific machine)").
		Param(ws.QueryParameter("machine-id", "restrict firmwares to the given machine").DataType("string")).
		Param(ws.QueryParameter("kind", "the firmware kind [bios|bmc]").DataType("string")).
		Param(ws.QueryParameter("vendor", "the vendor").DataType("string")).
		Param(ws.QueryParameter("board", "the board").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.FirmwaresResponse{}).
		Returns(http.StatusOK, "OK", v1.FirmwaresResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r *firmwareResource) uploadFirmware(request *restful.Request, response *restful.Response) {
	if r.s3Client == nil {
		r.SendError(response, httperrors.NewHTTPError(http.StatusInternalServerError, featureDisabledErr))
		return
	}

	kind, err := toFirmwareKind(request.PathParameter("kind"))
	if err != nil {
		r.SendError(response, httperrors.BadRequest(err))
		return
	}

	vendor := strings.ToLower(request.PathParameter("vendor"))
	board := strings.ToUpper(request.PathParameter("board"))
	revision := request.PathParameter("revision")

	// check that at least one machine matches kind, vendor and board
	validReq := false
	mm, err := r.ds.ListMachines()
	if err != nil {
		r.SendError(response, DefaultError(err))
		return
	}

	for _, m := range mm {
		fru := m.IPMI.Fru
		if strings.EqualFold(fru.BoardMfg, vendor) && strings.EqualFold(fru.BoardPartNumber, board) {
			validReq = true
			break
		}
	}
	if !validReq {
		r.SendError(response, httperrors.BadRequest(fmt.Errorf("there is no machine of vendor %s with board %s", vendor, board)))
		return
	}

	file, _, err := request.Request.FormFile("file")
	if err != nil {
		r.SendError(response, httperrors.InternalServerError(err))
		return
	}

	key := fmt.Sprintf("%s/%s/%s/%s", kind, vendor, board, revision)
	_, err = r.s3Client.PutObjectWithContext(context.Background(), &s3.PutObjectInput{
		Bucket: &r.s3Client.FirmwareBucket,
		Key:    &key,
		Body:   file,
	})
	if err != nil {
		r.SendError(response, httperrors.InternalServerError(err))
		return
	}

	response.WriteHeader(http.StatusOK)
}

func (r *firmwareResource) removeFirmware(request *restful.Request, response *restful.Response) {
	if r.s3Client == nil {
		r.SendError(response, httperrors.NewHTTPError(http.StatusInternalServerError, featureDisabledErr))
		return
	}

	kind, err := toFirmwareKind(request.PathParameter("kind"))
	if err != nil {
		r.SendError(response, httperrors.BadRequest(err))
		return
	}

	vendor := strings.ToLower(request.PathParameter("vendor"))
	board := strings.ToUpper(request.PathParameter("board"))
	revision := request.PathParameter("revision")

	key := fmt.Sprintf("%s/%s/%s/%s", kind, vendor, board, revision)
	_, err = r.s3Client.DeleteObjectWithContext(context.Background(), &s3.DeleteObjectInput{
		Bucket: &r.s3Client.FirmwareBucket,
		Key:    &key,
	})
	if err != nil {
		r.SendError(response, httperrors.InternalServerError(err))
		return
	}

	response.WriteHeader(http.StatusOK)
}

func (r *firmwareResource) listFirmwares(request *restful.Request, response *restful.Response) {
	if r.s3Client == nil {
		r.SendError(response, httperrors.NewHTTPError(http.StatusInternalServerError, featureDisabledErr))
		return
	}

	kind := guessFirmwareKind(request.QueryParameter("kind"))
	var kk []metal.FirmwareKind
	switch kind {
	case "":
		kk = append(kk, metal.FirmwareBMC)
		kk = append(kk, metal.FirmwareBIOS)
	default:
		kk = append(kk, kind)
	}

	rr := make(map[string]map[string]map[string][]string)
	for i := range kk {
		k := kk[i]
		rr[k] = make(map[string]map[string][]string)
		machineID := request.QueryParameter("machine-id")
		switch machineID {
		case "":
			vendor := request.QueryParameter("vendor")
			board := request.QueryParameter("board")

			err := r.s3Client.ListObjectsPagesWithContext(context.Background(), &s3.ListObjectsInput{
				Bucket: &r.s3Client.FirmwareBucket,
				Prefix: &k,
			}, func(page *s3.ListObjectsOutput, last bool) bool {
				for _, p := range page.Contents {
					insertRevisions(*p.Key, rr[k], vendor, board)
				}
				return true
			})
			if err != nil {
				r.SendError(response, httperrors.InternalServerError(err))
				return
			}
		default:
			_, f, err := getFirmware(r.ds, machineID)
			if err != nil {
				r.SendError(response, DefaultError(err))
				return
			}

			bb := make(map[string][]string)
			switch k {
			case metal.FirmwareBIOS:
				bb[f.Board] = []string{f.BiosVersion}
			case metal.FirmwareBMC:
				bb[f.Board] = []string{f.BmcVersion}
			}
			rr[k][f.Vendor] = bb
		}
	}

	r.Send(response, http.StatusOK, mapToFirmwareResponse(rr))
}

func getFirmware(ds *datastore.RethinkStore, machineID string) (*metal.Machine, *v1.Firmware, error) {
	m, err := ds.FindMachineByID(machineID)
	if err != nil {
		return nil, nil, err
	}

	fru := m.IPMI.Fru
	vendor := strings.ToLower(fru.BoardMfg)
	board := strings.ToUpper(fru.BoardPartNumber)

	return m, &v1.Firmware{
		Vendor:      vendor,
		Board:       board,
		BmcVersion:  m.IPMI.BMCVersion,
		BiosVersion: m.BIOS.Version,
	}, nil
}

func getFirmwareRevisions(s3Client *s3server.Client, kind, vendor, board string) ([]string, error) {
	r4, err := s3Client.ListObjectsWithContext(context.Background(), &s3.ListObjectsInput{
		Bucket: &s3Client.FirmwareBucket,
		Prefix: &kind,
	})
	if err != nil {
		return nil, err
	}

	var rr []string
	for _, c := range r4.Contents {
		f, ok := filterRevision(*c.Key, vendor, board)
		if ok {
			rr = append(rr, f.Revision)
		}
	}
	return rr, nil
}

func insertRevisions(path string, revisions map[string]map[string][]string, vendor, board string) {
	f, ok := filterRevision(path, vendor, board)
	if !ok {
		return
	}
	boardMap, ok := revisions[f.Vendor]
	if !ok {
		boardMap = make(map[string][]string)
		revisions[f.Vendor] = boardMap
	}
	for _, rev := range boardMap[f.Board] {
		if rev == f.Revision {
			return
		}
	}
	boardMap[f.Board] = append(boardMap[f.Board], f.Revision)
}

func filterRevision(path, vendor, board string) (*v1.Firmware, bool) {
	parts := strings.Split(path, "/")
	if len(parts) != 4 {
		return nil, false
	}
	v := parts[1]
	if vendor != "" && !strings.EqualFold(v, vendor) {
		return nil, false
	}
	b := parts[2]
	if board != "" && !strings.EqualFold(b, board) {
		return nil, false
	}
	return &v1.Firmware{
		Vendor:   v,
		Board:    b,
		Revision: parts[3],
	}, true
}

func guessFirmwareKind(kind string) string {
	if kind == "" {
		return ""
	}
	fk, err := toFirmwareKind(kind)
	if err != nil {
		return ""
	}
	return fk
}

func toFirmwareKind(kind string) (string, error) {
	for _, k := range metal.FirmwareKinds {
		if strings.EqualFold(k, kind) {
			return k, nil
		}
	}
	return "", fmt.Errorf("unknown firmware kind %q", kind)
}

func mapToFirmwareResponse(m map[string]map[string]map[string][]string) *v1.FirmwaresResponse {
	resp := &v1.FirmwaresResponse{
		Revisions: make(map[string]v1.VendorRevisions),
	}
	for k, vv := range m {
		resp.Revisions[k] = v1.VendorRevisions{
			VendorRevisions: make(map[string]v1.BoardRevisions),
		}
		for v, bb := range vv {
			resp.Revisions[k].VendorRevisions[v] = v1.BoardRevisions{
				BoardRevisions: make(map[string][]string),
			}
			for b, rr := range bb {
				resp.Revisions[k].VendorRevisions[v].BoardRevisions[b] = rr
			}
		}
	}
	return resp
}
