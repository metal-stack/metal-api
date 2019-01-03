package service

import "go.uber.org/zap"

var testlogger = zap.NewNop()
var emptyResult = map[string]interface{}{}

/*
var (
	testlogger = zap.NewNop()

	d1 = metal.Device{
		Base:   metal.Base{ID: "1"},
		SiteID: "1",
		SizeID: "1",
		Allocation: &metal.DeviceAllocation{
			Name:    "d1",
			ImageID: "1",
			Project: "p1",
		},
	}
	d2 = metal.Device{
		Base:   metal.Base{ID: "2"},
		SiteID: "1",
		SizeID: "1",
		Allocation: &metal.DeviceAllocation{
			Name:    "d2",
			ImageID: "1",
			Project: "p2",
		},
	}
	d3 = metal.Device{
		Base:   metal.Base{ID: "3"},
		SiteID: "1",
		SizeID: "1",
	}
	sz1 = metal.Size{
		Base: metal.Base{
			ID:          "1",
			Name:        "sz1",
			Description: "description 1",
		},
		Constraints: []metal.Constraint{metal.Constraint{
			MinCores:  1,
			MaxCores:  1,
			MinMemory: 100,
			MaxMemory: 100,
		}},
	}
	sz2 = metal.Size{
		Base: metal.Base{
			ID:          "2",
			Name:        "sz2",
			Description: "description 2",
		},
	}
	img1 = metal.Image{
		Base: metal.Base{
			ID:          "1",
			Name:        "img1",
			Description: "description 1",
		},
	}
	img2 = metal.Image{
		Base: metal.Base{
			ID:          "2",
			Name:        "img2",
			Description: "description 2",
		},
	}
	site1 = metal.Site{
		Base: metal.Base{
			ID:          "1",
			Name:        "site1",
			Description: "description 1",
		},
	}
	site2 = metal.Site{
		Base: metal.Base{
			ID:          "2",
			Name:        "site2",
			Description: "description 2",
		},
	}
	switch1 = metal.Switch{
		Base: metal.Base{
			ID: "switch1",
		},
		SiteID: "1",
		RackID: "rack1",
		DeviceConnections: metal.ConnectionMap{
			"d1": metal.Connections{
				metal.Connection{
					Nic: metal.Nic{
						MacAddress: metal.MacAddress("mymac"),
					},
					DeviceID: "d1",
				},
			},
		},
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
*/
