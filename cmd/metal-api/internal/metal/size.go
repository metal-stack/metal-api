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
func (c *Constraint) Matches(hw MachineHardware) (ConstraintMatchingLog, bool) {
	logentryFmt := fmt.Sprintf("%%d >= %d && %%d <= %d", c.Min, c.Max)
	cml := ConstraintMatchingLog{Constraint: *c, Log: fmt.Sprintf("no constraint matching %q", c.Type)}
	res := false
	switch c.Type {
	case CoreConstraint:
		res = uint64(hw.CPUCores) >= c.Min && uint64(hw.CPUCores) <= c.Max
		cml.Log = fmt.Sprintf(logentryFmt, hw.CPUCores, hw.CPUCores)
	case MemoryConstraint:
		res = hw.Memory >= c.Min && hw.Memory <= c.Max
		cml.Log = fmt.Sprintf(logentryFmt, hw.Memory, hw.Memory)
	case StorageConstraint:
		res = hw.DiskCapacity() >= c.Min && hw.DiskCapacity() <= c.Max
		cml.Log = fmt.Sprintf(logentryFmt, hw.DiskCapacity(), hw.DiskCapacity())
	}
	cml.Match = res
	return cml, res
}

// FromHardware searches a Size for given hardware specs. It will search
// for a size where the constraints matches the given hardware.
func (sz Sizes) FromHardware(hardware MachineHardware) (*Size, []*SizeMatchingLog, error) {
	var found []Size
	matchlog := make([]*SizeMatchingLog, 0)
nextsize:
	for _, s := range sz {
		ml := &SizeMatchingLog{Name: s.ID, Match: false}
		matchlog = append(matchlog, ml)
		for _, c := range s.Constraints {
			lg, match := c.Matches(hardware)
			ml.Constraints = append(ml.Constraints, lg)
			if !match {
				continue nextsize
			}
		}
		ml.Match = true
		found = append(found, s)
	}

	if len(found) == 0 {
		return nil, matchlog, fmt.Errorf("no size found for hardware (%s)", hardware.ReadableSpec())
	}
	if len(found) > 1 {
		return nil, matchlog, fmt.Errorf("%d sizes found for hardware (%s)", len(found), hardware.ReadableSpec())
	}
	return &found[0], matchlog, nil
}

// A ConstraintMatchingLog is used do return a log message to the caller
// beside the contraint itself.
type ConstraintMatchingLog struct {
	Constraint Constraint `json:"constraint,omitempty"`
	Match      bool       `json:"match"`
	Log        string     `json:"log,omitempty"`
}

// A SizeMatchingLog returns information about a list of constraints.
type SizeMatchingLog struct {
	Name        string                  `json:"name"`
	Log         string                  `json:"log,omitempty"`
	Match       bool                    `json:"match"`
	Constraints []ConstraintMatchingLog `json:"constraints,omitempty"`
}
