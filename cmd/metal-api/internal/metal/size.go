package metal

import (
	"fmt"
	"path/filepath"
	"slices"

	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	"github.com/samber/lo"
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
	GPUConstraint     ConstraintType = "gpu"
)

// A Constraint describes the hardware constraints for a given size.
type Constraint struct {
	Type       ConstraintType `rethinkdb:"type" json:"type"`
	Min        uint64         `rethinkdb:"min" json:"min"`
	Max        uint64         `rethinkdb:"max" json:"max"`
	Identifier string         `rethinkdb:"identifier" json:"identifier" description:"glob of the identifier of this type"`
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

func (c *Constraint) inRange(value uint64) bool {
	return value >= c.Min && value <= c.Max
}

// matches returns true if the given machine hardware is inside the min/max values of the
// constraint.
func (c *Constraint) matches(hw MachineHardware) bool {
	res := false
	switch c.Type {
	case CoreConstraint:
		res = c.inRange(uint64(hw.CPUCores))
	case MemoryConstraint:
		res = c.inRange(hw.Memory)
	case StorageConstraint:
		capacity, _ := diskCapacityOf(c.Identifier, hw.Disks)
		res = c.inRange(capacity)
	case GPUConstraint:
		for model, count := range hw.GPUModels() {
			idMatches, err := filepath.Match(c.Identifier, model)
			if err != nil {
				return false
			}
			res = c.inRange(count) && idMatches
			if res {
				break
			}
		}

	}
	return res
}

// matches returns true if all provided disks and later GPUs are covered with at least one constraint.
// With this we ensure that hardware matches exhaustive against the constraints.
func (hw *MachineHardware) matches(constraints []Constraint, constraintType ConstraintType) bool {
	filtered := lo.Filter(constraints, func(c Constraint, _ int) bool { return c.Type == constraintType })
	if len(filtered) == 0 {
		return true
	}

	switch constraintType {
	case StorageConstraint:
		unmatchedDisks := slices.Clone(hw.Disks)
		for _, c := range filtered {
			capacity, listOfDisks := diskCapacityOf(c.Identifier, hw.Disks)

			match := c.inRange(capacity)
			if !match {
				continue
			}

			unmatchedDisks, _ = lo.Difference(unmatchedDisks, listOfDisks)
		}
		return len(unmatchedDisks) == 0
	case GPUConstraint:
		// FIXME implement
		return true
	case CoreConstraint, MemoryConstraint:
		// Noop because we do not have different CPU types or Memory types
		return true
	default:
		return true
	}
}

// FromHardware searches a Size for given hardware specs. It will search
// for a size where the constraints matches the given hardware.
func (sz Sizes) FromHardware(hardware MachineHardware) (*Size, error) {
	var (
		foundByConstraint []Size
		foundByHardware   []Size
	)
nextsize:
	for _, s := range sz {
		for _, c := range s.Constraints {
			match := c.matches(hardware)
			if !match {
				continue nextsize
			}
		}
		foundByConstraint = append(foundByConstraint, s)
	}

	for _, sz := range foundByConstraint {
		match := hardware.matches(sz.Constraints, StorageConstraint)
		if match {
			foundByHardware = append(foundByHardware, sz)
		}
	}

	if len(foundByHardware) == 0 {
		return nil, NotFound("no size found for hardware (%s)", hardware.ReadableSpec())
	}
	if len(foundByHardware) > 1 {
		return nil, fmt.Errorf("%d sizes found for hardware (%s)", len(foundByHardware), hardware.ReadableSpec())
	}
	return &foundByHardware[0], nil
}

func (s *Size) overlaps(so *Size) bool {
	if len(lo.FromPtr(so).Constraints) == 0 {
		return false
	}
	srcTypes := lo.GroupBy(s.Constraints, func(item Constraint) ConstraintType {
		return item.Type
	})
	destTypes := lo.GroupBy(so.Constraints, func(item Constraint) ConstraintType {
		return item.Type
	})
	for t, srcConstraints := range srcTypes {
		constraints, ok := destTypes[t]
		if !ok {
			return false
		}
		for _, sc := range srcConstraints {
			for _, c := range constraints {
				if !c.overlaps(sc) {
					return false
				}
			}
		}
	}

	return true
}

// overlaps is proven correct, requires that constraint are validated before that max is not smaller than min
func (c *Constraint) overlaps(other Constraint) bool {
	if c.Type != other.Type {
		return false
	}

	if c.Identifier != other.Identifier {
		return false
	}

	if c.Min > other.Max {
		return false
	}

	if c.Max < other.Min {
		return false
	}
	return true
}

// Validate a size, returns error if a invalid size is passed
func (s *Size) Validate(partitions PartitionMap, projects map[string]*mdmv1.Project) error {
	constraintTypes := map[ConstraintType]bool{}
	for _, c := range s.Constraints {
		if c.Max < c.Min {
			return fmt.Errorf("size:%q type:%q max:%d is smaller than min:%d", s.ID, c.Type, c.Max, c.Min)
		}

		_, ok := constraintTypes[c.Type]
		if ok {
			return fmt.Errorf("size:%q type:%q min:%d max:%d has duplicate constraint type", s.ID, c.Type, c.Min, c.Max)
		}

		// Ensure GPU Constraints always have identifier specified
		if c.Type == GPUConstraint && c.Identifier == "" {
			return fmt.Errorf("size:%q type:%q min:%d max:%d is a gpu size but has no identifier specified", s.ID, c.Type, c.Min, c.Max)
		}

		if _, err := filepath.Match(c.Identifier, ""); err != nil {
			return fmt.Errorf("size:%q type:%q min:%d max:%d identifier:%q identifier is malformed:%w", s.ID, c.Type, c.Min, c.Max, c.Identifier, err)
		}

		constraintTypes[c.Type] = true
	}

	if err := s.Reservations.Validate(partitions, projects); err != nil {
		return fmt.Errorf("invalid size reservation: %w", err)
	}

	return nil
}

// Overlaps returns nil if Size does not overlap with any other size, otherwise returns overlapping Size
func (s *Size) Overlaps(ss *Sizes) *Size {
	for _, so := range *ss {
		so := so
		if s.ID == so.ID {
			continue
		}
		if s.overlaps(&so) {
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
