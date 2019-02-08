package metal

import (
	"fmt"

	humanize "github.com/dustin/go-humanize"
)

var (
	// UnknownSize is the size to use, when someone requires a size we do not know.
	UnknownSize = &Size{
		Base: Base{
			ID:   "unknown",
			Name: "unknown",
		},
	}
)

// A Size represents a supported machine size.
type Size struct {
	Base
	Constraints []Constraint `json:"constraints" modelDescription:"A Size describes our supported t-shirt sizes." description:"a list of constraints that defines this size" rethinkdb:"constraints"`
}

// A Constraint describes the hardware constraints for a given size. At the moment we only
// consider the cpu cores and the memory.
type Constraint struct {
	MinCores  int    `json:"mincores" rethinkdb:"mincores" description:"the minimal number of cores"`
	MaxCores  int    `json:"maxcores" rethinkdb:"maxcores" description:"the maximal number of cores"`
	MinMemory uint64 `json:"minmemory" rethinkdb:"minmemory" description:"the minimal amount of memory"`
	MaxMemory uint64 `json:"maxmemory" rethinkdb:"maxmemory" description:"the maximal amount of memory"`
}

// Sizes is a list of sizes.
type Sizes []Size

// SizeMap is an indexed map of sizes.
type SizeMap map[string]Size

// ByID creates a map of sizes with the id as the index.
func (sz Sizes) ByID() SizeMap {
	res := make(SizeMap)
	for i, f := range sz {
		res[f.ID] = sz[i]
	}
	return res
}

// FromHardware searches a Size for given hardware specs. It will search
// for a size where at least one constraint matches the given hardware.
func (sz Sizes) FromHardware(hardware MachineHardware) (*Size, error) {
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
			break
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
