package metal

import (
	"reflect"
	"testing"
)

func TestPartitions_ByID(t *testing.T) {
	testPartitions := []Partition{
		{
			Base: Base{
				ID:          "1",
				Name:        "partition1",
				Description: "description 1",
			},
		},
		{
			Base: Base{
				ID:          "2",
				Name:        "partition2",
				Description: "description 2",
			},
		},
		{
			Base: Base{
				ID:          "3",
				Name:        "partition3",
				Description: "description 3",
			},
		},
	}

	tests := []struct {
		name string
		sz   Partitions
		want PartitionMap
	}{
		{
			name: "ByID Test 1",
			sz:   testPartitions,
			want: map[string]Partition{testPartitions[0].ID: testPartitions[0], testPartitions[1].ID: testPartitions[1], testPartitions[2].ID: testPartitions[2]},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sz.ByID(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Partitions.ByID() = %v, want %v", got, tt.want)
			}
		})
	}
}
