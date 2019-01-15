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
			name: "testSites_ByID Test 1",
			sz:   testSites,
			want: map[string]Site{testSites[0].ID: testSites[0], testSites[1].ID: testSites[1], testSites[2].ID: testSites[2]},
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
