package metal

import (
	"reflect"
	"testing"
)

func TestSites_ByID(t *testing.T) {
	tests := []struct {
		name string
		sz   Sites
		want SiteMap
	}{
		{
			name: "TestSites_ByID Test 1",
			sz:   TestSites,
			want: map[string]Site{TestSites[0].ID: TestSites[0], TestSites[1].ID: TestSites[1], TestSites[2].ID: TestSites[2]},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sz.ByID(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Sites.ByID() = %v, want %v", got, tt.want)
			}
		})
	}
}
