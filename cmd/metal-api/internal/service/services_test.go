package service

import (
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/metal"
	"go.uber.org/zap"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

var (
	testlogger = zap.NewNop()

	d1 = metal.Device{
		ID:     "1",
		SiteID: "1",
		SizeID: "1",
		Allocation: &metal.DeviceAllocation{
			Name:    "d1",
			ImageID: "1",
			Project: "p1",
		},
	}
	d2 = metal.Device{
		ID:     "2",
		SiteID: "1",
		SizeID: "1",
		Allocation: &metal.DeviceAllocation{
			Name:    "d2",
			ImageID: "1",
			Project: "p2",
		},
	}
	d3 = metal.Device{
		ID:     "3",
		SiteID: "1",
		SizeID: "1",
	}
	sz1 = metal.Size{
		ID:          "1",
		Name:        "sz1",
		Description: "description 1",
		Constraints: []metal.Constraint{metal.Constraint{
			MinCores:  1,
			MaxCores:  1,
			MinMemory: 100,
			MaxMemory: 100,
		}},
	}
	sz2 = metal.Size{
		ID:          "2",
		Name:        "sz2",
		Description: "description 2",
	}
	img1 = metal.Image{
		ID:          "1",
		Name:        "img1",
		Description: "description 1",
	}
	img2 = metal.Image{
		ID:          "2",
		Name:        "img2",
		Description: "description 2",
	}
	site1 = metal.Site{
		ID:          "1",
		Name:        "site1",
		Description: "description 1",
	}
	site2 = metal.Site{
		ID:          "2",
		Name:        "site2",
		Description: "description 2",
	}

	emptyResult = map[string]interface{}{}
)

func initMockDB() (*datastore.RethinkStore, *r.Mock) {
	rs := datastore.New(
		testlogger,
		"db-addr",
		"mockdb",
		"db-user",
		"db-password",
	)
	mock := rs.Mock()

	return rs, mock
}
