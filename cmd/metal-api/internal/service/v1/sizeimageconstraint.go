package v1

import "github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"

type SizeImageConstraintBase struct {
	Images map[string]string `json:"images" description:"a list of images for this constraints apply"`
}

type SizeImageConstraintResponse struct {
	Common
	SizeImageConstraintBase `json:"constraints" description:"a list of constraints that for this size"`
}

type SizeImageConstraintCreateRequest struct {
	Common
	SizeImageConstraintBase `json:"constraints" description:"a list of constraints that for this size" optional:"true"`
}
type SizeImageConstraintTryRequest struct {
	SizeID  string `json:"size"`
	ImageID string `json:"image"`
}
type SizeImageConstraintUpdateRequest struct {
	Common
	SizeImageConstraintBase `json:"constraints" description:"a list of constraints that for this size" optional:"true"`
}

func NewSizeImageConstraint(s SizeImageConstraintCreateRequest) *metal.SizeImageConstraint {
	var (
		name        string
		description string
	)
	if s.Common.Describable.Name != nil {
		name = *s.Common.Describable.Name
	}
	if s.Common.Describable.Description != nil {
		description = *s.Common.Describable.Description
	}
	return &metal.SizeImageConstraint{
		Base: metal.Base{
			ID:          s.ID,
			Name:        name,
			Description: description,
		},
		Images: s.Images,
	}
}

func NewSizeImageConstraintResponse(s *metal.SizeImageConstraint) *SizeImageConstraintResponse {
	return &SizeImageConstraintResponse{
		Common: Common{
			Identifiable: Identifiable{ID: s.ID},
			Describable:  Describable{Name: &s.Name, Description: &s.Description},
		},
		SizeImageConstraintBase: SizeImageConstraintBase{
			Images: s.Images,
		},
	}
}
