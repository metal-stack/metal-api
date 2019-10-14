package v1

import (
	"time"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
)

type ImageBase struct {
	URL      *string   `json:"url" modelDescription:"an image that can be attached to a machine" description:"the url of this image" optional:"true"`
	Features []string  `json:"features" description:"features of this image" optional:"true"`
	ValidTo  time.Time `json:"validto" description:"date to which it is allowed to allocate machines from" optional:"false"`
}

type ImageCreateRequest struct {
	Common
	URL      string    `json:"url" description:"the url of this image"`
	Features []string  `json:"features" description:"features of this image" optional:"true"`
	ValidTo  time.Time `json:"validto" description:"date to which it is allowed to allocate machines from" optional:"false"`
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
		if v == true {
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
			ValidTo:  img.ValidTo,
		},
		Timestamps: Timestamps{
			Created: img.Created,
			Changed: img.Changed,
		},
	}
}
