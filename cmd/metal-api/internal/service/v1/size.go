package v1

import (
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

type SizeConstraint struct {
	Type metal.ConstraintType `json:"type" modelDescription:"a machine matches to a size in order to make them easier to categorize" enum:"cores|memory|storage" description:"the type of the constraint"`
	Min  uint64               `json:"min" description:"the minimum value of the constraint"`
	Max  uint64               `json:"max" description:"the maximum value of the constraint"`
}

type SizeCreateRequest struct {
	Common
	SizeConstraints []SizeConstraint `json:"constraints" description:"a list of constraints that defines this size"`
}

type SizeUpdateRequest struct {
	Common
	SizeConstraints *[]SizeConstraint `json:"constraints" description:"a list of constraints that defines this size" optional:"true"`
}

type SizeResponse struct {
	Common
	SizeConstraints []SizeConstraint `json:"constraints" description:"a list of constraints that defines this size"`
	Timestamps
}

type SizeSuggestRequest struct {
	MachineID string `json:"machineID" description:"machineID to retrieve size suggestion for"`
}

type SizeConstraintMatchingLog struct {
	Constraint SizeConstraint `json:"constraint" description:"the size constraint to which this log relates to"`
	Match      bool           `json:"match" description:"indicates whether the constraint matched or not"`
	Log        string         `json:"log" description:"a string represention of the matching condition"`
}

type SizeMatchingLog struct {
	Name        string                      `json:"name"`
	Log         string                      `json:"log"`
	Match       bool                        `json:"match"`
	Constraints []SizeConstraintMatchingLog `json:"constraints"`
}

func NewSizeMatchingLog(m *metal.SizeMatchingLog) *SizeMatchingLog {
	constraints := []SizeConstraintMatchingLog{}
	for i := range m.Constraints {
		constraint := SizeConstraintMatchingLog{
			Constraint: SizeConstraint{
				Type: m.Constraints[i].Constraint.Type,
				Min:  m.Constraints[i].Constraint.Min,
				Max:  m.Constraints[i].Constraint.Max,
			},
			Match: m.Constraints[i].Match,
			Log:   m.Constraints[i].Log,
		}
		constraints = append(constraints, constraint)
	}
	return &SizeMatchingLog{
		Name:        m.Name,
		Match:       m.Match,
		Log:         m.Log,
		Constraints: constraints,
	}
}

func NewSizeResponse(s *metal.Size) *SizeResponse {
	if s == nil {
		return nil
	}

	constraints := []SizeConstraint{}
	for _, c := range s.Constraints {
		constraint := SizeConstraint{
			Type: c.Type,
			Min:  c.Min,
			Max:  c.Max,
		}
		constraints = append(constraints, constraint)
	}

	return &SizeResponse{
		Common: Common{
			Identifiable: Identifiable{
				ID: s.ID,
			},
			Describable: Describable{
				Name:        &s.Name,
				Description: &s.Description,
			},
		},
		SizeConstraints: constraints,
		Timestamps: Timestamps{
			Created: s.Created,
			Changed: s.Changed,
		},
	}
}
