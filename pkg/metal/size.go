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

var (
	DummySizes = []*Size{
		&Size{
			ID:          "t1.small.x86",
			Name:        "t1.small.x86",
			Description: "The Tiny But Mighty!",
			Created:     time.Now(),
			Changed:     time.Now(),
		},
		&Size{
			ID:          "m2.xlarge.x86",
			Name:        "m2.xlarge.x86",
			Description: "The Latest and Greatest",
			Created:     time.Now(),
			Changed:     time.Now(),
		},
		&Size{
			ID:          "c1.large.arm",
			Name:        "c1.large.arm",
			Description: "The Armv8 Beast!",
			Created:     time.Now(),
			Changed:     time.Now(),
		},
	}
)
