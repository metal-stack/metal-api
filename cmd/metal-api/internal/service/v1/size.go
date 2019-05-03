package v1

import (
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
)

type SizeConstraint struct {
	Type metal.ConstraintType `json:"type" description:"the type of constraint"`
	Min  uint64               `json:"min" description:"the minimal value"`
	Max  uint64               `json:"max" description:"the maximal value"`
}

type SizeCreateRequest struct {
	Describeable
	SizeConstraints []SizeConstraint `json:"constraints" description:"a list of constraints that defines this size"`
}

type SizeUpdateRequest struct {
	Common
	SizeConstraints *[]SizeConstraint `json:"constraints" description:"a list of constraints that defines this size"`
}

type SizeListResponse struct {
	Common
	SizeConstraints []SizeConstraint `json:"constraints" description:"a list of constraints that defines this size"`
}

type SizeDetailResponse struct {
	SizeListResponse
	Timestamps
}

func NewSizeDetailResponse(s *metal.Size) *SizeDetailResponse {
	return &SizeDetailResponse{
		SizeListResponse: *NewSizeListResponse(s),
		Timestamps: Timestamps{
			Created: s.Created,
			Changed: s.Changed,
		},
	}
}

func NewSizeListResponse(s *metal.Size) *SizeListResponse {
	var constraints []SizeConstraint
	for _, c := range s.Constraints {
		constraint := SizeConstraint{
			Type: c.Type,
			Min:  c.Min,
			Max:  c.Max,
		}
		constraints = append(constraints, constraint)
	}
	return &SizeListResponse{
		Common: Common{
			Identifiable: Identifiable{
				ID: s.ID,
			},
			Describeable: Describeable{
				Name:        &s.Name,
				Description: &s.Description,
			},
		},
		SizeConstraints: constraints,
	}
}
