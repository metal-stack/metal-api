package metal

import (
	"reflect"
	"testing"
)

func TestNics_ByMac(t *testing.T) {

	// Create Nics
	var countOfNics = 3
	nicArray := make([]Nic, countOfNics)
	for i := 0; i < countOfNics; i++ {
		nicArray[i] = Nic{
			MacAddress: MacAddress("11:11:1" + string(i)),
			Name:       "swp" + string(i),
			Neighbors:  nil,
		}
	}

	// all have all as Neighbors
	for i := 0; i < countOfNics; i++ {
		nicArray[i].Neighbors = append(nicArray[0:i], nicArray[i+1:countOfNics]...)
	}

	map1 := make(map[MacAddress]*Nic)
	for i, n := range nicArray {
		map1[n.MacAddress] = &nicArray[i]
	}

	tests := []struct {
		name string
		nics Nics
		want map[MacAddress]*Nic
	}{
		// Test Data Array (only 1 data):
		{
			name: "TestNics_ByMac Test 1",
			nics: nicArray,
			want: map1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.nics.ByMac(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Nics.ByMac() = %v, want %v", got, tt.want)
			}
		})
	}
}
