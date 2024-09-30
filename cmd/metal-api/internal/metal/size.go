package metal

import (
	"errors"
	"fmt"
	"path/filepath"

	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	"github.com/samber/lo"
)

// A Size represents a supported machine size.
type Size struct {
	Base
	Constraints []Constraint      `rethinkdb:"constraints" json:"constraints"`
	Labels      map[string]string `rethinkdb:"labels" json:"labels"`
}

// ConstraintType ...
type ConstraintType string

// come constraint types
const (
	CoreConstraint    ConstraintType = "cores"
	MemoryConstraint  ConstraintType = "memory"
	StorageConstraint ConstraintType = "storage"
	GPUConstraint     ConstraintType = "gpu"
)

var allConstraintTypes = []ConstraintType{CoreConstraint, MemoryConstraint, StorageConstraint, GPUConstraint}

// A Constraint describes the hardware constraints for a given size.
type Constraint struct {
	Type       ConstraintType `rethinkdb:"type" json:"type"`
	Min        uint64         `rethinkdb:"min" json:"min"`
	Max        uint64         `rethinkdb:"max" json:"max"`
	Identifier string         `rethinkdb:"identifier" json:"identifier" description:"glob of the identifier of this type"`
}

func countCPU(cpu MetalCPU) (model string, count uint64) {
	return cpu.Model, uint64(cpu.Cores)
}

func countGPU(gpu MetalGPU) (model string, count uint64) {
	return gpu.Model, 1
}

func countDisk(disk BlockDevice) (model string, count uint64) {
	return disk.Name, disk.Size
}

func countMemory(size uint64) (model string, count uint64) {
	return "", size
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
		cores, _ := capacityOf(c.Identifier, hw.MetalCPUs, countCPU)
		res = c.inRange(cores)
	case MemoryConstraint:
		res = c.inRange(hw.Memory)
	case StorageConstraint:
		capacity, _ := capacityOf(c.Identifier, hw.Disks, countDisk)
		res = c.inRange(capacity)
	case GPUConstraint:
		count, _ := capacityOf(c.Identifier, hw.MetalGPUs, countGPU)
		res = c.inRange(count)
	}
	return res
}

// matches returns true if all provided disks and later GPUs are covered with at least one constraint.
// With this we ensure that hardware matches exhaustive against the constraints.
func (hw *MachineHardware) matches(constraints []Constraint, constraintType ConstraintType) bool {
	filtered := lo.Filter(constraints, func(c Constraint, _ int) bool { return c.Type == constraintType })

	switch constraintType {
	case StorageConstraint:
		return exhaustiveMatch(filtered, hw.Disks, countDisk)
	case GPUConstraint:
		return exhaustiveMatch(filtered, hw.MetalGPUs, countGPU)
	case CoreConstraint:
		return exhaustiveMatch(filtered, hw.MetalCPUs, countCPU)
	case MemoryConstraint:
		return exhaustiveMatch(filtered, []uint64{hw.Memory}, countMemory)
	default:
		return false
	}
}

// FromHardware searches a Size for given hardware specs. It will search
// for a size where the constraints matches the given hardware.
func (sz Sizes) FromHardware(hardware MachineHardware) (*Size, error) {
	var (
		matchedSizes []Size
	)

nextsize:
	for _, s := range sz {
		for _, c := range s.Constraints {
			if !c.matches(hardware) {
				continue nextsize
			}
		}

		for _, ct := range allConstraintTypes {
			if !hardware.matches(s.Constraints, ct) {
				continue nextsize
			}
		}

		matchedSizes = append(matchedSizes, s)
	}

	switch len(matchedSizes) {
	case 0:
		return nil, NotFound("no size found for hardware (%s)", hardware.ReadableSpec())
	case 1:
		return &matchedSizes[0], nil
	default:
		return nil, fmt.Errorf("%d sizes found for hardware (%s)", len(matchedSizes), hardware.ReadableSpec())
	}
}

func (s *Size) overlaps(so *Size) bool {
	if len(lo.FromPtr(so).Constraints) == 0 || len(lo.FromPtr(s).Constraints) == 0 {
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

	for t, destConstraints := range destTypes {
		constraints, ok := srcTypes[t]
		if !ok {
			return false
		}
		for _, sc := range destConstraints {
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

func (c *Constraint) validate() error {
	if c.Max < c.Min {
		return fmt.Errorf("max is smaller than min")
	}

	if _, err := filepath.Match(c.Identifier, ""); err != nil {
		return fmt.Errorf("identifier is malformed: %w", err)
	}

	switch t := c.Type; t {
	case GPUConstraint:
		if c.Identifier == "" {
			return fmt.Errorf("for gpu constraints an identifier is required")
		}
	case MemoryConstraint:
		if c.Identifier != "" {
			return fmt.Errorf("for memory constraints an identifier is not allowed")
		}
	case CoreConstraint, StorageConstraint:
	}

	return nil
}

// Validate a size, returns error if a invalid size is passed
func (s *Size) Validate(partitions PartitionMap, projects map[string]*mdmv1.Project) error {
	var (
		errs       []error
		typeCounts = map[ConstraintType]uint{}
	)

	for i, c := range s.Constraints {
		typeCounts[c.Type]++

		err := c.validate()
		if err != nil {
			errs = append(errs, fmt.Errorf("constraint at index %d is invalid: %w", i, err))
		}

		switch t := c.Type; t {
		case GPUConstraint, StorageConstraint:
		case MemoryConstraint, CoreConstraint:
			if typeCounts[t] > 1 {
				errs = append(errs, fmt.Errorf("constraint at index %d is invalid: type duplicates are not allowed for type %q", i, t))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("size %q is invalid: %w", s.ID, errors.Join(errs...))
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
