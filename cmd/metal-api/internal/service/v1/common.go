package v1

import (
	"time"
)

// emptyBody is useful because with go-restful you cannot define an insert / update endpoint
// without specifying a payload for reading. it would immediately intercept the request and
// return 406: Not Acceptable to the client.
type EmptyBody struct{}

type Identifiable struct {
	ID string `json:"id" description:"the unique ID of this entity" required:"true"`
}

type Describable struct {
	Name        *string `json:"name,omitempty" description:"a readable name for this entity" optional:"true"`
	Description *string `json:"description,omitempty" description:"a description for this entity" optional:"true"`
}

type Common struct {
	Identifiable
	Describable
}

type Timestamps struct {
	Created time.Time `json:"created" description:"the creation time of this entity" readOnly:"true" optional:"true"`
	Changed time.Time `json:"changed" description:"the last changed timestamp of this entity" readOnly:"true" optional:"true"`
}
