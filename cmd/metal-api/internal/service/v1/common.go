package v1

import (
	"time"
)

type Identifiable struct {
	ID string `json:"id" description:"the unique ID of this entity" required:"True"`
}

type Describeable struct {
	Name        *string `json:"name,omitempty" description:"a readable name for this entity" optional:"true"`
	Description *string `json:"description,omitempty" description:"a description for this entity" optional:"true"`
}

type Common struct {
	Identifiable
	Describeable
}

type Timestamps struct {
	Created time.Time `json:"created" description:"the creation time of this entity" readOnly:"true"`
	Changed time.Time `json:"changed" description:"the last changed timestamp of this entity" readOnly:"true"`
}
