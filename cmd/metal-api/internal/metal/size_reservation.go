package metal

import (
	"fmt"
	"slices"

	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
)

// SizeReservation defines a reservation of a size for machine allocations
type SizeReservation struct {
	Base
	SizeID       string            `rethinkdb:"sizeid" json:"sizeid"`
	Amount       int               `rethinkdb:"amount" json:"amount"`
	ProjectID    string            `rethinkdb:"projectid" json:"projectid"`
	PartitionIDs []string          `rethinkdb:"partitionids" json:"partitionids"`
	Labels       map[string]string `rethinkdb:"labels" json:"labels"`
}

type SizeReservations []SizeReservation

func (rs *SizeReservations) BySize() map[string]SizeReservations {
	res := map[string]SizeReservations{}
	for _, rv := range *rs {
		res[rv.SizeID] = append(res[rv.SizeID], rv)
	}
	return res
}

func (rs *SizeReservations) ForPartition(partitionID string) SizeReservations {
	if rs == nil {
		return nil
	}

	var result SizeReservations
	for _, r := range *rs {
		r := r
		if slices.Contains(r.PartitionIDs, partitionID) {
			result = append(result, r)
		}
	}

	return result
}

func (rs *SizeReservations) Validate(sizes SizeMap, partitions PartitionMap, projects map[string]*mdmv1.Project) error {
	if rs == nil {
		return nil
	}

	for _, r := range *rs {
		err := r.Validate(sizes, partitions, projects)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *SizeReservation) Validate(sizes SizeMap, partitions PartitionMap, projects map[string]*mdmv1.Project) error {
	if r.Amount <= 0 {
		return fmt.Errorf("amount must be a positive integer")
	}

	if _, ok := sizes[r.SizeID]; !ok {
		return fmt.Errorf("size must exist before creating a size reservation")
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

	return nil
}
