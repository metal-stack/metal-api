package metal

import (
	"reflect"
	"testing"
)

func TestImages_ByID(t *testing.T) {

	var nameArray = []string{"micro", "tiny", "microAndTiny"}
	length := len(nameArray)

	imageArray := make([]Image, length)
	for i, n := range nameArray {
		imageArray[i] = Image{
			Base: Base{
				Name: n,
				ID:   n,
			},
			URL: "example.net",
		}
	}

	imageMap := make(ImageMap)
	for i, f := range imageArray {
		imageMap[f.ID] = imageArray[i]
	}

	tests := []struct {
		name string
		ii   Images
		want ImageMap
	}{
		// Test Data Array (only 1 data):
		{
			name: "Test 1",
			ii:   imageArray,
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
