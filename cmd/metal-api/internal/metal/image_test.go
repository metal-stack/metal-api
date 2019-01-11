package metal

import (
	"reflect"
	"testing"
)

func TestImages_ByID(t *testing.T) {

	imageMap := make(ImageMap)
	for i, f := range TestImages {
		imageMap[f.ID] = TestImages[i]
	}

	tests := []struct {
		name string
		ii   Images
		want ImageMap
	}{
		// Test Data Array (only 1 data):
		{
			name: "Test 1",
			ii:   TestImages,
			want: imageMap,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ii.ByID(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Images.ByID() = %v, want %v", got, tt.want)
			}
		})
	}
}
