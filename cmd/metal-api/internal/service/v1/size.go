package v1

import (
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

type SizeConstraint struct {
	Type       metal.ConstraintType `json:"type" modelDescription:"a machine matches to a size in order to make them easier to categorize" enum:"cores|memory|storage|gpu" description:"the type of the constraint"`
	Min        uint64               `json:"min,omitempty" description:"the minimum value of the constraint"`
	Max        uint64               `json:"max,omitempty" description:"the maximum value of the constraint"`
	Identifier string               `json:"identifier,omitempty" description:"glob pattern which matches to the given type, for example gpu pci id"`
}

type SizeReservation struct {
	Amount       int               `json:"amount" description:"the amount of reserved machine allocations for this size"`
	Description  string            `json:"description,omitempty" description:"a description for this reservation"`
	ProjectID    string            `json:"projectid" description:"the project for which this size reservation is considered"`
	PartitionIDs []string          `json:"partitionids" description:"the partitions in which this size reservation is considered, the amount is valid for every partition"`
	Labels       map[string]string `json:"labels,omitempty" description:"free labels associated with this size reservation."`
}

type SizeCreateRequest struct {
	Common
	SizeConstraints  []SizeConstraint  `json:"constraints" description:"a list of constraints that defines this size"`
	SizeReservations []SizeReservation `json:"reservations,omitempty" description:"reservations for this size, which are considered during machine allocation" optional:"true"`
	Labels           map[string]string `json:"labels" description:"free labels that you associate with this size." optional:"true"`
}

type SizeUpdateRequest struct {
	Common
	SizeConstraints  *[]SizeConstraint `json:"constraints" description:"a list of constraints that defines this size" optional:"true"`
	SizeReservations []SizeReservation `json:"reservations,omitempty" description:"reservations for this size, which are considered during machine allocation" optional:"true"`
	Labels           map[string]string `json:"labels" description:"free labels that you associate with this size." optional:"true"`
}

type SizeResponse struct {
	Common
	SizeConstraints  []SizeConstraint  `json:"constraints" description:"a list of constraints that defines this size"`
	SizeReservations []SizeReservation `json:"reservations,omitempty" description:"reservations for this size, which are considered during machine allocation" optional:"true"`
	Labels           map[string]string `json:"labels" description:"free labels that you associate with this size."`
	Timestamps
}

type SizeReservationResponse struct {
	SizeID             string            `json:"sizeid" description:"the size id of this size reservation"`
	PartitionID        string            `json:"partitionid" description:"the partition id of this size reservation"`
	Tenant             string            `json:"tenant" description:"the tenant of this size reservation"`
	ProjectID          string            `json:"projectid" description:"the project id of this size reservation"`
	ProjectName        string            `json:"projectname" description:"the project name of this size reservation"`
	Reservations       int               `json:"reservations" description:"the amount of reservations of this size reservation"`
	UsedReservations   int               `json:"usedreservations" description:"the used amount of reservations of this size reservation"`
	ProjectAllocations int               `json:"projectallocations" description:"the amount of allocations of this project referenced by this size reservation"`
	Labels             map[string]string `json:"labels,omitempty" description:"free labels associated with this size reservation."`
}

type SizeReservationListRequest struct {
	SizeID      *string `json:"sizeid,omitempty" description:"the size id of this size reservation"`
	Tenant      *string `json:"tenant,omitempty" description:"the tenant of this size reservation"`
	ProjectID   *string `json:"projectid,omitempty" description:"the project id of this size reservation"`
	PartitionID *string `json:"partitionid,omitempty" description:"the partition id of this size reservation"`
}

type SizeSuggestRequest struct {
	MachineID string `json:"machineID" description:"machineID to retrieve size suggestion for"`
}

type SizeConstraintMatchingLog struct {
	Constraint SizeConstraint `json:"constraint" description:"the size constraint to which this log relates to"`
	Match      bool           `json:"match" description:"indicates whether the constraint matched or not"`
	Log        string         `json:"log" description:"a string representation of the matching condition"`
}

type SizeMatchingLog struct {
	Name        string                      `json:"name"`
	Log         string                      `json:"log"`
	Match       bool                        `json:"match"`
	Constraints []SizeConstraintMatchingLog `json:"constraints"`
}

func NewSizeResponse(s *metal.Size) *SizeResponse {
	if s == nil {
		return nil
	}

	constraints := []SizeConstraint{}
	for _, c := range s.Constraints {
		constraint := SizeConstraint{
			Type:       c.Type,
			Min:        c.Min,
			Max:        c.Max,
			Identifier: c.Identifier,
		}
		constraints = append(constraints, constraint)
	}

	reservations := []SizeReservation{}
	for _, r := range s.Reservations {
		reservation := SizeReservation{
			Amount:       r.Amount,
			Description:  r.Description,
			ProjectID:    r.ProjectID,
			PartitionIDs: r.PartitionIDs,
			Labels:       r.Labels,
		}
		reservations = append(reservations, reservation)
	}

	labels := map[string]string{}
	if s.Labels != nil {
		labels = s.Labels
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
		SizeReservations: reservations,
		SizeConstraints:  constraints,
		Timestamps: Timestamps{
			Created: s.Created,
			Changed: s.Changed,
		},
		Labels: labels,
	}
}
