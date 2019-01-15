package testdata

import (
	"fmt"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

// If you want to add some Test Data, add it also to the following places:
// -- To the Mocks, if needed (datastore.testdata)
// -- To the corrisponding arrays,( At the Bottom of this File)

// Also run the Tests (cd ./cloud-native/metal/metal-api/ 	& 	go test ./...)

var (
	// Devices
	D1 = metal.Device{
		Base:   metal.Base{ID: "1"},
		SiteID: "1",
		Site:   Site1,
		SizeID: "1",
		Size:   &Sz1,
		Allocation: &metal.DeviceAllocation{
			Name:    "d1",
			ImageID: "1",
			Project: "p1",
		},
	}
	D2 = metal.Device{
		Base:   metal.Base{ID: "2"},
		SiteID: "1",
		Site:   Site1,
		SizeID: "1",
		Size:   &Sz1,
		Allocation: &metal.DeviceAllocation{
			Name:    "d2",
			ImageID: "1",
			Project: "p2",
		},
	}
	D3 = metal.Device{
		Base:   metal.Base{ID: "3"},
		SiteID: "1",
		Site:   Site1,
		SizeID: "1",
		Size:   &Sz1,
	}
	D4 = metal.Device{
		Base:       metal.Base{ID: "4"},
		SiteID:     "1",
		Site:       Site1,
		SizeID:     "1",
		Size:       &Sz1,
		Allocation: nil,
	}
	D5 = metal.Device{
		Base: metal.Base{
			Name:        "1-core/100 B",
			Description: "a device with 1 core(s) and 100 B of RAM",
			ID:          "5",
		},
		RackID:     "1",
		SiteID:     "1",
		Site:       Site1,
		SizeID:     "1",
		Size:       &Sz1,
		Allocation: nil,
		Hardware:   DeviceHardware1,
	}
	D6 = metal.Device{
		Base: metal.Base{
			Name:        "1-core/100 B",
			Description: "a device with 1 core(s) and 100 B of RAM",
			ID:          "6",
		},
		RackID: "1",
		SiteID: "999",
		Site:   Site1,
		SizeID: "1", // No Size
		Size:   nil,
		Allocation: &metal.DeviceAllocation{
			Name:    "6",
			ImageID: "999",
			Project: "p2",
		},
		Hardware: DeviceHardware1,
	}
	D7 = metal.Device{
		Base: metal.Base{
			Name:        "1-core/100 B",
			Description: "a device with 1 core(s) and 100 B of RAM",
			ID:          "6",
		},
		RackID: "1",
		SiteID: "1",
		Site:   Site1,
		SizeID: "999", // No Size
		Size:   nil,
		Allocation: &metal.DeviceAllocation{
			Name:    "7",
			ImageID: "1",
			Project: "p2",
		},
		Hardware: DeviceHardware1,
	}
	D8 = metal.Device{
		Base: metal.Base{
			Name:        "1-core/100 B",
			Description: "a device with 1 core(s) and 100 B of RAM",
			ID:          "6",
		},
		RackID: "1",
		SiteID: "1",
		Site:   Site1,
		SizeID: "1", // No Size
		Size:   nil,
		Allocation: &metal.DeviceAllocation{
			Name:    "8",
			ImageID: "999",
			Project: "p2",
		},
		Hardware: DeviceHardware1,
	}

	// Sizes
	Sz1 = metal.Size{
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
	Sz2 = metal.Size{
		Base: metal.Base{
			ID:          "2",
			Name:        "sz2",
			Description: "description 2",
		},
	}
	Sz3 = metal.Size{
		Base: metal.Base{
			ID:          "3",
			Name:        "sz3",
			Description: "description 3",
		},
	}
	Sz999 = metal.Size{
		Base: metal.Base{
			ID:          "999",
			Name:        "sz1",
			Description: "description 1",
		},
		Constraints: []metal.Constraint{metal.Constraint{
			MinCores:  888,
			MaxCores:  1111,
			MinMemory: 100,
			MaxMemory: 100,
		}},
	}

	// Images
	Img1 = metal.Image{
		Base: metal.Base{
			ID:          "1",
			Name:        "Image 1",
			Description: "description 1",
		},
	}
	Img2 = metal.Image{
		Base: metal.Base{
			ID:          "2",
			Name:        "Image 2",
			Description: "description 2",
		},
	}
	Img3 = metal.Image{
		Base: metal.Base{
			ID:          "3",
			Name:        "Image 3",
			Description: "description 3",
		},
	}

	// Sites
	Site1 = metal.Site{
		Base: metal.Base{
			ID:          "1",
			Name:        "site1",
			Description: "description 1",
		},
	}
	Site2 = metal.Site{
		Base: metal.Base{
			ID:          "2",
			Name:        "site2",
			Description: "description 2",
		},
	}
	Site3 = metal.Site{
		Base: metal.Base{
			ID:          "3",
			Name:        "site3",
			Description: "description 3",
		},
	}

	// Switches
	Switch1 = metal.Switch{
		Base: metal.Base{
			ID: "switch1",
		},
		SiteID: "1",
		RackID: "1",
		Nics: []metal.Nic{
			Nic1,
			Nic2,
			Nic3,
		},
		DeviceConnections: metal.ConnectionMap{
			"1": metal.Connections{
				metal.Connection{
					Nic: metal.Nic{
						MacAddress: metal.MacAddress("11:11:11:11:11:11"),
					},
					DeviceID: "1",
				},
				metal.Connection{
					Nic: metal.Nic{
						MacAddress: metal.MacAddress("11:11:11:11:11:22"),
					},
					DeviceID: "1",
				},
			},
		},
	}
	Switch2 = metal.Switch{
		Base: metal.Base{
			ID: "switch2",
		},
		SiteID:            "1",
		RackID:            "2",
		DeviceConnections: metal.ConnectionMap{},
	}
	Switch3 = metal.Switch{
		Base: metal.Base{
			ID: "switch3",
		},
		SiteID:            "1",
		RackID:            "3",
		DeviceConnections: metal.ConnectionMap{},
	}

	// Nics
	Nic1 = metal.Nic{
		MacAddress: metal.MacAddress("11:11:11:11:11:11"),
		Name:       "eth0",
		Neighbors: []metal.Nic{
			metal.Nic{
				MacAddress: "21:11:11:11:11:11",
			},
			metal.Nic{
				MacAddress: "31:11:11:11:11:11",
			},
		},
	}
	Nic2 = metal.Nic{
		MacAddress: metal.MacAddress("21:11:11:11:11:11"),
		Name:       "swp1",
		Neighbors: []metal.Nic{
			metal.Nic{
				MacAddress: "11:11:11:11:11:11",
			},
			metal.Nic{
				MacAddress: "31:11:11:11:11:11",
			},
		},
	}
	Nic3 = metal.Nic{
		MacAddress: metal.MacAddress("31:11:11:11:11:11"),
		Name:       "swp2",
		Neighbors: []metal.Nic{
			metal.Nic{
				MacAddress: "21:11:11:11:11:11",
			},
			metal.Nic{
				MacAddress: "11:11:11:11:11:11",
			},
		},
	}

	// IPMIs
	IPMI1 = metal.IPMI{
		ID:         "IPMI-1",
		Address:    "192.168.0.1",
		MacAddress: "11:11:11",
		User:       "User",
		Password:   "Password",
		Interface:  "Interface",
	}

	// DeviceHardwares
	DeviceHardware1 = metal.DeviceHardware{
		Memory:   100,
		CPUCores: 1,
		Nics:     TestNics,
		Disks: []metal.BlockDevice{
			{
				Name: "blockdeviceName",
				Size: 1000000000000,
			},
		},
	}
	DeviceHardware2 = metal.DeviceHardware{
		Memory:   1000,
		CPUCores: 2,
		Nics:     TestNics,
		Disks: []metal.BlockDevice{
			{
				Name: "blockdeviceName",
				Size: 2000000000000,
			},
		},
	}

	// All Images
	TestImages = []metal.Image{
		Img1, Img2, Img3,
	}

	// All Sizes
	TestSizes = []metal.Size{
		Sz1, Sz2, Sz3,
	}

	// All Sites
	TestSites = []metal.Site{
		Site1, Site2, Site3,
	}

	// All Nics
	TestNics = metal.Nics{
		Nic1, Nic2, Nic3,
	}

	// All Switches
	TestSwitches = []metal.Switch{
		Switch1, Switch2, Switch3,
	}
	TestMacs = []string{
		"11:11:11:11:11:11",
		"11:11:11:11:11:22",
		"11:11:11:11:11:33",
		"11:11:11:11:11:11",
		"22:11:11:11:11:11",
		"33:11:11:11:11:11",
	}

	// Create the Connections Array
	TestConnections = []metal.Connection{
		metal.Connection{
			Nic: metal.Nic{
				Name:       "swp1",
				MacAddress: "11:11:11",
			},
			DeviceID: "device-1",
		},
		metal.Connection{
			Nic: metal.Nic{
				Name:       "swp2",
				MacAddress: "22:11:11",
			},
			DeviceID: "device-2",
		},
	}

	TestDeviceHardwares = []metal.DeviceHardware{
		DeviceHardware1, DeviceHardware2,
	}

	// All Devices
	TestDevices = []metal.Device{
		D1, D2, D3, D4, D5,
	}

	EmptyResult = map[string]interface{}{}
)

/*
InitMockDBData ...

Description:
This Function initializes the Data of a mocked rethink DB.
To get a Mocked r, execute datastore.InitMockDB().
If custom mocks should be used insted of here defined mocks, they can be added to the Mock Object bevore this function is called.

Hints for modifying this function:
If there need to be additional mocks added, they should be added before default mocs, which contain "r.MockAnything()"

Parameters:
- Mock 			// The Mock endpoint (Used for mocks)
*/
func InitMockDBData(mock *r.Mock) {

	// Create Tables
	mock.On(r.DB("mockdb").TableCreate(r.MockAnything()))
	mock.On(r.DB("mockdb").TableCreate(r.MockAnything(), r.TableCreateOpts{
		Shards: 1, Replicas: 1,
	}))

	// Create index
	mock.On(r.DB("mockdb").Table("device").IndexCreate("project")).Return(r.WriteResponse{}, nil)

	// X.Get(i)
	mock.On(r.DB("mockdb").Table("size").Get("1")).Return(Sz1, nil)
	mock.On(r.DB("mockdb").Table("size").Get("2")).Return(Sz2, nil)
	mock.On(r.DB("mockdb").Table("size").Get("3")).Return(Sz3, nil)
	mock.On(r.DB("mockdb").Table("size").Get("404")).Return(nil, fmt.Errorf("Test Error"))
	mock.On(r.DB("mockdb").Table("size").Get("999")).Return(nil, nil)
	mock.On(r.DB("mockdb").Table("site").Get("1")).Return(Site1, nil)
	mock.On(r.DB("mockdb").Table("site").Get("2")).Return(Site2, nil)
	mock.On(r.DB("mockdb").Table("site").Get("3")).Return(Site3, nil)
	mock.On(r.DB("mockdb").Table("site").Get("404")).Return(nil, fmt.Errorf("Test Error"))
	mock.On(r.DB("mockdb").Table("site").Get("999")).Return(nil, nil)
	mock.On(r.DB("mockdb").Table("image").Get("1")).Return(Img1, nil)
	mock.On(r.DB("mockdb").Table("image").Get("2")).Return(Img2, nil)
	mock.On(r.DB("mockdb").Table("image").Get("3")).Return(Img3, nil)
	mock.On(r.DB("mockdb").Table("image").Get("404")).Return(nil, fmt.Errorf("Test Error"))
	mock.On(r.DB("mockdb").Table("image").Get("999")).Return(nil, nil)
	mock.On(r.DB("mockdb").Table("device").Get("1")).Return(D1, nil)
	mock.On(r.DB("mockdb").Table("device").Get("2")).Return(D2, nil)
	mock.On(r.DB("mockdb").Table("device").Get("3")).Return(D3, nil)
	mock.On(r.DB("mockdb").Table("device").Get("4")).Return(D4, nil)
	mock.On(r.DB("mockdb").Table("device").Get("5")).Return(D5, nil)
	mock.On(r.DB("mockdb").Table("device").Get("6")).Return(D6, nil)
	mock.On(r.DB("mockdb").Table("device").Get("7")).Return(D7, nil)
	mock.On(r.DB("mockdb").Table("device").Get("8")).Return(D8, nil)
	mock.On(r.DB("mockdb").Table("device").Get("404")).Return(nil, fmt.Errorf("Test Error"))
	mock.On(r.DB("mockdb").Table("device").Get("999")).Return(nil, nil)
	mock.On(r.DB("mockdb").Table("switch").Get("switch1")).Return(Switch1, nil)
	mock.On(r.DB("mockdb").Table("switch").Get("switch2")).Return(Switch2, nil)
	mock.On(r.DB("mockdb").Table("switch").Get("switch3")).Return(Switch3, nil)
	mock.On(r.DB("mockdb").Table("switch").Get("switch404")).Return(nil, fmt.Errorf("Test Error"))
	mock.On(r.DB("mockdb").Table("switch").Get("switch999")).Return(nil, nil)
	mock.On(r.DB("mockdb").Table("ipmi").Get("IPMI-1")).Return(IPMI1, nil)

	// X.Get: Default: Moc all, return Empty?? Something else?:
	mock.On(r.DB("mockdb").Table("size").Get(r.MockAnything()).Replace(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("site").Get(r.MockAnything()).Replace(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("device").Get(r.MockAnything()).Replace(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("image").Get(r.MockAnything()).Replace(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("switch").Get(r.MockAnything()).Replace(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("ipmi").Get(r.MockAnything()).Replace(r.MockAnything())).Return(EmptyResult, nil)

	// X.GetAll
	mock.On(r.DB("mockdb").Table("size")).Return(TestSizes, nil)
	mock.On(r.DB("mockdb").Table("site")).Return(TestSites, nil)
	mock.On(r.DB("mockdb").Table("image")).Return(TestImages, nil)
	mock.On(r.DB("mockdb").Table("device")).Return(TestDevices, nil)
	mock.On(r.DB("mockdb").Table("switch")).Return(TestSwitches, nil)

	// X.Delete
	mock.On(r.DB("mockdb").Table("device").Get(r.MockAnything()).Delete()).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("image").Get(r.MockAnything()).Delete()).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("site").Get(r.MockAnything()).Delete()).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("size").Get(r.MockAnything()).Delete()).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("ipmi").Get(r.MockAnything()).Delete()).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("switch").Get(r.MockAnything()).Delete()).Return(EmptyResult, nil)

	// X.Get.Replace
	mock.On(r.DB("mockdb").Table("device").Get(r.MockAnything()).Replace(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("image").Get(r.MockAnything()).Replace(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("site").Get(r.MockAnything()).Replace(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("size").Get(r.MockAnything()).Replace(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("ipmi").Get(r.MockAnything()).Replace(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("switch").Get(r.MockAnything()).Replace(r.MockAnything())).Return(EmptyResult, nil)

	// X.insert
	mock.On(r.DB("mockdb").Table("ipmi").Insert(IPMI1, r.InsertOpts{
		Conflict: "replace",
	})).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("device").Insert(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("image").Insert(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("site").Insert(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("size").Insert(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("ipmi").Insert(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("switch").Insert(r.MockAnything())).Return(EmptyResult, nil)

	mock.On(r.DB("mockdb").Table("device").Insert(r.MockAnything(), r.InsertOpts{
		Conflict: "replace",
	})).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("image").Insert(r.MockAnything(), r.InsertOpts{
		Conflict: "replace",
	})).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("site").Insert(r.MockAnything(), r.InsertOpts{
		Conflict: "replace",
	})).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("size").Insert(r.MockAnything(), r.InsertOpts{
		Conflict: "replace",
	})).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("switch").Insert(r.MockAnything(), r.InsertOpts{
		Conflict: "replace",
	})).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("ipmi").Insert(r.MockAnything(), r.InsertOpts{
		Conflict: "replace",
	})).Return(EmptyResult, nil)

	// X.Filter (very Test Specific)
	mock.On(r.DB("mockdb").Table("device").Filter(func(var_1 r.Term) r.Term { return var_1.Field("macAddresses").Contains("11:11:11") })).Return([]metal.Device{
		D1,
	}, nil)

	mock.On(r.DB("mockdb").Table("switch").Filter(r.MockAnything(), r.FilterOpts{})).Return([]metal.Switch{}, nil)

	mock.On(r.DB("mockdb").Table("size").Get(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("site").Get(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("device").Get(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("image").Get(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("switch").Get(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("ipmi").Get(r.MockAnything())).Return(EmptyResult, nil)

	return
}
