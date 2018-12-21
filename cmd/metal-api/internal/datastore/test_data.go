package datastore

import (
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"go.uber.org/zap"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

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
						MacAddress: metal.MacAddress("11:11:11"),
					},
					DeviceID: "d1",
				},
			},
		},
	}
	switch2 = metal.Switch{
		Base: metal.Base{
			ID: "switch2",
		},
		SiteID:            "1",
		RackID:            "rack1",
		DeviceConnections: metal.ConnectionMap{},
	}
	switch3 = metal.Switch{
		Base: metal.Base{
			ID: "switch3",
		},
		SiteID:            "1",
		RackID:            "rack1",
		DeviceConnections: metal.ConnectionMap{},
	}
	nic1 = metal.Nic{
		MacAddress: metal.MacAddress("11:11:11"),
		Name:       "swp1",
		Neighbors:  nil,
	}
	nic2 = metal.Nic{
		MacAddress: metal.MacAddress("21:11:11"),
		Name:       "swp2",
		Neighbors:  nil,
	}
	nic3 = metal.Nic{
		MacAddress: metal.MacAddress("31:11:11"),
		Name:       "swp3",
		Neighbors:  nil,
	}
	nics = metal.Nics{
		nic1, nic2, nic3,
	}
	deviceHardware1 = metal.DeviceHardware{
		Memory:   100,
		CPUCores: 1,
		Nics:     nics,
		Disks: []metal.BlockDevice{
			{
				Name: "blockdeviceName",
				Size: 1000000000000,
			},
		},
	}
	deviceHardware2 = metal.DeviceHardware{
		Memory:   1000,
		CPUCores: 2,
		Nics:     nics,
		Disks: []metal.BlockDevice{
			{
				Name: "blockdeviceName",
				Size: 2000000000000,
			},
		},
	}

	emptyResult = map[string]interface{}{}
)

func initData() {
	nic1.Neighbors = metal.Nics{nic2, nic3}
	nic2.Neighbors = metal.Nics{nic1, nic3}
	nic3.Neighbors = metal.Nics{nic1, nic2}
}
func initMockDB() (*RethinkStore, *r.Mock) {
	rs := New(
		testlogger,
		"db-addr",
		"mockdb",
		"db-user",
		"db-password",
	)
	mock := rs.Mock()

	return rs, mock
}
