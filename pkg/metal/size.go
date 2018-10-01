package metal

import "time"

type Size struct {
	ID          string `json:"id" description:"a unique ID" unique:"true" modelDescription:"An image that can be put on a device."`
	Name        string `json:"name" description:"the readable name"`
	Description string `json:"description,omitempty" description:"a description for this image" optional:"true"`
	// Constraints []*Constraint `json:"constraints" description:"a list of constraints that defines this size" optional:"true"`
	Created time.Time `json:"created" description:"the creation time of this image" optional:"true" readOnly:"true"`
	Changed time.Time `json:"changed" description:"the last changed timestamp" optional:"true" readOnly:"true"`
}
