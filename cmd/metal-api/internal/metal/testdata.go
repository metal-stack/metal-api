package metal

// If you want to add some Test Data, add it also to the following places:
// -- To the Mocks, if needed (datastore.testdata)
// -- To the corrisponding arrays,( At the Bottom of this File)

// Also run the Tests (cd ./cloud-native/metal/metal-api/ 	& 	go test ./...)

var (
	// Devices
	d1 = Device{
		Base:   Base{ID: "1"},
		SiteID: "1",
		Site:   site1,
		SizeID: "1",
		Size:   &sz1,
		Allocation: &DeviceAllocation{
			Name:    "d1",
			ImageID: "1",
			Project: "p1",
		},
	}
	d2 = Device{
		Base:   Base{ID: "2"},
		SiteID: "1",
		Site:   site1,
		SizeID: "1",
		Size:   &sz1,
		Allocation: &DeviceAllocation{
			Name:    "d2",
			ImageID: "1",
			Project: "p2",
		},
	}
	d3 = Device{
		Base:   Base{ID: "3"},
		SiteID: "1",
		Site:   site1,
		SizeID: "1",
		Size:   &sz1,
	}
	d4 = Device{
		Base:       Base{ID: "4"},
		SiteID:     "1",
		Site:       site1,
		SizeID:     "1",
		Size:       &sz1,
		Allocation: nil,
	}
	d5 = Device{
		Base: Base{
			Name:        "1-core/100 B",
			Description: "a device with 1 core(s) and 100 B of RAM",
			ID:          "5",
		},
		RackID:     "1",
		SiteID:     "1",
		Site:       site1,
		SizeID:     "1",
		Size:       &sz1,
		Allocation: nil,
		Hardware:   deviceHardware1,
	}
	d6 = Device{
		Base: Base{
			Name:        "1-core/100 B",
			Description: "a device with 1 core(s) and 100 B of RAM",
			ID:          "6",
		},
		RackID: "1",
		SiteID: "999",
		Site:   site1,
		SizeID: "1", // No Size
		Size:   nil,
		Allocation: &DeviceAllocation{
			Name:    "6",
			ImageID: "999",
			Project: "p2",
		},
		Hardware: deviceHardware1,
	}
	d7 = Device{
		Base: Base{
			Name:        "1-core/100 B",
			Description: "a device with 1 core(s) and 100 B of RAM",
			ID:          "6",
		},
		RackID: "1",
		SiteID: "1",
		Site:   site1,
		SizeID: "999", // No Size
		Size:   nil,
		Allocation: &DeviceAllocation{
			Name:    "7",
			ImageID: "1",
			Project: "p2",
		},
		Hardware: deviceHardware1,
	}
	d8 = Device{
		Base: Base{
			Name:        "1-core/100 B",
			Description: "a device with 1 core(s) and 100 B of RAM",
			ID:          "6",
		},
		RackID: "1",
		SiteID: "1",
		Site:   site1,
		SizeID: "1", // No Size
		Size:   nil,
		Allocation: &DeviceAllocation{
			Name:    "8",
			ImageID: "999",
			Project: "p2",
		},
		Hardware: deviceHardware1,
	}

	// Sizes
	sz1 = Size{
		Base: Base{
			ID:          "1",
			Name:        "sz1",
			Description: "description 1",
		},
		Constraints: []Constraint{Constraint{
			MinCores:  1,
			MaxCores:  1,
			MinMemory: 100,
			MaxMemory: 100,
		}},
	}
	sz2 = Size{
		Base: Base{
			ID:          "2",
			Name:        "sz2",
			Description: "description 2",
		},
	}
	sz3 = Size{
		Base: Base{
			ID:          "3",
			Name:        "sz3",
			Description: "description 3",
		},
	}
	sz999 = Size{
		Base: Base{
			ID:          "999",
			Name:        "sz1",
			Description: "description 1",
		},
		Constraints: []Constraint{Constraint{
			MinCores:  888,
			MaxCores:  1111,
			MinMemory: 100,
			MaxMemory: 100,
		}},
	}

	// Images
	img1 = Image{
		Base: Base{
			ID:          "1",
			Name:        "Image 1",
			Description: "description 1",
		},
	}
	img2 = Image{
		Base: Base{
			ID:          "2",
			Name:        "Image 2",
			Description: "description 2",
		},
	}
	img3 = Image{
		Base: Base{
			ID:          "3",
			Name:        "Image 3",
			Description: "description 3",
		},
	}

	// Sites
	site1 = Site{
		Base: Base{
			ID:          "1",
			Name:        "site1",
			Description: "description 1",
		},
	}
	site2 = Site{
		Base: Base{
			ID:          "2",
			Name:        "site2",
			Description: "description 2",
		},
	}
	site3 = Site{
		Base: Base{
			ID:          "3",
			Name:        "site3",
			Description: "description 3",
		},
	}

	// Switches
	switch1 = Switch{
		Base: Base{
			ID: "switch1",
		},
		SiteID: "1",
		RackID: "1",
		Nics: []Nic{
			nic1,
			nic2,
			nic3,
		},
		DeviceConnections: ConnectionMap{
			"1": Connections{
				Connection{
					Nic: Nic{
						MacAddress: MacAddress("11:11:11:11:11:11"),
					},
					DeviceID: "1",
				},
				Connection{
					Nic: Nic{
						MacAddress: MacAddress("11:11:11:11:11:22"),
					},
					DeviceID: "1",
				},
			},
		},
	}
	switch2 = Switch{
		Base: Base{
			ID: "switch2",
		},
		SiteID:            "1",
		RackID:            "2",
		DeviceConnections: ConnectionMap{},
	}
	switch3 = Switch{
		Base: Base{
			ID: "switch3",
		},
		SiteID:            "1",
		RackID:            "3",
		DeviceConnections: ConnectionMap{},
	}

	// Nics
	nic1 = Nic{
		MacAddress: MacAddress("11:11:11:11:11:11"),
		Name:       "eth0",
		Neighbors: []Nic{
			Nic{
				MacAddress: "21:11:11:11:11:11",
			},
			Nic{
				MacAddress: "31:11:11:11:11:11",
			},
		},
	}
	nic2 = Nic{
		MacAddress: MacAddress("21:11:11:11:11:11"),
		Name:       "swp1",
		Neighbors: []Nic{
			Nic{
				MacAddress: "11:11:11:11:11:11",
			},
			Nic{
				MacAddress: "31:11:11:11:11:11",
			},
		},
	}
	nic3 = Nic{
		MacAddress: MacAddress("31:11:11:11:11:11"),
		Name:       "swp2",
		Neighbors: []Nic{
			Nic{
				MacAddress: "21:11:11:11:11:11",
			},
			Nic{
				MacAddress: "11:11:11:11:11:11",
			},
		},
	}

	// IPMIs
	ipmi1 = IPMI{
		ID:         "IPMI-1",
		Address:    "192.168.0.1",
		MacAddress: "11:11:11",
		User:       "User",
		Password:   "Password",
		Interface:  "Interface",
	}

	// DeviceHardwares
	deviceHardware1 = DeviceHardware{
		Memory:   100,
		CPUCores: 1,
		Nics:     testNics,
		Disks: []BlockDevice{
			{
				Name: "blockdeviceName",
				Size: 1000000000000,
			},
		},
	}
	deviceHardware2 = DeviceHardware{
		Memory:   1000,
		CPUCores: 2,
		Nics:     testNics,
		Disks: []BlockDevice{
			{
				Name: "blockdeviceName",
				Size: 2000000000000,
			},
		},
	}

	// All Images
	testImages = []Image{
		img1, img2, img3,
	}

	// All Sizes
	testSizes = []Size{
		sz1, sz2, sz3,
	}

	// All Sites
	testSites = []Site{
		site1, site2, site3,
	}

	// All Nics
	testNics = Nics{
		nic1, nic2, nic3,
	}

	// All Switches
	testSwitches = []Switch{
		switch1, switch2, switch3,
	}
	testMacs = []string{
		"11:11:11:11:11:11",
		"11:11:11:11:11:22",
		"11:11:11:11:11:33",
		"11:11:11:11:11:11",
		"22:11:11:11:11:11",
		"33:11:11:11:11:11",
	}

	// Create the Connections Array
	testConnections = []Connection{
		Connection{
			Nic: Nic{
				Name:       "swp1",
				MacAddress: "11:11:11",
			},
			DeviceID: "device-1",
		},
		Connection{
			Nic: Nic{
				Name:       "swp2",
				MacAddress: "22:11:11",
			},
			DeviceID: "device-2",
		},
	}

	testDeviceHardwares = []DeviceHardware{
		deviceHardware1, deviceHardware2,
	}

	// All Devices
	testDevices = []Device{
		d1, d2, d3, d4, d5,
	}

	emptyResult = map[string]interface{}{}
)
