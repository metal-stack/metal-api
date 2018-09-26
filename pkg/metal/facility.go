package metal

import "time"

type Facility struct {
	ID          string    `json:"id" description:"a unique ID" unique:"true" modelDescription:"A Facility describes the location where a device is placed."`
	Name        string    `json:"name" description:"the readable name"`
	Description string    `json:"description,omitempty" description:"a description for this facility" optional:"true"`
	Created     time.Time `json:"created" description:"the creation time of this facility" optional:"true" readOnly:"true"`
	Changed     time.Time `json:"changed" description:"the last changed timestamp" optional:"true" readOnly:"true"`
}

var (
	DummyFacilities = []*Facility{
		&Facility{
			ID:          "NBG1",
			Name:        "Nuernberg 1",
			Description: "Location number one in NBG",
			Created:     time.Now(),
			Changed:     time.Now(),
		},
		&Facility{
			ID:          "NBG2",
			Name:        "Nuernberg 2",
			Description: "Location number two in NBG",
			Created:     time.Now(),
			Changed:     time.Now(),
		},
		&Facility{
			ID:          "FRA",
			Name:        "Frankfurt",
			Description: "A location in frankfurt",
			Created:     time.Now(),
			Changed:     time.Now(),
		},
	}
)
