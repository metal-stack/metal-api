package metal

import (
	"fmt"
	"slices"

	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
)

// A Size represents a supported machine size.
type Size struct {
	Base
	Constraints  []Constraint      `rethinkdb:"constraints" json:"constraints"`
	Reservations Reservations      `rethinkdb:"reservations" json:"reservations"`
	Labels       map[string]string `rethinkdb:"labels" json:"labels"`
}

// Reservation defines a reservation of a size for machine allocations
type Reservation struct {
	Amount       int      `rethinkdb:"amount" json:"amount"`
	Description  string   `rethinkdb:"description" json:"description"`
	ProjectID    string   `rethinkdb:"projectid" json:"projectid"`
	PartitionIDs []string `rethinkdb:"partitionids" json:"partitionids"`
}

type Reservations []Reservation

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
	Type ConstraintType `rethinkdb:"type" json:"type"`
	Min  uint64         `rethinkdb:"min" json:"min"`
	Max  uint64         `rethinkdb:"max" json:"max"`
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

// UnknownSize is the size to use, when someone requires a size we do not know.
func UnknownSize() *Size {
	return &Size{
		Base: Base{
			ID:   "unknown",
			Name: "unknown",
		},
	}
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
	var matchedlog *SizeMatchingLog
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
		matchedlog = ml
		found = append(found, s)
	}

	if len(found) == 0 {
		return nil, matchlog, NotFound("no size found for hardware (%s)", hardware.ReadableSpec())
	}
	if len(found) > 1 {
		return nil, matchlog, fmt.Errorf("%d sizes found for hardware (%s)", len(found), hardware.ReadableSpec())
	}
	return &found[0], []*SizeMatchingLog{matchedlog}, nil
}

func (s *Size) overlaps(so *Size) bool {
	if len(so.Constraints) == 0 {
		return false
	}
	for _, c := range s.Constraints {
		for _, co := range so.Constraints {
			if c.Type == co.Type && ((c.Min < co.Min && c.Max < co.Min) || (c.Min > co.Min && c.Min > co.Max)) {
				return false
			}
		}
	}
	return true
}

// Validate a size, returns error if a invalid size is passed
func (s *Size) Validate(partitions PartitionMap, projects map[string]*mdmv1.Project) error {
	for _, c := range s.Constraints {
		if c.Max < c.Min {
			return fmt.Errorf("size:%q type:%q max:%d is smaller than min:%d", s.ID, c.Type, c.Max, c.Min)
		}
	}

	if err := s.Reservations.Validate(partitions, projects); err != nil {
		return fmt.Errorf("invalid size reservation: %w", err)
	}

	return nil
}

// Overlaps returns nil if Size does not overlap with any other size, otherwise returns overlapping Size
func (s *Size) Overlaps(ss *Sizes) *Size {
	for i := range *ss {
		so := (*ss)[i]
		if s.Name != so.Name && s.overlaps(&so) {
			return &so
		}
	}
	return nil
}

func (rs *Reservations) ForPartition(partitionID string) Reservations {
	if rs == nil {
		return nil
	}

	var result Reservations
	for _, r := range *rs {
		r := r
		if slices.Contains(r.PartitionIDs, partitionID) {
			result = append(result, r)
		}
	}

	return result
}

func (rs *Reservations) ForProject(projectID string) Reservations {
	if rs == nil {
		return nil
	}

	var result Reservations
	for _, r := range *rs {
		r := r
		if r.ProjectID == projectID {
			result = append(result, r)
		}
	}

	return result
}

func (rs *Reservations) Validate(partitions PartitionMap, projects map[string]*mdmv1.Project) error {
	if rs == nil {
		return nil
	}

	for _, r := range *rs {
		if r.Amount <= 0 {
			return fmt.Errorf("amount must be a positive integer")
		}

		if len(r.PartitionIDs) == 0 {
			return fmt.Errorf("at least one partition id must be specified")
		}
		ids := map[string]bool{}
		for _, partition := range r.PartitionIDs {
			ids[partition] = true
			if _, ok := partitions[partition]; !ok {
				return fmt.Errorf("partition must exist before creating a size reservation")
			}
		}
		if len(ids) != len(r.PartitionIDs) {
			return fmt.Errorf("partitions must not contain duplicates")
		}

		if r.ProjectID == "" {
			return fmt.Errorf("project id must be specified")
		}
		if _, ok := projects[r.ProjectID]; !ok {
			return fmt.Errorf("project must exist before creating a size reservation")
		}
	}

	return nil
}

// A ConstraintMatchingLog is used do return a log message to the caller
// beside the constraint itself.
type ConstraintMatchingLog struct {
	Constraint Constraint
	Match      bool
	Log        string
}

// A SizeMatchingLog returns information about a list of constraints.
type SizeMatchingLog struct {
	Name        string
	Log         string
	Match       bool
	Constraints []ConstraintMatchingLog
}
