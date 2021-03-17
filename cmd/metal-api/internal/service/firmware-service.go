package service

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	s3server "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/s3client"
	"net/http"
	"strings"

	"github.com/pkg/errors"

	"github.com/metal-stack/metal-lib/httperrors"
	"go.uber.org/zap"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/utils"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
)

var featureDisabledErr = errors.New("this feature is currently disabled")

type firmwareResource struct {
	webResource
	s3Client *s3server.Client
}

type FirmwareKind = string

const (
	bios FirmwareKind = "bios"
	bmc  FirmwareKind = "bmc"
)

var firmwareKinds = []string{
	bios,
	bmc,
}

// NewFirmware returns a webservice for firmware specific endpoints.
func NewFirmware(ds *datastore.RethinkStore, s3Client *s3server.Client) (*restful.WebService, error) {
	r := firmwareResource{
		webResource: webResource{
			ds: ds,
		},
		s3Client: s3Client,
	}
	return r.webService(), nil
}

// webService creates the webservice endpoint
func (r firmwareResource) webService() *restful.WebService {
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
		Param(ws.QueryParameter("id", "restrict firmwares to the machine identified by this query parameter").DataType("string")).
		Param(ws.QueryParameter("kind", "the firmware kind [bios|bmc]").DataType("string")).
		Param(ws.QueryParameter("vendor", "the vendor").DataType("string")).
		Param(ws.QueryParameter("board", "the board").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]v1.Firmwares{}).
		Returns(http.StatusOK, "OK", []v1.Firmwares{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r firmwareResource) uploadFirmware(request *restful.Request, response *restful.Response) {
	if r.s3Client == nil && checkError(request, response, utils.CurrentFuncName(), featureDisabledErr) {
		return
	}

	kind, err := strictCheckFirmwareKind(request.PathParameter("kind"))
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	vendor := strings.ToLower(request.PathParameter("vendor"))
	board := strings.ToUpper(request.PathParameter("board"))
	revision := request.PathParameter("revision")

	// check that at least one machine matches kind, vendor and board
	validReq := false
	mm, err := r.ds.ListMachines()
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	for _, m := range mm {
		fru := m.IPMI.Fru
		v := strings.ToLower(fru.BoardMfg)
		b := strings.ToUpper(fru.BoardPartNumber)
		if v == vendor && b == board {
			validReq = true
			break
		}
	}
	if !validReq {
		if checkError(request, response, utils.CurrentFuncName(), fmt.Errorf("there is no machine of vendor %s with board %s", vendor, board)) {
			return
		}
	}

	file, _, err := request.Request.FormFile("file")
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	key := fmt.Sprintf("%s/%s/%s/%s", kind, vendor, board, revision)
	uploader := s3manager.NewUploader(r.s3Client.Session)
	_, err = uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket: &r.s3Client.FirmwareBucket,
		Key:    &key,
		Body:   file,
	})
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeader(http.StatusOK)
}

func (r firmwareResource) removeFirmware(request *restful.Request, response *restful.Response) {
	if r.s3Client == nil && checkError(request, response, utils.CurrentFuncName(), featureDisabledErr) {
		return
	}

	kind, err := strictCheckFirmwareKind(request.PathParameter("kind"))
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	vendor := strings.ToLower(request.PathParameter("vendor"))
	board := strings.ToUpper(request.PathParameter("board"))
	revision := request.PathParameter("revision")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	key := fmt.Sprintf("%s/%s/%s/%s", kind, vendor, board, revision)
	_, err = r.s3Client.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: &r.s3Client.FirmwareBucket,
		Key:    &key,
	})
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeader(http.StatusOK)
}

