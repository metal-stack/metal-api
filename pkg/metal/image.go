package metal

import (
	"time"
)

type Image struct {
	ID          string    `json:"id" description:"a unique ID" unique:"true" modelDescription:"An image that can be put on a device." rethinkdb:"id,omitempty"`
	Name        string    `json:"name" description:"the readable name" rethinkdb:"name"`
	Description string    `json:"description,omitempty" description:"a description for this image" optional:"true" rethinkdb:"description"`
	Url         string    `json:"url" description:"the url to this image" rethinkdb:"url"`
	Created     time.Time `json:"created" description:"the creation time of this image" optional:"true" readOnly:"true" rethinkdb:"created"`
	Changed     time.Time `json:"changed" description:"the last changed timestamp" optional:"true" readOnly:"true" rethinkdb:"changed"`
}

type Images []Image
type ImageMap map[string]Image

func (ii Images) ByID() ImageMap {
	res := make(ImageMap)
	for i, f := range ii {
		res[f.ID] = ii[i]
	}
	return res
}
