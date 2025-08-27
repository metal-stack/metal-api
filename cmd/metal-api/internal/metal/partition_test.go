package metal

import (
	"errors"
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

func TestNewScalerRange(t *testing.T) {
	tests := []struct {
		name    string
		min     string
		max     string
		want    *ScalerRange
		wantErr error
	}{
		{
			name:    "min max format mismatch",
			min:     "15%",
			max:     "30",
			want:    nil,
			wantErr: errors.New("minimum and maximum pool sizes must either be both in percent or both an absolute value"),
		},
		{
			name:    "parse error for min",
			min:     "15#%",
			max:     "30",
			want:    nil,
			wantErr: errors.New("could not parse minimum waiting pool size"),
		},
		{
			name:    "parse error for max",
			min:     "15",
			max:     "#30",
			want:    nil,
			wantErr: errors.New("could not parse maximum waiting pool size"),
		},
		{
			name:    "max less than min",
			min:     "15",
			max:     "1",
			want:    nil,
			wantErr: errors.New("minimum waiting pool size must be less or equal to maximum pool size"),
		},
		{
			name:    "0 is not allowed",
			min:     "0",
			max:     "0",
			want:    nil,
			wantErr: errors.New("minimum and maximum waiting pool sizes must be greater than 0"),
		},
		{
			name:    "everything okay",
			min:     "15",
			max:     "30",
			want:    &ScalerRange{minSize: 15, maxSize: 30},
			wantErr: nil,
		},
		{
			name:    "everything okay in percent",
			min:     "15%",
			max:     "30%",
			want:    &ScalerRange{minSize: 15, maxSize: 30, inPercent: true},
			wantErr: nil,
		},
		{
			name:    "pool scaling disabled",
			want:    &ScalerRange{isDisabled: true},
			wantErr: nil,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewScalerRange(tt.min, tt.max)
			if (err != nil || tt.wantErr != nil) && (err == nil || tt.wantErr == nil || err.Error() != tt.wantErr.Error()) {
				t.Errorf("NewScalerRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewScalerRange() = %v, want %v", got, tt.want)
			}
		})
	}
}
