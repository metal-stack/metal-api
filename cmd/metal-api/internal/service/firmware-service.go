package service

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	s3server "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/s3"
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
		Param(ws.PathParameter("kind", "the kind, i.e. 'bios' or 'bmc'").DataType("string")).
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
		Param(ws.PathParameter("kind", "the kind, i.e. 'bios' or 'bmc'").DataType("string")).
		Param(ws.PathParameter("vendor", "the vendor").DataType("string")).
		Param(ws.PathParameter("board", "the board").DataType("string")).
		Param(ws.PathParameter("revision", "the firmware revision").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(v1.EmptyBody{}).
		Returns(http.StatusOK, "OK", nil).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/").
		To(admin(r.availableFirmwares)).
		Operation("availableFirmwares").
		Doc("returns all available firmwares as well as all available firmwares for a specific machine").
		Param(ws.QueryParameter("id", "restrict available firmwares to the machine identified by this query parameter").DataType("string")).
		Param(ws.QueryParameter("kind", "the kind, i.e. 'bios' or 'bmc'").DataType("string")).
		Param(ws.PathParameter("vendor", "the vendor").DataType("string")).
		Param(ws.PathParameter("board", "the board").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(v1.AvailableFirmwares{}).
		Returns(http.StatusOK, "OK", v1.AvailableFirmwares{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}

func (r firmwareResource) uploadFirmware(request *restful.Request, response *restful.Response) {
	if r.s3Client == nil {
		if checkError(request, response, utils.CurrentFuncName(), featureDisabledErr) {
			return
		}
	}

	kind, err := checkFirmwareKind(request.PathParameter("kind"))
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
		v := strings.ToLower(fru.ProductManufacturer)
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

	err = r.ensureBucket(ctx, r.s3Client.FirmwareBucket)
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	key := fmt.Sprintf("%s/%s/%s/%s", kind, vendor, board, revision)
	uploader := manager.NewUploader(r.s3Client)
	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: &r.s3Client.FirmwareBucket,
		Key:    &key,
		Body:   file,
	})
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeader(http.StatusOK)
}

func (r firmwareResource) ensureBucket(ctx context.Context, bucket string) error {
	params := &s3.CreateBucketInput{
		Bucket: &bucket,
	}
	_, err := r.s3Client.CreateBucket(ctx, params)
	if err != nil {
		var bae *types.BucketAlreadyExists
		var baoby *types.BucketAlreadyOwnedByYou
		switch {
		case errors.As(err, &bae):
		case errors.As(err, &baoby):
		default:
			return err
		}
	}
	return nil
}

func (r firmwareResource) removeFirmware(request *restful.Request, response *restful.Response) {
	if r.s3Client == nil {
		if checkError(request, response, utils.CurrentFuncName(), featureDisabledErr) {
			return
		}
	}

	kind, err := checkFirmwareKind(request.PathParameter("kind"))
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}
	vendor := strings.ToLower(request.PathParameter("vendor"))
	board := strings.ToUpper(request.PathParameter("board"))
	revision := request.PathParameter("revision")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	key := fmt.Sprintf("%s/%s/%s/%s", kind, vendor, board, revision)
	_, err = r.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &r.s3Client.FirmwareBucket,
		Key:    &key,
	})
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	response.WriteHeader(http.StatusOK)
}

func (r firmwareResource) availableFirmwares(request *restful.Request, response *restful.Response) {
	kind, err := checkFirmwareKind(request.QueryParameter("kind"))
	if checkError(request, response, utils.CurrentFuncName(), err) {
		return
	}

	resp := &v1.AvailableFirmwares{
		Revisions: make(map[string]map[string][]string),
	}
	id := request.QueryParameter("id")
	switch id {
	case "":
		vendor, board, err := getVendorAndBoard(r.ds, id)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
		rr, err := getFirmwareRevisions(r.s3Client, kind, vendor, board)
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
		rm := make(map[string][]string)
		rm[board] = rr
		resp.Revisions[vendor] = rm
	default:
		vendor := strings.ToLower(request.QueryParameter("vendor"))
		board := strings.ToUpper(request.QueryParameter("board"))

		mm, err := r.ds.ListMachines()
		if checkError(request, response, utils.CurrentFuncName(), err) {
			return
		}
		for _, m := range mm {
			fru := m.IPMI.Fru

			v := strings.ToLower(fru.ProductManufacturer)
			if vendor != "" && vendor != v {
				continue
			}
			b := strings.ToUpper(fru.BoardPartNumber)
			if board != "" && board != b {
				continue
			}

			rr, err := getFirmwareRevisions(r.s3Client, kind, v, b)
			if checkError(request, response, utils.CurrentFuncName(), err) {
				return
			}

			rm, ok := resp.Revisions[v]
			if !ok {
				rm = make(map[string][]string)
				resp.Revisions[v] = rm
			}
			rm[b] = rr
		}
	}

	err = response.WriteHeaderAndEntity(http.StatusOK, resp)
	if err != nil {
		utils.Logger(request).Sugar().Error("Failed to send response", zap.Error(err))
		return
	}
}

func getVendorAndBoard(ds *datastore.RethinkStore, machineID string) (string, string, error) {
	m, err := ds.FindMachineByID(machineID)
	if err != nil {
		return "", "", err
	}

	fru := m.IPMI.Fru
	vendor := strings.ToLower(fru.ProductManufacturer)
	board := strings.ToUpper(fru.BoardPartNumber)
	return vendor, board, nil
}

func getFirmwareRevisions(s3Client *s3server.Client, kind, vendor, board string) ([]string, error) {
	if s3Client == nil {
		return nil, featureDisabledErr
	}

	prefix := fmt.Sprintf("%s/%s/%s", kind, vendor, board)
	r4, err := s3Client.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
		Bucket: &s3Client.FirmwareBucket,
		Prefix: &prefix,
	})
	if err != nil {
		return nil, err
	}

	var rr []string
	for _, c := range r4.Contents {
		parts := strings.Split(*c.Key, "/")
		rev := parts[len(parts)-1]
		rr = append(rr, rev)
	}
	return rr, nil
}

func checkFirmwareKind(kind string) (string, error) {
	for _, k := range firmwareKinds {
		if strings.EqualFold(k, kind) {
			return k, nil
		}
	}
	return "", fmt.Errorf("unknown firmware kind %q", kind)
}
