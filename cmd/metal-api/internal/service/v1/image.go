package v1

import (
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
)

type ImageBase struct {
	URL *string `json:"url" description:"the url of this image" optional:"true"`
}

type ImageCreateRequest struct {
	Describeable
	URL string `json:"url" description:"the url of this image"`
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
	return &ImageDetailResponse{
		ImageListResponse: *NewImageListResponse(img),
		Timestamps: Timestamps{
			Created: img.Created,
			Changed: img.Changed,
		},
	}
}

func NewImageListResponse(img *metal.Image) *ImageListResponse {
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
			URL: &img.URL,
		},
	}
}
