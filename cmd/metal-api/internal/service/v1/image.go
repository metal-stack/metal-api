package v1

import (
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
)

type ImageBase struct {
	URL      *string  `json:"url" modelDescription:"an image that can be attached to a machine" description:"the url of this image" optional:"true"`
	Features []string `json:"features" description:"features of this image" optional:"true"`
}

type ImageCreateRequest struct {
	Common
	URL      string   `json:"url" description:"the url of this image"`
	Features []string `json:"features" description:"features of this image" optional:"true"`
}

type ImageUpdateRequest struct {
	Common
	ImageBase
}

type ImageResponse struct {
	Common
	ImageBase
	Timestamps
}

func NewImageResponse(img *metal.Image) *ImageResponse {
	if img == nil {
		return nil
	}
	features := []string{}
	for k, v := range img.Features {
		if v {
			features = append(features, string(k))
		}
	}
	return &ImageResponse{
		Common: Common{
			Identifiable: Identifiable{
				ID: img.ID,
			},
			Describable: Describable{
				Name:        &img.Name,
				Description: &img.Description,
			},
		},
		ImageBase: ImageBase{
			URL:      &img.URL,
			Features: features,
		},
		Timestamps: Timestamps{
			Created: img.Created,
			Changed: img.Changed,
		},
	}
}
