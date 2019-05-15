package metal

import (
	"reflect"
	"testing"
)

func TestImages_ByID(t *testing.T) {

	testImages := []Image{
		{
			Base: Base{
				ID:          "1",
				Name:        "Image 1",
				Description: "description 1",
			},
		},
		{
			Base: Base{
				ID:          "2",
				Name:        "Image 2",
				Description: "description 2",
			},
		},
		{
			Base: Base{
				ID:          "3",
				Name:        "Image 3",
				Description: "description 3",
			},
		},
	}

	imageMap := make(ImageMap)
	for i, f := range testImages {
		imageMap[f.ID] = testImages[i]
	}

	tests := []struct {
		name string
		ii   Images
		want ImageMap
	}{
		// Test Data Array (only 1 data):
		{
			name: "TestImages_ByID Test 1",
			ii:   testImages,
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
