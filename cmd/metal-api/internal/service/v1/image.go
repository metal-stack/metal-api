package v1

import (
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
)

type ImageBase struct {
	URL      *string  `json:"url" modelDescription:"an image that can be attached to a machine" description:"the url of this image" optional:"true"`
	Features []string `json:"features" description:"features of this image" enum:"machine|firewall" optional:"true"`
}

type ImageCreateRequest struct {
	Common
	URL      string   `json:"url" description:"the url of this image"`
	Features []string `json:"features" description:"features of this image" enum:"machine|firewall" optional:"true"`
}

type ImageUpdateRequest struct {
	Common
	ImageBase
}

type ImageListResponse struct {
	Common
	ImageBase
}

type ImageDetailResponse struct {
	ImageListResponse
	Timestamps
}

func NewImageDetailResponse(img *metal.Image) *ImageDetailResponse {
	if img == nil {
		return nil
	}
	return &ImageDetailResponse{
		ImageListResponse: *NewImageListResponse(img),
		Timestamps: Timestamps{
			Created: img.Created,
			Changed: img.Changed,
		},
	}
}

func NewImageListResponse(img *metal.Image) *ImageListResponse {
	if img == nil {
		return nil
	}
	features := []string{}
	for k, v := range img.Features {
		if v == true {
			features = append(features, string(k))
		}
	}
	return &ImageListResponse{
		Common: Common{
			Identifiable: Identifiable{
				ID: img.ID,
			},
			Describeable: Describeable{
				Name:        &img.Name,
				Description: &img.Description,
			},
		},
		ImageBase: ImageBase{
			URL:      &img.URL,
			Features: features,
		},
	}
}
