package testdata

import (
	"fmt"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

// If you want to add some Test Data, add it also to the following places:
// -- To the Mocks, ==> eof
// -- To the corrisponding lists,

// Also run the Tests:
// cd ./cloud-native/metal/metal-api/
// go test ./...
// -- OR
// go test -cover ./...
// -- OR
// cd ./PACKAGE
// go test -coverprofile=cover.out ./...
// go tool cover -func=cover.out					// Console Output
// (go tool cover -html=cover.out -o cover.html) 	// Html output

var Testlogger = zap.NewNop()

var TestloggerSugar = zapup.MustRootLogger().Sugar()

var (
	// Machines
	M1 = metal.Machine{
		Base:        metal.Base{ID: "1"},
		PartitionID: "1",
		Partition:   Partition1,
		SizeID:      "1",
		Size:        &Sz1,
		Allocation: &metal.MachineAllocation{
			Name:    "d1",
			ImageID: "1",
			Project: "p1",
		},
	}
	M2 = metal.Machine{
		Base:        metal.Base{ID: "2"},
		PartitionID: "1",
		Partition:   Partition1,
		SizeID:      "1",
		Size:        &Sz1,
		Allocation: &metal.MachineAllocation{
			Name:    "d2",
			ImageID: "1",
			Project: "p2",
		},
	}
	M3 = metal.Machine{
		Base:        metal.Base{ID: "3"},
		PartitionID: "1",
		Partition:   Partition1,
		SizeID:      "1",
		Size:        &Sz1,
	}
	M4 = metal.Machine{
		Base:        metal.Base{ID: "4"},
		PartitionID: "1",
		Partition:   Partition1,
		SizeID:      "1",
		Size:        &Sz1,
		Allocation:  nil,
	}
	M5 = metal.Machine{
		Base: metal.Base{
			Name:        "1-core/100 B",
			Description: "a machine with 1 core(s) and 100 B of RAM",
			ID:          "5",
		},
		RackID:      "1",
		PartitionID: "1",
		Partition:   Partition1,
		SizeID:      "1",
		Size:        &Sz1,
		Allocation:  nil,
		Hardware:    MachineHardware1,
	}
	M6 = metal.Machine{
		Base: metal.Base{
			Name:        "1-core/100 B",
			Description: "a machine with 1 core(s) and 100 B of RAM",
			ID:          "6",
		},
		RackID:      "1",
		PartitionID: "999",
		Partition:   Partition1,
		SizeID:      "1", // No Size
		Size:        nil,
		Allocation: &metal.MachineAllocation{
			Name:    "6",
			ImageID: "999",
			Project: "p2",
		},
		Hardware: MachineHardware1,
	}
	M7 = metal.Machine{
		Base: metal.Base{
			Name:        "1-core/100 B",
			Description: "a machine with 1 core(s) and 100 B of RAM",
			ID:          "6",
		},
		RackID:      "1",
		PartitionID: "1",
		Partition:   Partition1,
		SizeID:      "999", // No Size
		Size:        nil,
		Allocation: &metal.MachineAllocation{
			Name:    "7",
			ImageID: "1",
			Project: "p2",
		},
		Hardware: MachineHardware1,
	}
	M8 = metal.Machine{
		Base: metal.Base{
			Name:        "1-core/100 B",
			Description: "a machine with 1 core(s) and 100 B of RAM",
			ID:          "6",
		},
		RackID:      "1",
		PartitionID: "1",
		Partition:   Partition1,
		SizeID:      "1", // No Size
		Size:        nil,
		Allocation: &metal.MachineAllocation{
			Name:    "8",
			ImageID: "999",
			Project: "p2",
		},
		Hardware: MachineHardware1,
	}

	// Sizes
	Sz1 = metal.Size{
		Base: metal.Base{
			ID:          "1",
			Name:        "sz1",
			Description: "description 1",
		},
		Constraints: []metal.Constraint{
			metal.Constraint{
				Type: metal.CoreConstraint,
				Min:  1,
				Max:  1,
			},
			metal.Constraint{
				Type: metal.MemoryConstraint,
				Min:  100,
				Max:  100,
			},
		},
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
		Constraints: []metal.Constraint{
			metal.Constraint{
				Type: metal.CoreConstraint,
				Min:  888,
				Max:  1111,
			},
			metal.Constraint{
				Type: metal.MemoryConstraint,
				Min:  100,
				Max:  100,
			},
		},
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

	// partitions
	Partition1 = metal.Partition{
		Base: metal.Base{
			ID:          "1",
			Name:        "partition1",
			Description: "description 1",
		},
	}
	Partition2 = metal.Partition{
		Base: metal.Base{
			ID:          "2",
			Name:        "partition2",
			Description: "description 2",
		},
	}
	Partition3 = metal.Partition{
		Base: metal.Base{
			ID:          "3",
			Name:        "partition3",
			Description: "description 3",
		},
	}

	// Switches
	Switch1 = metal.Switch{
		Base: metal.Base{
			ID: "switch1",
		},
		PartitionID: "1",
		RackID:      "1",
		Nics: []metal.Nic{
			Nic1,
			Nic2,
			Nic3,
		},
		MachineConnections: metal.ConnectionMap{
			"1": metal.Connections{
				metal.Connection{
					Nic: metal.Nic{
						MacAddress: metal.MacAddress("11:11:11:11:11:11"),
					},
					MachineID: "1",
				},
				metal.Connection{
					Nic: metal.Nic{
						MacAddress: metal.MacAddress("11:11:11:11:11:22"),
					},
					MachineID: "1",
				},
			},
		},
	}
	Switch2 = metal.Switch{
		Base: metal.Base{
			ID: "switch2",
		},
		PartitionID:        "1",
		RackID:             "2",
		MachineConnections: metal.ConnectionMap{},
	}
	Switch3 = metal.Switch{
		Base: metal.Base{
			ID: "switch3",
		},
		PartitionID:        "1",
		RackID:             "3",
		MachineConnections: metal.ConnectionMap{},
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
		Fru: metal.Fru{
			ChassisPartNumber:   "CSE-217BHQ+-R2K22BP2",
			ChassisPartSerial:   "C217BAH31AG0535",
			BoardMfg:            "Supermicro",
			BoardMfgSerial:      "HM188S012423",
			BoardPartNumber:     "X11DPT-B",
			ProductManufacturer: "Supermicro",
			ProductPartNumber:   "SYS-2029BT-HNTR",
			ProductSerial:       "A328789X9108135",
		},
	}

	// MachineHardwares
	MachineHardware1 = metal.MachineHardware{
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
	MachineHardware2 = metal.MachineHardware{
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

	// All Events
	TestEvents = []metal.ProvisioningEventContainer{}

	// All Sizes
	TestSizes = []metal.Size{
		Sz1, Sz2, Sz3,
	}

	// All partitions
	TestPartitions = []metal.Partition{
		Partition1, Partition2, Partition3,
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
			MachineID: "machine-1",
		},
		metal.Connection{
			Nic: metal.Nic{
				Name:       "swp2",
				MacAddress: "22:11:11",
			},
			MachineID: "machine-2",
		},
	}

	TestMachinesHardwares = []metal.MachineHardware{
		MachineHardware1, MachineHardware2,
	}

	// All Machines
	TestMachines = []metal.Machine{
		M1, M2, M3, M4, M5,
	}

	// All Vrfs
	Vrf1 = metal.Vrf{
		ID:        1,
		ProjectID: "p",
		Tenant:    "t",
	}
	TestVrfs = []metal.Vrf{
		Vrf1,
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

	// X.Get(i)
	mock.On(r.DB("mockdb").Table("size").Get("1")).Return(Sz1, nil)
	mock.On(r.DB("mockdb").Table("size").Get("2")).Return(Sz2, nil)
	mock.On(r.DB("mockdb").Table("size").Get("3")).Return(Sz3, nil)
	mock.On(r.DB("mockdb").Table("size").Get("404")).Return(nil, fmt.Errorf("Test Error"))
	mock.On(r.DB("mockdb").Table("size").Get("999")).Return(nil, nil)
	mock.On(r.DB("mockdb").Table("partition").Get("1")).Return(Partition1, nil)
	mock.On(r.DB("mockdb").Table("partition").Get("2")).Return(Partition2, nil)
	mock.On(r.DB("mockdb").Table("partition").Get("3")).Return(Partition3, nil)
	mock.On(r.DB("mockdb").Table("partition").Get("404")).Return(nil, fmt.Errorf("Test Error"))
	mock.On(r.DB("mockdb").Table("partition").Get("999")).Return(nil, nil)
	mock.On(r.DB("mockdb").Table("image").Get("1")).Return(Img1, nil)
	mock.On(r.DB("mockdb").Table("image").Get("2")).Return(Img2, nil)
	mock.On(r.DB("mockdb").Table("image").Get("3")).Return(Img3, nil)
	mock.On(r.DB("mockdb").Table("image").Get("404")).Return(nil, fmt.Errorf("Test Error"))
	mock.On(r.DB("mockdb").Table("image").Get("999")).Return(nil, nil)
	mock.On(r.DB("mockdb").Table("machine").Get("1")).Return(M1, nil)
	mock.On(r.DB("mockdb").Table("machine").Get("2")).Return(M2, nil)
	mock.On(r.DB("mockdb").Table("machine").Get("3")).Return(M3, nil)
	mock.On(r.DB("mockdb").Table("machine").Get("4")).Return(M4, nil)
	mock.On(r.DB("mockdb").Table("machine").Get("5")).Return(M5, nil)
	mock.On(r.DB("mockdb").Table("machine").Get("6")).Return(M6, nil)
	mock.On(r.DB("mockdb").Table("machine").Get("7")).Return(M7, nil)
	mock.On(r.DB("mockdb").Table("machine").Get("8")).Return(M8, nil)
	mock.On(r.DB("mockdb").Table("machine").Get("404")).Return(nil, fmt.Errorf("Test Error"))
	mock.On(r.DB("mockdb").Table("machine").Get("999")).Return(nil, nil)
	mock.On(r.DB("mockdb").Table("switch").Get("switch1")).Return(Switch1, nil)
	mock.On(r.DB("mockdb").Table("switch").Get("switch2")).Return(Switch2, nil)
	mock.On(r.DB("mockdb").Table("switch").Get("switch3")).Return(Switch3, nil)
	mock.On(r.DB("mockdb").Table("switch").Get("switch404")).Return(nil, fmt.Errorf("Test Error"))
	mock.On(r.DB("mockdb").Table("switch").Get("switch999")).Return(nil, nil)
	mock.On(r.DB("mockdb").Table("ipmi").Get("IPMI-1")).Return(IPMI1, nil)
	mock.On(r.DB("mockdb").Table("wait").Get("3").Changes()).Return([]interface{}{
		map[string]interface{}{"new_val": M3},
	}, nil)

	// Default: Return Empty result
	mock.On(r.DB("mockdb").Table("size").Get(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("partition").Get(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("machine").Get(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("image").Get(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("switch").Get(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("ipmi").Get(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("event").Get(r.MockAnything())).Return(EmptyResult, nil)

	// X.GetTable
	mock.On(r.DB("mockdb").Table("size")).Return(TestSizes, nil)
	mock.On(r.DB("mockdb").Table("partition")).Return(TestPartitions, nil)
	mock.On(r.DB("mockdb").Table("image")).Return(TestImages, nil)
	mock.On(r.DB("mockdb").Table("machine")).Return(TestMachines, nil)
	mock.On(r.DB("mockdb").Table("switch")).Return(TestSwitches, nil)
	mock.On(r.DB("mockdb").Table("event")).Return(TestEvents, nil)
	mock.On(r.DB("mockdb").Table("vrf")).Return(TestVrfs, nil)

	// X.Delete
	mock.On(r.DB("mockdb").Table("machine").Get(r.MockAnything()).Delete()).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("image").Get(r.MockAnything()).Delete()).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("partition").Get(r.MockAnything()).Delete()).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("size").Get(r.MockAnything()).Delete()).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("ipmi").Get(r.MockAnything()).Delete()).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("switch").Get(r.MockAnything()).Delete()).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("wait").Get(r.MockAnything()).Delete()).Return(EmptyResult, nil)

	// X.Get.Replace
	mock.On(r.DB("mockdb").Table("machine").Get(r.MockAnything()).Replace(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("image").Get(r.MockAnything()).Replace(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("partition").Get(r.MockAnything()).Replace(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("size").Get(r.MockAnything()).Replace(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("ipmi").Get(r.MockAnything()).Replace(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("switch").Get(r.MockAnything()).Replace(r.MockAnything())).Return(EmptyResult, nil)

	// X.insert
	mock.On(r.DB("mockdb").Table("machine").Insert(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("image").Insert(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("partition").Insert(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("size").Insert(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("ipmi").Insert(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("switch").Insert(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("wait").Insert(r.MockAnything())).Return(EmptyResult, nil)

	mock.On(r.DB("mockdb").Table("machine").Insert(r.MockAnything(), r.InsertOpts{
		Conflict: "replace",
	})).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("image").Insert(r.MockAnything(), r.InsertOpts{
		Conflict: "replace",
	})).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("partition").Insert(r.MockAnything(), r.InsertOpts{
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
	mock.On(r.DB("mockdb").Table("wait").Insert(r.MockAnything(), r.InsertOpts{
		Conflict: "replace",
	})).Return(EmptyResult, nil)

	return
}
