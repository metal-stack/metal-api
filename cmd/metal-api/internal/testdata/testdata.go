package testdata

import (
	"fmt"
	"github.com/metal-stack/metal-lib/pkg/tag"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-lib/zapup"
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
		SizeID:      "1",
		Allocation: &metal.MachineAllocation{
			Name:    "d1",
			ImageID: "1",
			Project: "p1",
		},
		IPMI: IPMI1,
		Tags: []string{"1"},
	}
	M2 = metal.Machine{
		Base:        metal.Base{ID: "2"},
		PartitionID: "1",
		SizeID:      "1",
		Allocation: &metal.MachineAllocation{
			Name:    "d2",
			ImageID: "1",
			Project: "p2",
		},
	}
	M3 = metal.Machine{
		Base:        metal.Base{ID: "3"},
		PartitionID: "1",
		SizeID:      "1",
	}
	M4 = metal.Machine{
		Base:        metal.Base{ID: "4"},
		PartitionID: "1",
		SizeID:      "1",
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
		SizeID:      "1",
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
		SizeID:      "1", // No Size
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
		SizeID:      "999", // No Size
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
		SizeID:      "1", // No Size
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
			{
				Type: metal.CoreConstraint,
				Min:  1,
				Max:  1,
			},
			{
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
			{
				Type: metal.CoreConstraint,
				Min:  888,
				Max:  1111,
			},
			{
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
		URL: "http://somewhere/image1.zip",
	}
	Img2 = metal.Image{
		Base: metal.Base{
			ID:          "2",
			Name:        "Image 2",
			Description: "description 2",
		},
		URL: "http://somewhere/image2.zip",
	}
	Img3 = metal.Image{
		Base: metal.Base{
			ID:          "3",
			Name:        "Image 3",
			Description: "description 3",
		},
	}

	// Networks
	prefix1    = metal.Prefix{IP: "185.1.2.0", Length: "26"}
	prefix2    = metal.Prefix{IP: "100.64.2.0", Length: "16"}
	prefix3    = metal.Prefix{IP: "192.0.0.0", Length: "16"}
	prefixIPAM = metal.Prefix{IP: "10.0.0.0", Length: "16"}

	prefixes1    = []metal.Prefix{prefix1, prefix2}
	prefixes2    = []metal.Prefix{prefix2}
	prefixes3    = []metal.Prefix{prefix3}
	prefixesIPAM = []metal.Prefix{prefixIPAM}

	Nw1 = metal.Network{
		Base: metal.Base{
			ID:          "1",
			Name:        "Network 1",
			Description: "description 1",
		},
		PartitionID:  Partition1.ID,
		Prefixes:     prefixes1,
		PrivateSuper: true,
	}
	Nw2 = metal.Network{
		Base: metal.Base{
			ID:          "2",
			Name:        "Network 2",
			Description: "description 2",
		},
		Prefixes: prefixes2,
		Underlay: true,
	}
	Nw3 = metal.Network{
		Base: metal.Base{
			ID:          "3",
			Name:        "Network 3",
			Description: "description 3",
		},
		Prefixes:        prefixes3,
		PartitionID:     Partition1.ID,
		ParentNetworkID: Nw1.ID,
	}

	Partition1PrivateSuperNetwork = metal.Network{
		Base: metal.Base{
			ID: "super-tenant-network-1",
		},
		Prefixes:        metal.Prefixes{{IP: "10.0.0.0", Length: "16"}},
		PartitionID:     Partition1.ID,
		ParentNetworkID: "",
		ProjectID:       "",
		PrivateSuper:    true,
		Nat:             false,
		Underlay:        false,
	}

	Partition2PrivateSuperNetwork = metal.Network{
		Base: metal.Base{
			ID: "super-tenant-network-2",
		},
		Prefixes:        metal.Prefixes{{IP: "10.3.0.0", Length: "16"}},
		PartitionID:     Partition2.ID,
		ParentNetworkID: "",
		ProjectID:       "",
		PrivateSuper:    true,
		Nat:             false,
		Underlay:        false,
	}

	Partition1UnderlayNetwork = metal.Network{
		Base: metal.Base{
			ID: "underlay-network-1",
		},
		Prefixes:        metal.Prefixes{{IP: "10.1.0.0", Length: "24"}},
		PartitionID:     Partition1.ID,
		ParentNetworkID: "",
		ProjectID:       "",
		PrivateSuper:    false,
		Nat:             false,
		Underlay:        true,
	}

	Partition2UnderlayNetwork = metal.Network{
		Base: metal.Base{
			ID: "underlay-network-1",
		},
		Prefixes:        metal.Prefixes{{IP: "10.2.0.0", Length: "24"}},
		PartitionID:     Partition2.ID,
		ParentNetworkID: "",
		ProjectID:       "",
		PrivateSuper:    false,
		Nat:             false,
		Underlay:        true,
	}

	Partition1InternetNetwork = metal.Network{
		Base: metal.Base{
			ID: "internet-network-1",
		},
		Prefixes:        metal.Prefixes{{IP: "212.34.0.0", Length: "24"}},
		PartitionID:     Partition1.ID,
		ParentNetworkID: "",
		ProjectID:       "",
		PrivateSuper:    false,
		Nat:             true,
		Underlay:        false,
	}

	Partition2InternetNetwork = metal.Network{
		Base: metal.Base{
			ID: "internet-network-2",
		},
		Prefixes:        metal.Prefixes{{IP: "212.35.0.0", Length: "24"}},
		PartitionID:     Partition2.ID,
		ParentNetworkID: "",
		ProjectID:       "",
		PrivateSuper:    false,
		Nat:             true,
		Underlay:        false,
	}

	Partition1ExistingPrivateNetwork = metal.Network{
		Base: metal.Base{
			ID: "existing-private-network-1",
		},
		Prefixes:        metal.Prefixes{{IP: "10.0.204.0", Length: "22"}},
		PartitionID:     Partition1.ID,
		ParentNetworkID: Partition1PrivateSuperNetwork.ID,
		ProjectID:       "project-uuid",
		PrivateSuper:    false,
		Nat:             false,
		Underlay:        false,
	}

	Partition2ExistingPrivateNetwork = metal.Network{
		Base: metal.Base{
			ID: "existing-private-network-2",
		},
		Prefixes:        metal.Prefixes{{IP: "10.3.88.0", Length: "22"}},
		PartitionID:     Partition2.ID,
		ParentNetworkID: Partition2PrivateSuperNetwork.ID,
		ProjectID:       "project-uuid",
		PrivateSuper:    false,
		Nat:             false,
		Underlay:        false,
	}

	NwIPAM = metal.Network{
		Base: metal.Base{
			ID:          "4",
			Name:        "IPAM Network",
			Description: "description IPAM",
		},
		Prefixes: prefixesIPAM,
	}

	// IPs
	IP1 = metal.IP{
		IPAddress:   "1.2.3.4",
		Name:        "Image 1",
		Description: "description 1",
		Type:        "ephemeral",
		ProjectID:   "1",
	}
	IP2 = metal.IP{
		IPAddress:   "2.3.4.5",
		Name:        "Image 2",
		Description: "description 2",
		Type:        "static",
		ProjectID:   "1",
	}
	IP3 = metal.IP{
		IPAddress:   "3.4.5.6",
		Name:        "Image 3",
		Description: "description 3",
		Type:        "static",
		Tags:        []string{tag.MachineID},
		ProjectID:   "1",
	}
	IPAMIP = metal.IP{
		Name:        "IPAM IP",
		Description: "description IPAM",
		ProjectID:   "1",
	}
	Partition1InternetIP = metal.IP{
		IPAddress: "212.34.0.1",
		ProjectID: Partition1ExistingPrivateNetwork.ProjectID,
		NetworkID: Partition1InternetNetwork.ID,
	}
	Partition2InternetIP = metal.IP{
		IPAddress: "212.34.0.2",
		ProjectID: Partition1ExistingPrivateNetwork.ProjectID,
		NetworkID: Partition2InternetNetwork.ID,
	}

	// partitions
	Partition1 = metal.Partition{
		Base: metal.Base{
			ID:          "1",
			Name:        "partition1",
			Description: "description 1",
		},
		PrivateNetworkPrefixLength: 22,
	}
	Partition2 = metal.Partition{
		Base: metal.Base{
			ID:          "2",
			Name:        "partition2",
			Description: "description 2",
		},
		PrivateNetworkPrefixLength: 22,
	}
	Partition3 = metal.Partition{
		Base: metal.Base{
			ID:          "3",
			Name:        "partition3",
			Description: "description 3",
		},
		PrivateNetworkPrefixLength: 22,
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
			{
				MacAddress: "21:11:11:11:11:11",
			},
			{
				MacAddress: "31:11:11:11:11:11",
			},
		},
	}
	Nic2 = metal.Nic{
		MacAddress: metal.MacAddress("21:11:11:11:11:11"),
		Name:       "swp1",
		Neighbors: []metal.Nic{
			{
				MacAddress: "11:11:11:11:11:11",
			},
			{
				MacAddress: "31:11:11:11:11:11",
			},
		},
	}
	Nic3 = metal.Nic{
		MacAddress: metal.MacAddress("31:11:11:11:11:11"),
		Name:       "swp2",
		Neighbors: []metal.Nic{
			{
				MacAddress: "21:11:11:11:11:11",
			},
			{
				MacAddress: "11:11:11:11:11:11",
			},
		},
	}

	// IPMIs
	IPMI1 = metal.IPMI{
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
	// All Networks
	TestNetworks = []metal.Network{
		Nw1, Nw2, Nw3, NwIPAM,
		// Partition1ExistingPrivateNetwork,
		// Partition1InternetNetwork,
		// Partition1PrivateSuperNetwork,
		// Partition1UnderlayNetwork,
		// Partition2InternetNetwork,
		// Partition2PrivateSuperNetwork,
		// Partition2UnderlayNetwork,
	}
	// All IPs
	TestIPs = []metal.IP{
		IP1, IP2, IP3,
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
		{
			Nic: metal.Nic{
				Name:       "swp1",
				MacAddress: "11:11:11",
			},
			MachineID: "machine-1",
		},
		{
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
	TestMachines = metal.Machines{
		M1, M2, M3, M4, M5,
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
	mock.On(r.DB("mockdb").Table("network").Get(Nw1.ID)).Return(Nw1, nil)
	mock.On(r.DB("mockdb").Table("network").Get(Nw2.ID)).Return(Nw2, nil)
	mock.On(r.DB("mockdb").Table("network").Get(Nw3.ID)).Return(Nw3, nil)
	mock.On(r.DB("mockdb").Table("network").Get(Partition1ExistingPrivateNetwork.ID)).Return(Partition1ExistingPrivateNetwork, nil)
	mock.On(r.DB("mockdb").Table("network").Get(Partition1InternetNetwork.ID)).Return(Partition1InternetNetwork, nil)
	mock.On(r.DB("mockdb").Table("network").Get(Partition1PrivateSuperNetwork.ID)).Return(Partition1PrivateSuperNetwork, nil)
	mock.On(r.DB("mockdb").Table("network").Get(Partition1UnderlayNetwork.ID)).Return(Partition1UnderlayNetwork, nil)
	mock.On(r.DB("mockdb").Table("network").Get(Partition2ExistingPrivateNetwork.ID)).Return(Partition2ExistingPrivateNetwork, nil)
	mock.On(r.DB("mockdb").Table("network").Get(Partition2InternetNetwork.ID)).Return(Partition2InternetNetwork, nil)
	mock.On(r.DB("mockdb").Table("network").Get(Partition2PrivateSuperNetwork.ID)).Return(Partition2PrivateSuperNetwork, nil)
	mock.On(r.DB("mockdb").Table("network").Get(Partition2UnderlayNetwork.ID)).Return(Partition2UnderlayNetwork, nil)

	mock.On(r.DB("mockdb").Table("network").Get("404")).Return(nil, fmt.Errorf("Test Error"))
	mock.On(r.DB("mockdb").Table("network").Get("999")).Return(nil, nil)
	mock.On(r.DB("mockdb").Table("network").Filter(func(var_3 r.Term) r.Term { return var_3.Field("partitionid").Eq("1") }).Filter(func(var_4 r.Term) r.Term { return var_4.Field("privatesuper").Eq(true) })).Return(Nw3, nil)

	mock.On(r.DB("mockdb").Table("ip").Get("1.2.3.4")).Return(IP1, nil)
	mock.On(r.DB("mockdb").Table("ip").Get("2.3.4.5")).Return(IP2, nil)
	mock.On(r.DB("mockdb").Table("ip").Get("3.4.5.6")).Return(IP3, nil)
	mock.On(r.DB("mockdb").Table("ip").Get("8.8.8.8")).Return(nil, fmt.Errorf("Test Error"))
	mock.On(r.DB("mockdb").Table("ip").Get("9.9.9.9")).Return(nil, nil)
	mock.On(r.DB("mockdb").Table("ip").Get(Partition1InternetIP.IPAddress)).Return(Partition1InternetIP, nil)
	mock.On(r.DB("mockdb").Table("ip").Get(Partition2InternetIP.IPAddress)).Return(Partition2InternetIP, nil)
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
	mock.On(r.DB("mockdb").Table("machine").Filter(func(var_1 r.Term) r.Term {
		return var_1.Field("partitionid").Eq(Partition1.ID)
	})).Return(metal.Machines{M1, M2, M3, M4, M5, M7, M8}, nil)
	mock.On(r.DB("mockdb").Table("switch").Get("switch1")).Return(Switch1, nil)
	mock.On(r.DB("mockdb").Table("switch").Get("switch2")).Return(Switch2, nil)
	mock.On(r.DB("mockdb").Table("switch").Get("switch3")).Return(Switch3, nil)
	mock.On(r.DB("mockdb").Table("switch").Get("switch404")).Return(nil, fmt.Errorf("Test Error"))
	mock.On(r.DB("mockdb").Table("switch").Get("switch999")).Return(nil, nil)
	mock.On(r.DB("mockdb").Table("wait").Get("3").Changes()).Return([]interface{}{
		map[string]interface{}{"new_val": M3},
	}, nil)
	mock.On(r.DB("mockdb").Table("integerpool").Get(r.MockAnything()).Delete(r.DeleteOpts{ReturnChanges: true})).Return(r.WriteResponse{Changes: []r.ChangeResponse{r.ChangeResponse{OldValue: map[string]interface{}{"id": float64(12345)}}}}, nil)

	// Default: Return Empty result
	mock.On(r.DB("mockdb").Table("size").Get(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("partition").Get(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("machine").Get(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("image").Get(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("network").Get(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("ip").Get(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("switch").Get(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("project").Get(r.MockAnything())).Return(EmptyResult, nil)

	mock.On(r.DB("mockdb").Table("event").Get(r.MockAnything())).Return(EmptyResult, nil)

	// X.GetTable
	mock.On(r.DB("mockdb").Table("size")).Return(TestSizes, nil)
	mock.On(r.DB("mockdb").Table("partition")).Return(TestPartitions, nil)
	mock.On(r.DB("mockdb").Table("image")).Return(TestImages, nil)
	mock.On(r.DB("mockdb").Table("network")).Return(TestNetworks, nil)
	mock.On(r.DB("mockdb").Table("ip")).Return(TestIPs, nil)
	mock.On(r.DB("mockdb").Table("machine")).Return(TestMachines, nil)
	mock.On(r.DB("mockdb").Table("switch")).Return(TestSwitches, nil)
	mock.On(r.DB("mockdb").Table("event")).Return(TestEvents, nil)

	// X.Delete
	mock.On(r.DB("mockdb").Table("machine").Get(r.MockAnything()).Delete()).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("image").Get(r.MockAnything()).Delete()).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("network").Get(r.MockAnything()).Delete()).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("ip").Get(r.MockAnything()).Delete()).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("partition").Get(r.MockAnything()).Delete()).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("size").Get(r.MockAnything()).Delete()).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("switch").Get(r.MockAnything()).Delete()).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("wait").Get(r.MockAnything()).Delete()).Return(EmptyResult, nil)

	// X.Get.Replace
	mock.On(r.DB("mockdb").Table("machine").Get(r.MockAnything()).Replace(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("image").Get(r.MockAnything()).Replace(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("network").Get(r.MockAnything()).Replace(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("ip").Get(r.MockAnything()).Replace(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("partition").Get(r.MockAnything()).Replace(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("size").Get(r.MockAnything()).Replace(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("switch").Get(r.MockAnything()).Replace(r.MockAnything())).Return(EmptyResult, nil)

	// X.insert
	mock.On(r.DB("mockdb").Table("machine").Insert(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("image").Insert(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("network").Insert(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("ip").Insert(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("partition").Insert(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("size").Insert(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("switch").Insert(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("wait").Insert(r.MockAnything())).Return(EmptyResult, nil)

	mock.On(r.DB("mockdb").Table("machine").Insert(r.MockAnything(), r.InsertOpts{
		Conflict: "replace",
	})).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("image").Insert(r.MockAnything(), r.InsertOpts{
		Conflict: "replace",
	})).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("network").Insert(r.MockAnything(), r.InsertOpts{
		Conflict: "replace",
	})).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("ip").Insert(r.MockAnything(), r.InsertOpts{
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
	mock.On(r.DB("mockdb").Table("wait").Insert(r.MockAnything(), r.InsertOpts{
		Conflict: "replace",
	})).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("event").Insert(r.MockAnything(), r.InsertOpts{
		Conflict: "replace",
	})).Return(EmptyResult, nil)

	return
}
