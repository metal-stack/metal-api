package metal

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	mdmv1 "github.com/metal-stack/masterdata-api/api/v1"
	"github.com/metal-stack/metal-lib/pkg/testcommon"
)

func TestReservations_ForPartition(t *testing.T) {
	tests := []struct {
		name        string
		rs          *SizeReservations
		partitionID string
		want        SizeReservations
	}{
		{
			name:        "nil",
			rs:          nil,
			partitionID: "a",
			want:        nil,
		},
		{
			name: "correctly filtered",
			rs: &SizeReservations{
				{
					PartitionIDs: []string{"a", "b"},
				},
				{
					PartitionIDs: []string{"c"},
				},
				{
					PartitionIDs: []string{"a"},
				},
			},
			partitionID: "a",
			want: SizeReservations{
				{
					PartitionIDs: []string{"a", "b"},
				},
				{
					PartitionIDs: []string{"a"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rs.ForPartition(tt.partitionID); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Reservations.ForPartition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReservations_Validate(t *testing.T) {
	tests := []struct {
		name       string
		sizes      SizeMap
		partitions PartitionMap
		projects   map[string]*mdmv1.Project
		rs         *SizeReservations
		wantErr    error
	}{
		{
			name:       "empty reservations",
			sizes:      nil,
			partitions: nil,
			projects:   nil,
			rs:         nil,
			wantErr:    nil,
		},
		{
			name: "invalid amount",
			sizes: SizeMap{
				"c1": Size{},
			},
			partitions: PartitionMap{
				"a": Partition{},
				"b": Partition{},
				"c": Partition{},
			},
			projects: map[string]*mdmv1.Project{
				"1": {},
				"2": {},
				"3": {},
			},
			rs: &SizeReservations{
				{
					SizeID:       "c1",
					Amount:       -3,
					ProjectID:    "3",
					PartitionIDs: []string{"b"},
				},
			},
			wantErr: fmt.Errorf("amount must be a positive integer"),
		},
		{
			name: "size does not exist",
			sizes: SizeMap{
				"c1": Size{},
			},
			partitions: PartitionMap{
				"a": Partition{},
				"b": Partition{},
				"c": Partition{},
			},
			projects: map[string]*mdmv1.Project{
				"1": {},
				"2": {},
				"3": {},
			},
			rs: &SizeReservations{
				{
					SizeID:       "c2",
					Amount:       3,
					ProjectID:    "3",
					PartitionIDs: []string{"d"},
				},
			},
			wantErr: fmt.Errorf("size must exist before creating a size reservation"),
		},
		{
			name: "no partitions referenced",
			sizes: SizeMap{
				"c1": Size{},
			},
			partitions: PartitionMap{
				"a": Partition{},
				"b": Partition{},
				"c": Partition{},
			},
			projects: map[string]*mdmv1.Project{
				"1": {},
				"2": {},
				"3": {},
			},
			rs: &SizeReservations{
				{
					SizeID:    "c1",
					Amount:    3,
					ProjectID: "3",
				},
			},
			wantErr: fmt.Errorf("at least one partition id must be specified"),
		},
		{
			name: "partition does not exist",
			sizes: SizeMap{
				"c1": Size{},
			},
			partitions: PartitionMap{
				"a": Partition{},
				"b": Partition{},
				"c": Partition{},
			},
			projects: map[string]*mdmv1.Project{
				"1": {},
				"2": {},
				"3": {},
			},
			rs: &SizeReservations{
				{
					SizeID:       "c1",
					Amount:       3,
					ProjectID:    "3",
					PartitionIDs: []string{"d"},
				},
			},
			wantErr: fmt.Errorf("partition must exist before creating a size reservation"),
		},
		{
			name: "partition duplicates",
			sizes: SizeMap{
				"c1": Size{},
			},
			partitions: PartitionMap{
				"a": Partition{},
				"b": Partition{},
				"c": Partition{},
			},
			projects: map[string]*mdmv1.Project{
				"1": {},
				"2": {},
				"3": {},
			},
			rs: &SizeReservations{
				{
					SizeID:       "c1",
					Amount:       3,
					ProjectID:    "3",
					PartitionIDs: []string{"a", "b", "c", "b"},
				},
			},
			wantErr: fmt.Errorf("partitions must not contain duplicates"),
		},
		{
			name: "no project referenced",
			sizes: SizeMap{
				"c1": Size{},
			},
			partitions: PartitionMap{
				"a": Partition{},
				"b": Partition{},
				"c": Partition{},
			},
			projects: map[string]*mdmv1.Project{
				"1": {},
				"2": {},
				"3": {},
			},
			rs: &SizeReservations{
				{
					SizeID:       "c1",
					Amount:       3,
					PartitionIDs: []string{"a"},
				},
			},
			wantErr: fmt.Errorf("project id must be specified"),
		},
		{
			name: "project does not exist",
			sizes: SizeMap{
				"c1": Size{},
			},
			partitions: PartitionMap{
				"a": Partition{},
				"b": Partition{},
				"c": Partition{},
			},
			projects: map[string]*mdmv1.Project{
				"1": {},
				"2": {},
				"3": {},
			},
			rs: &SizeReservations{
				{
					SizeID:       "c1",
					Amount:       3,
					ProjectID:    "4",
					PartitionIDs: []string{"a"},
				},
			},
			wantErr: fmt.Errorf("project must exist before creating a size reservation"),
		},
		{
			name: "valid reservation",
			sizes: SizeMap{
				"c1": Size{},
			},
			partitions: PartitionMap{
				"a": Partition{},
				"b": Partition{},
				"c": Partition{},
			},
			projects: map[string]*mdmv1.Project{
				"1": {},
				"2": {},
				"3": {},
			},
			rs: &SizeReservations{
				{
					SizeID:       "c1",
					Amount:       3,
					ProjectID:    "2",
					PartitionIDs: []string{"b", "c"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rs.Validate(tt.sizes, tt.partitions, tt.projects)
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (-want +got):\n%s", diff)
			}
		})
	}
}
