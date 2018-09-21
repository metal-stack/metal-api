package maas

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

type Image struct {
	ID          uuid.UUID `json:"id" description:"a unique ID" unique:"true" modelDescription:"An image that can be put on a device."`
	Name        string    `json:"name" description:"the readable name"`
	Description string    `json:"description" description:"a description for this image" optional:"true"`
	Url         string    `json:"url" description:"the url to this image"`
	Created     time.Time `json:"created" description:"the creation time of this image" optional:"true"`
	Changed     time.Time `json:"changed" description:"the last changed timestamp" optional:"true"`
}
