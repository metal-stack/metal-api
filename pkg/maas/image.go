package maas

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

type Image struct {
	ID          string    `json:"id" description:"a unique ID" unique:"true" modelDescription:"An image that can be put on a device."`
	Name        string    `json:"name" description:"the readable name"`
	Description string    `json:"description,omitempty" description:"a description for this image" optional:"true"`
	Url         string    `json:"url" description:"the url to this image"`
	Created     time.Time `json:"created" description:"the creation time of this image" optional:"true" readOnly:"true"`
	Changed     time.Time `json:"changed" description:"the last changed timestamp" optional:"true" readOnly:"true"`
}

var (
	DummyImages = []*Image{
		&Image{
			ID:          uuid.Must(uuid.NewV4()).String(),
			Name:        "Discovery",
			Description: "Image for initial discovery",
			Url:         "https://registry.maas/discovery/dicoverer:latest",
			Created:     time.Now(),
			Changed:     time.Now(),
		},
		&Image{
			ID:          uuid.Must(uuid.NewV4()).String(),
			Name:        "Alpine 3.8",
			Description: "Alpine 3.8",
			Url:         "https://registry.maas/alpine/alpine:3.8",
			Created:     time.Now(),
			Changed:     time.Now(),
		},
	}
)
