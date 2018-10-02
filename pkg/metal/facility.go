package metal

import "time"

type Facility struct {
	ID          string    `json:"id" description:"a unique ID" unique:"true" modelDescription:"A Facility describes the location where a device is placed."  rethinkdb:"id,omitempty"`
	Name        string    `json:"name" description:"the readable name" rethinkdb:"name"`
	Description string    `json:"description,omitempty" description:"a description for this facility" optional:"true" rethinkdb:"description"`
	Created     time.Time `json:"created" description:"the creation time of this facility" optional:"true" readOnly:"true" rethinkdb:"created"`
	Changed     time.Time `json:"changed" description:"the last changed timestamp" optional:"true" readOnly:"true" rethinkdb:"changed"`
}

type Facilities []Facility
type FacilityMap map[string]Facility

func (fcs Facilities) ByID() FacilityMap {
	res := make(FacilityMap)
	for i, f := range fcs {
		res[f.ID] = fcs[i]
	}
	return res
}