func (r firmwareResource) listFirmwares(request *restful.Request, response *restful.Response) {
	if r.s3Client == nil && checkError(request, response, utils.CurrentFuncName(), featureDisabledErr) {
		return
	}

	kind, err := checkFirmwareKind(request.QueryParameter("kind"))
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	var kk []FirmwareKind
	switch kind {
	case "":
		kk = append(kk, bmc)
		kk = append(kk, bios)
	default:
		kk = append(kk, kind)
	}

	var resp []v1.Firmwares
	for i := range kk {
		k := kk[i]
		ff := v1.Firmwares{
			Kind: k,
		}
		id := request.QueryParameter("id")
		switch id {
		case "":
			vendor := strings.ToLower(request.QueryParameter("vendor"))
			board := strings.ToUpper(request.QueryParameter("board"))

			vendorBoards := make(map[string]map[string][]string)

			err := r.s3Client.ListObjectsPagesWithContext(context.Background(), &s3.ListObjectsInput{
				Bucket: &r.s3Client.FirmwareBucket,
				Prefix: &k,
			}, func(page *s3.ListObjectsOutput, last bool) bool {
				// Add the objects to the channel for each page
				for _, c := range page.Contents {
					parts := strings.Split(*c.Key, "/")
					if len(parts) != 4 {
						continue
					}
					v := parts[1]
					if vendor != "" && vendor != v {
						continue
					}
					b := parts[2]
					if board != "" && board != b {
						continue
					}
					boardMap, ok := vendorBoards[v]
					if !ok {
						boardMap = make(map[string][]string)
						vendorBoards[v] = boardMap
					}
					rev := parts[3]
					boardMap[b] = append(boardMap[b], rev)
				}
				return true
			})
			if checkError(request, response, utils.CurrentFuncName(), err) {
				return
			}

			for v, bb := range vendorBoards {
				for b, rr := range bb {
					bf := v1.BoardFirmwares{
						Board:     b,
						Revisions: rr,
					}
					found := false
					for i, vv := range ff.VendorFirmwares {
						if v == vv.Vendor {
							vv.BoardFirmwares = append(vv.BoardFirmwares, bf)
							ff.VendorFirmwares[i] = vv
							found = true
							break
						}
					}
					if !found {
						ff.VendorFirmwares = append(ff.VendorFirmwares, v1.VendorFirmwares{
							Vendor:         v,
							BoardFirmwares: []v1.BoardFirmwares{bf},
						})
					}
				}
			}
		default:
			f, err := getFirmware(r.ds, id)
			if checkError(request, response, utils.CurrentFuncName(), err) {
				return
			}
			rr, err := getFirmwareRevisions(r.s3Client, k, f.Vendor, f.Board)
			if checkError(request, response, utils.CurrentFuncName(), err) {
				return
			}
			ff.VendorFirmwares = []v1.VendorFirmwares{
				{
					Vendor: f.Vendor,
					BoardFirmwares: []v1.BoardFirmwares{
						{
							Board:     f.Board,
							Revisions: rr,
						},
					},
				},
			}
		}

		resp = append(resp, ff)
	}

	err = response.WriteHeaderAndEntity(http.StatusOK, resp)
	if err != nil {
		utils.Logger(request).Sugar().Error("Failed to send response", zap.Error(err))
		return
	}
}

func getFirmware(ds *datastore.RethinkStore, machineID string) (*v1.Firmware, error) {
	m, err := ds.FindMachineByID(machineID)
	if err != nil {
		return nil, err
	}

	fru := m.IPMI.Fru
	vendor := strings.ToLower(fru.BoardMfg)
	board := strings.ToUpper(fru.BoardPartNumber)

	return &v1.Firmware{
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
		parts := strings.Split(*c.Key, "/")
		if len(parts) != 4 {
			continue
		}
		v := parts[1]
		if vendor != "" && v != vendor {
			continue
		}
		b := parts[2]
		if board != "" && b != board {
			continue
		}
		rev := parts[3]
		rr = append(rr, rev)
	}
	return rr, nil
}

func checkFirmwareKind(kind string) (string, error) {
	if kind == "" {
		return "", nil
	}
	return strictCheckFirmwareKind(kind)
}

func strictCheckFirmwareKind(kind string) (string, error) {
	for _, k := range firmwareKinds {
		if strings.EqualFold(k, kind) {
			return k, nil
		}
	}
	return "", fmt.Errorf("unknown firmware kind %q", kind)
}
