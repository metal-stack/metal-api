package testdata

import (
	"errors"
	"time"

	"github.com/metal-stack/metal-lib/pkg/tag"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

var (
	// Machines
	M1 = metal.Machine{
		Base:        metal.Base{ID: "1"},
		PartitionID: "1",
		SizeID:      "1",
		Allocation: &metal.MachineAllocation{
			Name:    "d1",
			ImageID: "image-1",
			Project: "p1",
			Role:    metal.RoleMachine,
			MachineNetworks: []*metal.MachineNetwork{
				{
					NetworkID: "3",
					Private:   true,
					Vrf:       1,
				},
			},
		},
		Hardware: metal.MachineHardware{
			MetalCPUs: []metal.MetalCPU{
				{
					Model:   "Intel Xeon Silver",
					Cores:   8,
					Threads: 8,
				},
			},
			Memory: 1 << 30,
			Disks: []metal.BlockDevice{
				{
					Size: 1000,
				},
				{
					Size: 1000,
				},
				{
					Size: 1000,
				},
			},
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
			ImageID: "image-1",
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
			ImageID: "image-999",
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
			ImageID: "image-999",
			Project: "p2",
		},
		Hardware: MachineHardware1,
	}
	M9 = metal.Machine{
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
			ImageID: "ubuntu-19.10",
			Project: "p2",
		},
		Hardware: MachineHardware1,
	}
	M10 = metal.Machine{
		Base: metal.Base{
			Name:        "1-core/100 B",
			Description: "a machine with 1 core(s) and 100 B of RAM",
			ID:          "10",
		},
		RackID:      "1",
		PartitionID: "1",
		SizeID:      "1", // No Size
		Allocation: &metal.MachineAllocation{
			Name:    "10",
			ImageID: "image-4",
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
			{
				Type: metal.StorageConstraint,
				Min:  1000000000000,
				Max:  1000000000000,
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

	expireDate = time.Now().Add(time.Hour).Truncate(time.Second)
	// Images
	Img1 = metal.Image{
		Base: metal.Base{
			ID:          "image-1",
			Name:        "Image 1",
			Description: "description 1",
		},
		URL:            "http://images.metal-stack.io/metal-os/master/ubuntu/20.04/20200730/img.tar.lz4",
		OS:             "image",
		Version:        "1.0.0",
		ExpirationDate: expireDate,
		Classification: metal.ClassificationPreview,
	}
	Img2 = metal.Image{
		Base: metal.Base{
			ID:          "image-2",
			Name:        "Image 2",
			Description: "description 2",
		},
		URL:            "http://images.metal-stack.io/metal-os/master/ubuntu/20.04/20200730/img.tar.lz4",
		OS:             "image",
		Version:        "2.0.0",
		ExpirationDate: expireDate,
		Classification: metal.ClassificationSupported,
	}
	Img3 = metal.Image{
		Base: metal.Base{
			ID:          "image-3",
			Name:        "Image 3",
			Description: "description 3",
		},
		URL:            "http://images.metal-stack.io/metal-os/master/ubuntu/20.04/20200730/img.tar.lz4",
		OS:             "image",
		Version:        "3.0.0",
		ExpirationDate: expireDate,
		Classification: metal.ClassificationDeprecated,
	}
	Img4 = metal.Image{
		Base: metal.Base{
			ID:          "image-4",
			Name:        "Image 4",
			Description: "description 4",
		},
		URL: "http://images.metal-stack.io/metal-os/master/ubuntu/20.04/20200730/img.tar.lz4",
	}
	// Networks
	prefix1       = metal.Prefix{IP: "185.1.2.0", Length: "26"}
	prefix2       = metal.Prefix{IP: "100.64.0.0", Length: "16"}
	prefix3       = metal.Prefix{IP: "192.0.0.0", Length: "16"}
	prefixIPAM    = metal.Prefix{IP: "10.0.0.0", Length: "16"}
	superPrefix   = metal.Prefix{IP: "10.1.0.0", Length: "16"}
	superPrefixV6 = metal.Prefix{IP: "2001::", Length: "48"}
	cpl1          = metal.ChildPrefixLength{metal.IPv4AddressFamily: 28}
	cpl2          = metal.ChildPrefixLength{metal.IPv4AddressFamily: 22}

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
		PartitionID:              Partition1.ID,
		Prefixes:                 prefixes1,
		PrivateSuper:             true,
		DefaultChildPrefixLength: cpl1,
	}
	Nw2 = metal.Network{
		Base: metal.Base{
			ID:          "2",
			Name:        "Network 2",
			Description: "description 2",
		},
		PartitionID:              Partition1.ID,
		Prefixes:                 prefixes2,
		Underlay:                 true,
		DefaultChildPrefixLength: cpl2,
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
		Prefixes:                   metal.Prefixes{{IP: "10.0.0.0", Length: "16"}},
		PartitionID:                Partition1.ID,
		ParentNetworkID:            "",
		ProjectID:                  "",
		PrivateSuper:               true,
		Nat:                        false,
		Underlay:                   false,
		AdditionalAnnouncableCIDRs: []string{"10.240.0.0/12"},
	}

	Partition2PrivateSuperNetwork = metal.Network{
		Base: metal.Base{
			ID: "super-tenant-network-2",
		},
		Prefixes:                 metal.Prefixes{superPrefix},
		PartitionID:              Partition2.ID,
		DefaultChildPrefixLength: metal.ChildPrefixLength{metal.IPv4AddressFamily: 22},
		ParentNetworkID:          "",
		ProjectID:                "",
		PrivateSuper:             true,
		Nat:                      false,
		Underlay:                 false,
	}

	Partition2PrivateSuperNetworkV6 = metal.Network{
		Base: metal.Base{
			ID: "super-tenant-network-2-v6",
		},
		Prefixes:                 metal.Prefixes{superPrefixV6},
		PartitionID:              Partition2.ID,
		DefaultChildPrefixLength: metal.ChildPrefixLength{metal.IPv6AddressFamily: 64},
		ParentNetworkID:          "",
		ProjectID:                "",
		PrivateSuper:             true,
		Nat:                      false,
		Underlay:                 false,
	}

	Partition4PrivateSuperNetworkMixed = metal.Network{
		Base: metal.Base{
			ID: "super-tenant-network-2-mixed",
		},
		Prefixes:                 metal.Prefixes{superPrefix, superPrefixV6},
		PartitionID:              "4",
		DefaultChildPrefixLength: metal.ChildPrefixLength{metal.IPv4AddressFamily: 22, metal.IPv6AddressFamily: 64},
		ParentNetworkID:          "",
		ProjectID:                "",
		PrivateSuper:             true,
		Nat:                      false,
		Underlay:                 false,
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

	Partition1ExistingSharedNetwork = metal.Network{
		Base: metal.Base{
			ID: "existing-shared-network-1",
		},
		Prefixes:        metal.Prefixes{{IP: "10.0.205.0", Length: "22"}},
		PartitionID:     Partition1.ID,
		ParentNetworkID: Partition1PrivateSuperNetwork.ID,
		ProjectID:       "shared-uuid",
		PrivateSuper:    false,
		Nat:             false,
		Shared:          true,
	}

	Partition1ExistingSharedNetwork2 = metal.Network{
		Base: metal.Base{
			ID: "existing-shared-network-2",
		},
		Prefixes:        metal.Prefixes{{IP: "10.0.206.0", Length: "22"}},
		PartitionID:     Partition1.ID,
		ParentNetworkID: Partition1PrivateSuperNetwork.ID,
		ProjectID:       "shared-uuid2",
		PrivateSuper:    false,
		Nat:             false,
		Shared:          true,
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
		Prefixes:        prefixesIPAM,
	}

	// IPs
	IP1 = metal.IP{
		IPAddress:   "1.2.3.4",
		Name:        "Image 1",
		Description: "description 1",
		Type:        metal.Ephemeral,
		ProjectID:   "1",
	}
	IP2 = metal.IP{
		IPAddress:   "2.3.4.5",
		Name:        "Image 2",
		Description: "description 2",
		Type:        metal.Static,
		ProjectID:   "1",
	}
	IP3 = metal.IP{
		IPAddress:   "3.4.5.6",
		Name:        "Image 3",
		Description: "description 3",
		Type:        metal.Static,
		Tags:        []string{tag.MachineID},
		ProjectID:   "1",
	}
	IP4 = metal.IP{
		IPAddress:   "2001:0db8:85a3::1",
		Name:        "IPv6 4",
		Description: "description 4",
		Type:        metal.Ephemeral,
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
	Partition1SpecificSharedIP = metal.IP{
		IPAddress: "10.0.205.1",
		ProjectID: Partition1ExistingSharedNetwork.ProjectID,
		NetworkID: Partition1ExistingSharedNetwork.ID,
	}
	Partition1SpecificSharedConsumerIP = metal.IP{
		IPAddress: "10.0.205.2",
		ProjectID: Partition1ExistingPrivateNetwork.ProjectID,
		NetworkID: Partition1ExistingSharedNetwork.ID,
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
	Partition4 = metal.Partition{
		Base: metal.Base{
			ID:          "4",
			Name:        "partition4",
			Description: "description 4",
		},
	}
	// Switches
	Switch1 = metal.Switch{
		Base: metal.Base{
			ID: "switch1",
		},
		OS:          &metal.SwitchOS{Vendor: metal.SwitchOSVendorCumulus},
		PartitionID: "1",
		RackID:      "1",
		Nics: []metal.Nic{
			Nic2,
			Nic3,
		},
		MachineConnections: metal.ConnectionMap{
			"1": metal.Connections{
				metal.Connection{
					Nic: metal.Nic{
						Name:       "swp2",
						MacAddress: metal.MacAddress("21:11:11:11:11:11"),
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
		OS:          &metal.SwitchOS{Vendor: metal.SwitchOSVendorCumulus},
		PartitionID: "1",
		RackID:      "1",
		Nics: []metal.Nic{
			Nic4,
		},
		MachineConnections: metal.ConnectionMap{},
	}
	Switch3 = metal.Switch{
		Base: metal.Base{
			ID: "switch3",
		},
		OS:                 &metal.SwitchOS{Vendor: metal.SwitchOSVendorCumulus},
		PartitionID:        "1",
		RackID:             "3",
		MachineConnections: metal.ConnectionMap{},
	}
	SwitchReplaceFor1 = metal.Switch{
		Base: metal.Base{
			ID: "switch1",
		},
		OS:          &metal.SwitchOS{Vendor: metal.SwitchOSVendorCumulus},
		PartitionID: "1",
		RackID:      "1",
		Nics: []metal.Nic{
			metal.Nic{
				Name:       Nic1.Name,
				MacAddress: "98:98:98:98:98:91",
			},
			metal.Nic{
				Name:       Nic2.Name,
				MacAddress: "98:98:98:98:98:92",
			},
			metal.Nic{
				Name:       Nic3.Name,
				MacAddress: "98:98:98:98:98:93",
			},
		},
		MachineConnections: metal.ConnectionMap{},
	}

	// Nics
	Nic1 = metal.Nic{
		MacAddress: metal.MacAddress("11:11:11:11:11:11"),
		Name:       "swp1",
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
		Name:       "swp2",
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
		Name:       "swp3",
		Neighbors: []metal.Nic{
			{
				MacAddress: "21:11:11:11:11:11",
			},
			{
				MacAddress: "11:11:11:11:11:11",
			},
		},
	}
	Nic4 = metal.Nic{
		MacAddress: metal.MacAddress("41:11:11:11:11:11"),
		Name:       "swp2",
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
		Memory: 100,
		MetalCPUs: []metal.MetalCPU{
			{
				Model:   "Intel Xeon Silver",
				Cores:   1,
				Threads: 1,
			},
		},
		Nics: TestNics,
		Disks: []metal.BlockDevice{
			{
				Name: "blockdeviceName",
				Size: 1000000000000,
			},
		},
	}
	MachineHardware2 = metal.MachineHardware{
		Memory: 1000,
		MetalCPUs: []metal.MetalCPU{
			{
				Model:   "Intel Xeon Silver",
				Cores:   2,
				Threads: 2,
			},
		},
		Nics: TestNics,
		Disks: []metal.BlockDevice{
			{
				Name: "blockdeviceName",
				Size: 2000000000000,
			},
		},
	}

	// All Images
	TestImages = []metal.Image{
		Img1, Img2, Img3, Img4,
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
		IP1, IP2, IP3, IP4,
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
		Nic1, Nic2, Nic3, Nic4,
	}

	// All Switches
	TestSwitches = metal.Switches{
		Switch1, Switch2, Switch3, SwitchReplaceFor1,
	}
	TestSwitchStates = []metal.SwitchStatus{}
	TestMacs         = []string{
		"11:11:11:11:11:11",
		"11:11:11:11:11:22",
		"11:11:11:11:11:33",
		"11:11:11:11:11:11",
		"22:11:11:11:11:11",
		"33:11:11:11:11:11",
	}

	TestMachinesHardwares = []metal.MachineHardware{
		MachineHardware1, MachineHardware2,
	}

	// All Machines
	TestMachines = metal.Machines{
		M1, M2, M3, M4, M5,
	}

	EmptyResult = map[string]any{}
)

/*
InitMockDBData ...

Description:
This Function initializes the Data of a mocked rethink DB.
To get a Mocked r, execute datastore.InitMockDB().
If custom mocks should be used instead of here defined mocks, they can be added to the Mock Object before this function is called.

Hints for modifying this function:
If there need to be additional mocks added, they should be added before default mocks, which contain "r.MockAnything()"

Parameters:
- Mock 			// The Mock endpoint (Used for mocks)
*/
func InitMockDBData(mock *r.Mock) {

	// X.Get(i)
	mock.On(r.DB("mockdb").Table("size").Get("1")).Return(Sz1, nil)
	mock.On(r.DB("mockdb").Table("size").Get("2")).Return(Sz2, nil)
	mock.On(r.DB("mockdb").Table("size").Get("3")).Return(Sz3, nil)
	mock.On(r.DB("mockdb").Table("size").Get("404")).Return(nil, errors.New("Test Error"))
	mock.On(r.DB("mockdb").Table("size").Get("999")).Return(nil, nil)
	mock.On(r.DB("mockdb").Table("partition").Get("1")).Return(Partition1, nil)
	mock.On(r.DB("mockdb").Table("partition").Get("2")).Return(Partition2, nil)
	mock.On(r.DB("mockdb").Table("partition").Get("3")).Return(Partition3, nil)
	mock.On(r.DB("mockdb").Table("partition").Get("4")).Return(Partition4, nil)

	mock.On(r.DB("mockdb").Table("partition").Get("404")).Return(nil, errors.New("Test Error"))
	mock.On(r.DB("mockdb").Table("partition").Get("999")).Return(nil, nil)
	mock.On(r.DB("mockdb").Table("image").Get("image-1")).Return(Img1, nil)
	mock.On(r.DB("mockdb").Table("image").Get("image-2")).Return(Img2, nil)
	mock.On(r.DB("mockdb").Table("image").Get("image-3")).Return(Img3, nil)
	mock.On(r.DB("mockdb").Table("image").Get("image-4")).Return(Img4, nil)
	mock.On(r.DB("mockdb").Table("image").Get("404")).Return(nil, errors.New("Test Error"))
	mock.On(r.DB("mockdb").Table("image").Get("image-999")).Return(nil, nil)
	mock.On(r.DB("mockdb").Table("network").Get(Nw1.ID)).Return(Nw1, nil)
	mock.On(r.DB("mockdb").Table("network").Get(Nw2.ID)).Return(Nw2, nil)
	mock.On(r.DB("mockdb").Table("network").Get(Nw3.ID)).Return(Nw3, nil)
	mock.On(r.DB("mockdb").Table("network").Get(Partition1ExistingPrivateNetwork.ID)).Return(Partition1ExistingPrivateNetwork, nil)
	mock.On(r.DB("mockdb").Table("network").Get(Partition1ExistingSharedNetwork.ID)).Return(Partition1ExistingSharedNetwork, nil)
	mock.On(r.DB("mockdb").Table("network").Get(Partition1ExistingSharedNetwork2.ID)).Return(Partition1ExistingSharedNetwork2, nil)
	mock.On(r.DB("mockdb").Table("network").Get(Partition1InternetNetwork.ID)).Return(Partition1InternetNetwork, nil)
	mock.On(r.DB("mockdb").Table("network").Get(Partition1PrivateSuperNetwork.ID)).Return(Partition1PrivateSuperNetwork, nil)
	mock.On(r.DB("mockdb").Table("network").Get(Partition1UnderlayNetwork.ID)).Return(Partition1UnderlayNetwork, nil)
	mock.On(r.DB("mockdb").Table("network").Get(Partition2ExistingPrivateNetwork.ID)).Return(Partition2ExistingPrivateNetwork, nil)
	mock.On(r.DB("mockdb").Table("network").Get(Partition2InternetNetwork.ID)).Return(Partition2InternetNetwork, nil)
	mock.On(r.DB("mockdb").Table("network").Get(Partition2PrivateSuperNetwork.ID)).Return(Partition2PrivateSuperNetwork, nil)
	mock.On(r.DB("mockdb").Table("network").Get(Partition2UnderlayNetwork.ID)).Return(Partition2UnderlayNetwork, nil)
	mock.On(r.DB("mockdb").Table("network").Get(Partition2PrivateSuperNetworkV6.ID)).Return(Partition2PrivateSuperNetworkV6, nil)

	mock.On(r.DB("mockdb").Table("network").Get("404")).Return(nil, errors.New("Test Error"))
	mock.On(r.DB("mockdb").Table("network").Get("999")).Return(nil, nil)
	mock.On(r.DB("mockdb").Table("network").Filter(
		func(var_3 r.Term) r.Term { return var_3.Field("partitionid").Eq("1") }).Filter(
		func(var_4 r.Term) r.Term { return var_4.Field("privatesuper").Eq(true) })).Return(Nw3, nil)
	mock.On(r.DB("mockdb").Table("network").Filter(
		func(var_3 r.Term) r.Term { return var_3.Field("partitionid").Eq("2") }).Filter(
		func(var_4 r.Term) r.Term { return var_4.Field("privatesuper").Eq(true) })).Return(Partition2PrivateSuperNetwork, nil)
	mock.On(r.DB("mockdb").Table("network").Filter(
		func(var_3 r.Term) r.Term { return var_3.Field("partitionid").Eq("3") }).Filter(
		func(var_4 r.Term) r.Term { return var_4.Field("privatesuper").Eq(true) })).Return(nil, nil)
	mock.On(r.DB("mockdb").Table("network").Filter(
		func(var_3 r.Term) r.Term { return var_3.Field("partitionid").Eq("4") }).Filter(
		func(var_4 r.Term) r.Term { return var_4.Field("privatesuper").Eq(true) })).Return(Partition4PrivateSuperNetworkMixed, nil)

	mock.On(r.DB("mockdb").Table("ip").Get("1.2.3.4")).Return(IP1, nil)
	mock.On(r.DB("mockdb").Table("ip").Get("2.3.4.5")).Return(IP2, nil)
	mock.On(r.DB("mockdb").Table("ip").Get("3.4.5.6")).Return(IP3, nil)
	mock.On(r.DB("mockdb").Table("ip").Get("8.8.8.8")).Return(nil, errors.New("Test Error"))
	mock.On(r.DB("mockdb").Table("ip").Get("9.9.9.9")).Return(nil, nil)
	mock.On(r.DB("mockdb").Table("ip").Get("2001:0db8:85a3::1")).Return(IP4, nil)
	mock.On(r.DB("mockdb").Table("ip").Get(Partition1InternetIP.IPAddress)).Return(Partition1InternetIP, nil)
	mock.On(r.DB("mockdb").Table("ip").Get(Partition2InternetIP.IPAddress)).Return(Partition2InternetIP, nil)
	mock.On(r.DB("mockdb").Table("ip").Get(Partition1SpecificSharedIP.IPAddress)).Return(Partition1SpecificSharedIP, nil)
	mock.On(r.DB("mockdb").Table("ip").Get(Partition1SpecificSharedConsumerIP.IPAddress)).Return(Partition1SpecificSharedConsumerIP, nil)
	mock.On(r.DB("mockdb").Table("machine").Get("1")).Return(M1, nil)
	mock.On(r.DB("mockdb").Table("machine").Get("2")).Return(M2, nil)
	mock.On(r.DB("mockdb").Table("machine").Get("3")).Return(M3, nil)
	mock.On(r.DB("mockdb").Table("machine").Get("4")).Return(M4, nil)
	mock.On(r.DB("mockdb").Table("machine").Get("5")).Return(M5, nil)
	mock.On(r.DB("mockdb").Table("machine").Get("6")).Return(M6, nil)
	mock.On(r.DB("mockdb").Table("machine").Get("7")).Return(M7, nil)
	mock.On(r.DB("mockdb").Table("machine").Get("8")).Return(M8, nil)
	mock.On(r.DB("mockdb").Table("machine").Get("404")).Return(nil, errors.New("Test Error"))
	mock.On(r.DB("mockdb").Table("machine").Get("999")).Return(nil, nil)
	mock.On(r.DB("mockdb").Table("machine").Filter(func(var_1 r.Term) r.Term {
		return var_1.Field("partitionid").Eq(Partition1.ID)
	})).Return(metal.Machines{M1, M2, M3, M4, M5, M7, M8}, nil)
	mock.On(r.DB("mockdb").Table("switch").Get("switch1")).Return(Switch1, nil)
	mock.On(r.DB("mockdb").Table("switch").Get("switch2")).Return(Switch2, nil)
	mock.On(r.DB("mockdb").Table("switch").Get("switch3")).Return(Switch3, nil)
	mock.On(r.DB("mockdb").Table("switch").Get("switch404")).Return(nil, errors.New("Test Error"))
	mock.On(r.DB("mockdb").Table("switch").Get("switch999")).Return(nil, nil)
	mock.On(r.DB("mockdb").Table("switchstatus").Get("switch999")).Return(nil, nil)
	mock.On(r.DB("mockdb").Table("switchstatus").Get(Switch1.ID)).Return(metal.SwitchStatus{
		Base: metal.Base{
			ID: Switch1.ID,
		},
	}, nil)
	mock.On(r.DB("mockdb").Table("wait").Get("3").Changes()).Return([]interface{}{
		map[string]interface{}{"new_val": M3},
	}, nil)
	mock.On(r.DB("mockdb").Table("integerpool").Get(r.MockAnything()).Delete(r.DeleteOpts{ReturnChanges: true})).Return(r.WriteResponse{Changes: []r.ChangeResponse{r.ChangeResponse{OldValue: map[string]interface{}{"id": float64(12345)}}}}, nil)

	// Find
	mock.On(r.DB("mockdb").Table("sizereservation").Filter(r.MockAnything())).Return(metal.SizeReservations{}, nil)

	// Default: Return Empty result
	mock.On(r.DB("mockdb").Table("size").Get(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("partition").Get(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("machine").Get(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("image").Get(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("network").Get(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("ip").Get(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("switch").Get(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("switchstatus").Get(r.MockAnything())).Return(EmptyResult, nil)
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
	mock.On(r.DB("mockdb").Table("switchstatus")).Return(TestSwitchStates, nil)
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
	mock.On(r.DB("mockdb").Table("switchstatus").Get(r.MockAnything()).Replace(r.MockAnything())).Return(EmptyResult, nil)

	// X.insert
	mock.On(r.DB("mockdb").Table("machine").Insert(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("image").Insert(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("network").Insert(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("ip").Insert(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("partition").Insert(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("size").Insert(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("switch").Insert(r.MockAnything())).Return(EmptyResult, nil)
	mock.On(r.DB("mockdb").Table("switchstatus").Insert(r.MockAnything())).Return(EmptyResult, nil)
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
	mock.On(r.DB("mockdb").Table("switchstatus").Insert(r.MockAnything(), r.InsertOpts{
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
