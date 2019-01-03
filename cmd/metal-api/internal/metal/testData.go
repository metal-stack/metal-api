package metal

var (
	D1 = Device{
		Base:   Base{ID: "1"},
		SiteID: "1",
		Site:   Site1,
		SizeID: "1",
		Size:   &Sz1,
		Allocation: &DeviceAllocation{
			Name:    "d1",
			ImageID: "1",
			Project: "p1",
		},
	}
	D2 = Device{
		Base:   Base{ID: "2"},
		SiteID: "1",
		Site:   Site1,
		SizeID: "1",
		Size:   &Sz1,
		Allocation: &DeviceAllocation{
			Name:    "d2",
			ImageID: "1",
			Project: "p2",
		},
	}
	D2_without_alloc = Device{
		Base:       Base{ID: "2"},
		SiteID:     "1",
		Site:       Site1,
		SizeID:     "1",
		Size:       &Sz1,
		Allocation: nil,
	}
	D3 = Device{
		Base:   Base{ID: "3"},
		SiteID: "1",
		Site:   Site1,
		SizeID: "1",
		Size:   &Sz1,
	}

	Sz1 = Size{
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
	Sz2 = Size{
		Base: Base{
			ID:          "2",
			Name:        "sz2",
			Description: "description 2",
		},
	}
	Img1 = Image{
		Base: Base{
			ID:          "1",
			Name:        "img1",
			Description: "description 1",
		},
	}
	Img2 = Image{
		Base: Base{
			ID:          "2",
			Name:        "img2",
			Description: "description 2",
		},
	}
	Site1 = Site{
		Base: Base{
			ID:          "1",
			Name:        "site1",
			Description: "description 1",
		},
	}
	Site2 = Site{
		Base: Base{
			ID:          "2",
			Name:        "site2",
			Description: "description 2",
		},
	}
	Switch1 = Switch{
		Base: Base{
			ID: "switch1",
		},
		SiteID: "1",
		RackID: "rack1",
		DeviceConnections: ConnectionMap{
			"d1": Connections{
				Connection{
					Nic: Nic{
						MacAddress: MacAddress("11:11:11"),
					},
					DeviceID: "d1",
				},
			},
		},
	}
	Switch2 = Switch{
		Base: Base{
			ID: "switch2",
		},
		SiteID:            "1",
		RackID:            "rack1",
		DeviceConnections: ConnectionMap{},
	}
	Switch3 = Switch{
		Base: Base{
			ID: "switch3",
		},
		SiteID:            "1",
		RackID:            "rack1",
		DeviceConnections: ConnectionMap{},
	}
	Nic1 = Nic{
		MacAddress: MacAddress("11:11:11"),
		Name:       "swp1",
		Neighbors:  nil,
	}
	Nic2 = Nic{
		MacAddress: MacAddress("21:11:11"),
		Name:       "swp2",
		Neighbors:  nil,
	}
	Nic3 = Nic{
		MacAddress: MacAddress("31:11:11"),
		Name:       "swp3",
		Neighbors:  nil,
	}
	Nics1 = Nics{
		Nic1, Nic2, Nic3,
	}
	DeviceHardware1 = DeviceHardware{
		Memory:   100,
		CPUCores: 1,
		Nics:     Nics1,
		Disks: []BlockDevice{
			{
				Name: "blockdeviceName",
				Size: 1000000000000,
			},
		},
	}
	DeviceHardware2 = DeviceHardware{
		Memory:   1000,
		CPUCores: 2,
		Nics:     Nics1,
		Disks: []BlockDevice{
			{
				Name: "blockdeviceName",
				Size: 2000000000000,
			},
		},
	}
	TestImageArray = []Image{
		Img1, Img2,
	}
	TestSizeArray = []Size{
		Sz1, Sz2,
	}
	TestSiteArray = []Site{
		Site1, Site2,
	}
	EmptyResult = map[string]interface{}{}
	IPMI1       = IPMI{
		ID:         "IPMI-1",
		Address:    "192.168.0.1",
		MacAddress: "11:11:11",
		User:       "User",
		Password:   "Password",
		Interface:  "Interface",
	}
)

func prepareTests() {
	Nic1.Neighbors = Nics{Nic2, Nic3}
	Nic2.Neighbors = Nics{Nic1, Nic3}
	Nic3.Neighbors = Nics{Nic1, Nic2}
}
