package metal

import (
	"fmt"
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

// ConstraintType ...
type ConstraintType string

// come constraint types
const (
	CoreConstraint    ConstraintType = "cores"
	MemoryConstraint  ConstraintType = "memory"
	StorageConstraint ConstraintType = "storage"
)

// A Constraint describes the hardware constraints for a given size. At the moment we only
// consider the cpu cores and the memory.
type Constraint struct {
	Type ConstraintType `json:"type" rethinkdb:"type" description:"the type of constraint"`
	Min  uint64         `json:"min" rethinkdb:"min" description:"the minimal value"`
	Max  uint64         `json:"max" rethinkdb:"max" description:"the maximal value"`
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

// Matches returns true if the given machine hardware is inside the min/max values of the
// constraint.
func (c *Constraint) Matches(hw MachineHardware) bool {
	switch c.Type {
	case CoreConstraint:
		return uint64(hw.CPUCores) >= c.Min && uint64(hw.CPUCores) <= c.Max
	case MemoryConstraint:
		return hw.Memory >= c.Min && hw.Memory <= c.Max
	case StorageConstraint:
		return hw.DiskCapacity() >= c.Min && hw.DiskCapacity() <= c.Max
	}
	return false
}

// FromHardware searches a Size for given hardware specs. It will search
// for a size where the constraints matches the given hardware.
func (sz Sizes) FromHardware(hardware MachineHardware) (*Size, error) {
	var found []Size
nextsize:
	for _, s := range sz {
		for _, c := range s.Constraints {
			if !c.Matches(hardware) {
				continue nextsize
			}
		}
		found = append(found, s)
	}

	if len(found) == 0 {
		return nil, fmt.Errorf("no size found for hardware (%s)", hardware.ReadableSpec())
	}
	if len(found) > 1 {
		return nil, fmt.Errorf("%d sizes found for hardware (%s)", len(found), hardware.ReadableSpec())
	}
	return &found[0], nil
}
