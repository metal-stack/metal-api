package metal

import (
	"reflect"
	"testing"
)

func getMap(x int, y int, baseArray []Base) map[string]Site {
	// Returns a Site Map
	return map[string]Site{baseArray[x].ID: Site{baseArray[x]}, baseArray[y].ID: Site{baseArray[y]}}
}

func getTestStruct(x int, y int, baseArray []Base, siteArray []Site) struct {
	name  string
	sites Sites
	want  SiteMap
} {
	/*
		Returns 1 struct of Test Data
	*/
	return struct {
		name  string
		sites Sites
		want  SiteMap
	}{

		name: "real live data",
		sites: Sites{
			siteArray[x],
			siteArray[y],
		},
		want: getMap(x, y, baseArray),
	}
}

func getAllTestStructs(baseArray []Base, siteArray []Site) []struct {
	name  string
	sites Sites
	want  SiteMap
} {
	/*
		Returns an struct Array of all Test Data
	*/
	structArray := make([]struct {
		name  string
		sites Sites
		want  SiteMap
	}, len(siteArray)*len(baseArray))
	index := 0
	for i := 0; i < len(baseArray); i++ {
		for j := 0; j < len(siteArray); j++ {
			structArray[index] = getTestStruct(i, j, baseArray, siteArray)
			index++
		}
	}
	return structArray
}

func TestSites_ByID(t *testing.T) {

	// Definition of Base names (can be increased, all test are generated from this array automaticaly)
	var nameArray = []string{"micro", "tiny", "microAndTiny"}
	length := len(nameArray)

	// Create Base Array
	baseArray := make([]Base, length)
	for i := 0; i < length; i++ {
		baseArray[i] = Base{
			Name: nameArray[i],
			ID:   "test",
		}
	}

	// Create Site Array
	siteArray := make([]Site, length)
	for i := 0; i < length; i++ {
		siteArray[i] = Site{
			Base: baseArray[i],
		}
	}

	// Create all Test Data
	tests := getAllTestStructs(baseArray, siteArray)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sites.ByID(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Sites.ByID() = %v, want %v", got, tt.want)
			}
		})
	}
}
