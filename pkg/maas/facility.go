package maas

import "time"

// A Facility is an entity which describes the location where a device
// can be placed.
type Facility struct {
	ID          string    `json:"id" description:"a unique ID" unique:"true" modelDescription:"A Facility describes the location where a device is placed."`
	Name        string    `json:"name" description:"the readable name"`
	Description string    `json:"description" description:"a description for this facility" optional:"true"`
	Created     time.Time `json:"created" description:"the creation time of this facility" optional:"true"`
	Changed     time.Time `json:"changed" description:"the last changed timestamp" optional:"true"`
}
