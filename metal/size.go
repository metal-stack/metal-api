package metal

import (
	"fmt"
	"time"

	humanize "github.com/dustin/go-humanize"
)

var (
	UnknownSize = &Size{
		ID:   "unknown",
		Name: "unknown",
	}
)

type Size struct {
	ID          string       `json:"id" description:"a unique ID" unique:"true" modelDescription:"An image that can be put on a device." rethinkdb:"id,omitempty"`
	Name        string       `json:"name" description:"the readable name" rethinkdb:"name"`
	Description string       `json:"description,omitempty" description:"a description for this image" optional:"true" rethinkdb:"description"`
	Constraints []Constraint `json:"constraints" description:"a list of constraints that defines this size" rethinkdb:"constraints"`
	Created     time.Time    `json:"created" description:"the creation time of this image" optional:"true" readOnly:"true" rethinkdb:"created"`
	Changed     time.Time    `json:"changed" description:"the last changed timestamp" optional:"true" readOnly:"true" rethinkdb:"changed"`
}

// A Constraint describes the hardware constraints for a given size. At the moment we only
// consider the cpu cores and the memory.
type Constraint struct {
	MinCores  int    `json:"mincores" rethinkdb:"mincores" description:"the minimal number of cores"`
	MaxCores  int    `json:"maxcores" rethinkdb:"maxcores" description:"the maximal number of cores"`
	MinMemory uint64 `json:"minmemory" rethinkdb:"minmemory" description:"the minimal amount of memory"`
	MaxMemory uint64 `json:"maxmemory" rethinkdb:"maxmemory" description:"the maximal amount of memory"`
}

type Sizes []Size
type SizeMap map[string]Size

func (sz Sizes) ByID() SizeMap {
	res := make(SizeMap)
	for i, f := range sz {
		res[f.ID] = sz[i]
	}
	return res
}

/*
	return &metal.Size{
		ID:   "t1-small-x86",
		Name: "t1-small-x86",
	}, nil

	/*return &metal.Size{
		ID:   "unknown",
		Name: "unknown",
	}, nil
*/

func (sz Sizes) FromHardware(hardware DeviceHardware) (*Size, error) {
	// this could be done by a DB-query too, but we will not have that many
	// sizes i think. so a go implementation of this mapping would be ok

	var found []Size
	for _, s := range sz {
		for _, c := range s.Constraints {
			if hardware.CPUCores < c.MinCores || hardware.CPUCores > c.MaxCores {
				continue
			}
			if hardware.Memory < c.MinMemory || hardware.Memory > c.MaxMemory {
				continue
			}
			found = append(found, s)
		}
	}

	if len(found) == 0 {
		return nil, fmt.Errorf("no size found for %d cores and %s bytes", hardware.CPUCores, humanize.Bytes(hardware.Memory))
	}
	if len(found) > 1 {
		return nil, fmt.Errorf("%d sizes found for %d cores and %s bytes", len(found), hardware.CPUCores, humanize.Bytes(hardware.Memory))
	}
	return &found[0], nil
}
