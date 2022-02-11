package service

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/metal-stack/metal-lib/zapup"
)

type ImageService struct {
	ds *datastore.RethinkStore
}

// NewImageService returns an image service.
func NewImageService(ds *datastore.RethinkStore) *ImageService {
	return &ImageService{
		ds: ds,
	}
}

func (ir ImageService) Get(id string) (*metal.Image, error) {
	img, err := ir.ds.GetImage(id)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func (ir ImageService) Find(id string) ([]metal.Image, error) {
	imgs, err := ir.ds.FindImages(id)
	if err != nil {
		return nil, err
	}

	return imgs, nil
}

func (ir ImageService) FindLatest(id string) (*metal.Image, error) {
	img, err := ir.ds.FindImage(id)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func (ir ImageService) List() ([]metal.Image, error) {
	imgs, err := ir.ds.ListImages()
	if err != nil {
		return nil, err
	}

	return imgs, nil
}

func (ir ImageService) Create(img *metal.Image) error {
	defaultImage(img)

	err := validateImage(img)
	if err != nil {
		return err
	}

	err = ir.ds.CreateImage(img)
	if err != nil {
		return err
	}

	return nil
}

func defaultImage(img *metal.Image) {
	if img.Classification == "" {
		img.Classification = metal.ClassificationPreview
	}

	if img.ExpirationDate.IsZero() {
		img.ExpirationDate = time.Now().Add(metal.DefaultImageExpiration)
	}

	os, v, err := utils.GetOsAndSemverFromImage(img.ID)
	if err == nil {
		if img.OS == "" {
			img.OS = os
		}
		if img.Version == "" {
			img.Version = v.String()
		}
	}
}

func validateImage(img *metal.Image) error {
	if img.ID == "" {
		return errors.New("id should not be empty")
	}

	if img.URL == "" {
		return errors.New("url should not be empty")
	}

	for f := range img.Features {
		_, err := metal.ImageFeatureTypeFrom(string(f))
		if err != nil {
			return err
		}
	}

	os, v, err := utils.GetOsAndSemverFromImage(img.ID)
	if err != nil {
		return err
	}

	if img.OS != os {
		return fmt.Errorf("os must be derived from image id: %s", img.OS)
	}

	if img.Version != v.String() {
		return fmt.Errorf("version must be derived from image id: %s", img.Version)
	}

	if img.ExpirationDate.IsZero() {
		img.ExpirationDate = time.Now().Add(metal.DefaultImageExpiration)
	}

	_, err = metal.VersionClassificationFrom(string(img.Classification))
	if err != nil {
		return err
	}

	err = checkImageURL(img.ID, img.URL)
	if err != nil {
		return err
	}

	return nil
}

func checkImageURL(id, url string) error {
	// nolint
	res, err := http.Head(url)
	if err != nil {
		return fmt.Errorf("image:%s is not accessible under:%s error:%w", id, url, err)
	}
	if res.StatusCode >= 400 {
		return fmt.Errorf("image:%s is not accessible under:%s status:%s", id, url, res.Status)
	}
	return nil
}

func (ir ImageService) Delete(id string) (*metal.Image, error) {
	img, err := ir.ds.GetImage(id)
	if err != nil {
		return nil, err
	}

	var ms metal.Machines
	err = ir.ds.SearchMachines(&datastore.MachineSearchQuery{
		AllocationImageID: &img.ID,
	}, &ms)
	if err != nil {
		return nil, err
	}

	if len(ms) > 0 {
		return nil, fmt.Errorf("image %s is in use by %d machines", img.ID, len(ms))
	}

	err = ir.ds.DeleteImage(img)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func (ir ImageService) Update(img *metal.Image) (*metal.Image, error) {
	oldImage, err := ir.ds.GetImage(img.ID)
	if err != nil {
		return nil, err
	}

	newImage := *oldImage

	if img.Name != "" {
		newImage.Name = img.Name
	}
	if img.Description != "" {
		newImage.Description = img.Description
	}
	if img.URL != "" {
		newImage.URL = img.URL
	}
	if len(img.Features) > 0 {
		newImage.Features = img.Features
	}
	if img.Classification != "" {
		newImage.Classification = img.Classification
	}
	if !img.ExpirationDate.IsZero() {
		newImage.ExpirationDate = img.ExpirationDate
	}

	err = validateImage(&newImage)
	if err != nil {
		return nil, err
	}

	err = ir.ds.UpdateImage(oldImage, &newImage)
	if err != nil {
		return nil, err
	}

	return &newImage, nil
}

// networkUsageCollector implements the prometheus collector interface.
type imageUsageCollector struct {
	ir *ImageService
}

func RegisterImageUsageCollector(ds *datastore.RethinkStore) error {
	iuc := imageUsageCollector{ir: NewImageService(ds)}

	err := prometheus.Register(iuc)
	if err != nil {
		return fmt.Errorf("failed to register prometheus: %w", err)
	}

	return nil
}

var usedImageDesc = prometheus.NewDesc(
	"metal_image_used_total",
	"The total number of machines using a image",
	[]string{"imageID", "name", "os", "classification", "created", "expirationDate", "base", "features"}, nil,
)

func (iuc imageUsageCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(iuc, ch)
}

func (iuc imageUsageCollector) Collect(ch chan<- prometheus.Metric) {
	// FIXME bad workaround to be able to run make spec
	if iuc.ir == nil || iuc.ir.ds == nil {
		return
	}
	imgs, err := iuc.ir.ds.ListImages()
	if err != nil {
		return
	}
	images := make(map[string]metal.Image)
	for _, i := range imgs {
		images[i.ID] = i
	}
	// init with 0
	usage := make(map[string]int)
	for _, i := range imgs {
		usage[i.ID] = 0
	}
	// loop over machines and count
	machines, err := iuc.ir.ds.ListMachines()
	if err != nil {
		return
	}
	for _, m := range machines {
		if m.Allocation == nil {
			continue
		}
		usage[m.Allocation.ImageID]++
	}

	for i, count := range usage {
		image := images[i]

		metric, err := prometheus.NewConstMetric(
			usedImageDesc,
			prometheus.CounterValue,
			float64(count),
			image.ID,
			image.Name,
			image.OS,
			string(image.Classification),
			fmt.Sprintf("%d", image.Created.Unix()),
			fmt.Sprintf("%d", image.ExpirationDate.Unix()),
			string(image.Base.ID),
			image.ImageFeatureString(),
		)
		if err != nil {
			zapup.MustRootLogger().Error("Failed create metric for UsedImages", zap.Error(err))
			return
		}
		ch <- metric
	}
}
