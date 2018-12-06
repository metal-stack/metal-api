package metal

import "time"

type Site struct {
	ID          string    `json:"id" description:"a unique ID" unique:"true" modelDescription:"A Facility describes the location where a device is placed."  rethinkdb:"id,omitempty"`
	Name        string    `json:"name" description:"the readable name" rethinkdb:"name"`
	Description string    `json:"description" description:"a small description" rethinkdb:"description"`
	Created     time.Time `json:"created" description:"the creation time of this image" optional:"true" readOnly:"true" rethinkdb:"created"`
	Changed     time.Time `json:"changed" description:"the last changed timestamp" optional:"true" readOnly:"true" rethinkdb:"changed"`
}

type Sites []Site
type SiteMap map[string]Site

func (sz Sites) ByID() SiteMap {
	res := make(SiteMap)
	for i, s := range sz {
		res[s.ID] = sz[i]
	}
	return res
}
