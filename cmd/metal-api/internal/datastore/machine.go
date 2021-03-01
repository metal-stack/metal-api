package datastore

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	mathrand "math/rand"
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

// MachineSearchQuery can be used to search machines.
type MachineSearchQuery struct {
	ID          *string  `json:"id" optional:"true"`
	Name        *string  `json:"name" optional:"true"`
	PartitionID *string  `json:"partition_id" optional:"true"`
	SizeID      *string  `json:"sizeid" optional:"true"`
	RackID      *string  `json:"rackid" optional:"true"`
	Tags        []string `json:"tags" optional:"true"`

	// allocation
	AllocationName      *string `json:"allocation_name" optional:"true"`
	AllocationProject   *string `json:"allocation_project" optional:"true"`
	AllocationImageID   *string `json:"allocation_image_id" optional:"true"`
	AllocationHostname  *string `json:"allocation_hostname" optional:"true"`
	AllocationSucceeded *bool   `json:"allocation_succeeded" optional:"true"`

	// network
	NetworkIDs                 []string `json:"network_ids" optional:"true"`
	NetworkPrefixes            []string `json:"network_prefixes" optional:"true"`
	NetworkIPs                 []string `json:"network_ips" optional:"true"`
	NetworkDestinationPrefixes []string `json:"network_destination_prefixes" optional:"true"`
	NetworkVrfs                []int64  `json:"network_vrfs" optional:"true"`
	NetworkPrivate             *bool    `json:"network_private" optional:"true"`
	NetworkASNs                []int64  `json:"network_asns" optional:"true"`
	NetworkNat                 *bool    `json:"network_nat" optional:"true"`
	NetworkUnderlay            *bool    `json:"network_underlay" optional:"true"`

	// hardware
	HardwareMemory   *int64 `json:"hardware_memory" optional:"true"`
	HardwareCPUCores *int64 `json:"hardware_cpu_cores" optional:"true"`

	// nics
	NicsMacAddresses         []string `json:"nics_mac_addresses" optional:"true"`
	NicsNames                []string `json:"nics_names" optional:"true"`
	NicsVrfs                 []string `json:"nics_vrfs" optional:"true"`
	NicsNeighborMacAddresses []string `json:"nics_neighbor_mac_addresses" optional:"true"`
	NicsNeighborNames        []string `json:"nics_neighbor_names" optional:"true"`
	NicsNeighborVrfs         []string `json:"nics_neighbor_vrfs" optional:"true"`

	// disks
	DiskNames []string `json:"disk_names" optional:"true"`
	DiskSizes []int64  `json:"disk_sizes" optional:"true"`

	// state
	StateValue *string `json:"state_value" optional:"true"`

	// ipmi
	IpmiAddress    *string `json:"ipmi_address" optional:"true"`
	IpmiMacAddress *string `json:"ipmi_mac_address" optional:"true"`
	IpmiUser       *string `json:"ipmi_user" optional:"true"`
	IpmiInterface  *string `json:"ipmi_interface" optional:"true"`

	// fru
	FruChassisPartNumber   *string `json:"fru_chassis_part_number" optional:"true"`
	FruChassisPartSerial   *string `json:"fru_chassis_part_serial" optional:"true"`
	FruBoardMfg            *string `json:"fru_board_mfg" optional:"true"`
	FruBoardMfgSerial      *string `json:"fru_board_mfg_serial" optional:"true"`
	FruBoardPartNumber     *string `json:"fru_board_part_number" optional:"true"`
	FruProductManufacturer *string `json:"fru_product_manufacturer" optional:"true"`
	FruProductPartNumber   *string `json:"fru_product_part_number" optional:"true"`
	FruProductSerial       *string `json:"fru_product_serial" optional:"true"`
}

// GenerateTerm generates the project search query term.
func (p *MachineSearchQuery) generateTerm(rs *RethinkStore) *r.Term {
	q := *rs.machineTable()

	if p.ID != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("id").Eq(*p.ID)
		})
	}

	if p.Name != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("name").Eq(*p.Name)
		})
	}

	if p.PartitionID != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("partitionid").Eq(*p.PartitionID)
		})
	}

	if p.SizeID != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("sizeid").Eq(*p.SizeID)
		})
	}

	if p.RackID != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("rackid").Eq(*p.RackID)
		})
	}

	for _, tag := range p.Tags {
		tag := tag
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("tags").Contains(r.Expr(tag))
		})
	}

	if p.AllocationName != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("allocation").Field("name").Eq(*p.AllocationName)
		})
	}

	if p.AllocationProject != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("allocation").Field("project").Eq(*p.AllocationProject)
		})
	}

	if p.AllocationImageID != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("allocation").Field("imageid").Eq(*p.AllocationImageID)
		})
	}

	if p.AllocationHostname != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("allocation").Field("hostname").Eq(*p.AllocationHostname)
		})
	}

	if p.AllocationSucceeded != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("allocation").Field("succeeded").Eq(*p.AllocationSucceeded)
		})
	}

	for _, id := range p.NetworkIDs {
		id := id
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("allocation").Field("networks").Map(func(nw r.Term) r.Term {
				return nw.Field("networkid")
			}).Contains(r.Expr(id))
		})
	}

	for _, prefix := range p.NetworkPrefixes {
		prefix := prefix
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("allocation").Field("networks").Map(func(nw r.Term) r.Term {
				return nw.Field("prefixes")
			}).Contains(r.Expr(prefix))
		})
	}

	for _, ip := range p.NetworkIPs {
		ip := ip
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("allocation").Field("networks").Map(func(nw r.Term) r.Term {
				return nw.Field("ips")
			}).Contains(r.Expr(ip))
		})
	}

	for _, destPrefix := range p.NetworkDestinationPrefixes {
		destPrefix := destPrefix
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("allocation").Field("networks").Map(func(nw r.Term) r.Term {
				return nw.Field("destinationprefixes")
			}).Contains(r.Expr(destPrefix))
		})
	}

	for _, vrf := range p.NetworkVrfs {
		vrf := vrf
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("allocation").Field("networks").Map(func(nw r.Term) r.Term {
				return nw.Field("vrf")
			}).Contains(r.Expr(vrf))
		})
	}

	if p.NetworkPrivate != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("allocation").Field("networks").Map(func(nw r.Term) r.Term {
				return nw.Field("private")
			}).Contains(*p.NetworkPrivate)
		})
	}

	for _, asn := range p.NetworkASNs {
		asn := asn
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("allocation").Field("networks").Map(func(nw r.Term) r.Term {
				return nw.Field("asn")
			}).Contains(r.Expr(asn))
		})
	}

	if p.NetworkNat != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("allocation").Field("networks").Map(func(nw r.Term) r.Term {
				return nw.Field("nat")
			}).Contains(*p.NetworkNat)
		})
	}

	if p.NetworkUnderlay != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("allocation").Field("networks").Map(func(nw r.Term) r.Term {
				return nw.Field("underlay")
			}).Contains(*p.NetworkUnderlay)
		})
	}

	if p.HardwareMemory != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("hardware").Field("memory").Eq(*p.HardwareMemory)
		})
	}

	if p.HardwareCPUCores != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("hardware").Field("cpu_cores").Eq(*p.HardwareCPUCores)
		})
	}

	for _, mac := range p.NicsMacAddresses {
		mac := mac
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("hardware").Field("network_interfaces").Map(func(nic r.Term) r.Term {
				return nic.Field("macAddress")
			}).Contains(r.Expr(mac))
		})
	}

	for _, name := range p.NicsNames {
		name := name
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("hardware").Field("network_interfaces").Map(func(nic r.Term) r.Term {
				return nic.Field("name")
			}).Contains(r.Expr(name))
		})
	}

	for _, vrf := range p.NicsVrfs {
		vrf := vrf
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("hardware").Field("network_interfaces").Map(func(nic r.Term) r.Term {
				return nic.Field("vrf")
			}).Contains(r.Expr(vrf))
		})
	}

	for _, mac := range p.NicsNeighborMacAddresses {
		mac := mac
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("hardware").Field("network_interfaces").Map(func(nic r.Term) r.Term {
				return nic.Field("neighbors").Map(func(neigh r.Term) r.Term {
					return neigh.Field("macAddress")
				})
			}).Contains(r.Expr(mac))
		})
	}

	for _, name := range p.NicsNames {
		name := name
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("hardware").Field("network_interfaces").Map(func(nic r.Term) r.Term {
				return nic.Field("neighbors").Map(func(neigh r.Term) r.Term {
					return neigh.Field("name")
				})
			}).Contains(r.Expr(name))
		})
	}

	for _, vrf := range p.NicsVrfs {
		vrf := vrf
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("hardware").Field("network_interfaces").Map(func(nic r.Term) r.Term {
				return nic.Field("neighbors").Map(func(neigh r.Term) r.Term {
					return neigh.Field("vrf")
				})
			}).Contains(r.Expr(vrf))
		})
	}

	for _, name := range p.DiskNames {
		name := name
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("block_devices").Map(func(bd r.Term) r.Term {
				return bd.Field("name")
			}).Contains(r.Expr(name))
		})
	}

	for _, size := range p.DiskSizes {
		size := size
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("block_devices").Map(func(bd r.Term) r.Term {
				return bd.Field("size")
			}).Contains(r.Expr(size))
		})
	}

	if p.StateValue != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("state_value").Eq(*p.StateValue)
		})
	}

	if p.IpmiAddress != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("ipmi").Field("address").Eq(*p.IpmiAddress)
		})
	}

	if p.IpmiMacAddress != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("ipmi").Field("mac").Eq(*p.IpmiMacAddress)
		})
	}

	if p.IpmiUser != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("ipmi").Field("user").Eq(*p.IpmiUser)
		})
	}

	if p.IpmiInterface != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("ipmi").Field("interface").Eq(*p.IpmiInterface)
		})
	}

	if p.FruChassisPartNumber != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("ipmi").Field("fru").Field("chassis_part_number").Eq(*p.FruChassisPartNumber)
		})
	}

	if p.FruChassisPartSerial != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("ipmi").Field("fru").Field("chassis_part_serial").Eq(*p.FruChassisPartSerial)
		})
	}

	if p.FruBoardMfg != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("ipmi").Field("fru").Field("board_mfg").Eq(*p.FruBoardMfg)
		})
	}

	if p.FruBoardMfgSerial != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("ipmi").Field("fru").Field("board_mfg_serial").Eq(*p.FruBoardMfgSerial)
		})
	}

	if p.FruBoardPartNumber != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("ipmi").Field("fru").Field("board_part_number").Eq(*p.FruBoardPartNumber)
		})
	}

	if p.FruProductManufacturer != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("ipmi").Field("fru").Field("product_manufacturer").Eq(*p.FruProductManufacturer)
		})
	}

	if p.FruProductPartNumber != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("ipmi").Field("fru").Field("product_part_number").Eq(*p.FruProductPartNumber)
		})
	}

	if p.FruProductSerial != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("ipmi").Field("fru").Field("product_serial").Eq(*p.FruProductSerial)
		})
	}

	return &q
}

// FindMachineByID returns a machine for a given id.
func (rs *RethinkStore) FindMachineByID(id string) (*metal.Machine, error) {
	var m metal.Machine
	err := rs.findEntityByID(rs.machineTable(), &m, id)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// FindMachine returns a machine by the given query, fails if there is no record or multiple records found.
func (rs *RethinkStore) FindMachine(q *MachineSearchQuery, ms *metal.Machine) error {
	return rs.findEntity(q.generateTerm(rs), &ms)
}

// SearchMachines returns the result of the machines search request query.
func (rs *RethinkStore) SearchMachines(q *MachineSearchQuery, ms *metal.Machines) error {
	return rs.searchEntities(q.generateTerm(rs), ms)
}

// ListMachines returns all machines.
func (rs *RethinkStore) ListMachines() (metal.Machines, error) {
	ms := make(metal.Machines, 0)
	err := rs.listEntities(rs.machineTable(), &ms)
	return ms, err
}

// CreateMachine creates a new machine in the database as "unallocated new machines".
// If the given machine has an allocation, the function returns an error because
// allocated machines cannot be created. If there is already a machine with the
// given ID in the database it will be replaced the the given machine.
// CreateNetwork creates a new network.
func (rs *RethinkStore) CreateMachine(m *metal.Machine) error {
	if m.Allocation != nil {
		return fmt.Errorf("a machine cannot be created when it is allocated: %q: %+v", m.ID, *m.Allocation)
	}
	return rs.createEntity(rs.machineTable(), m)
}

// DeleteMachine removes a machine from the database.
func (rs *RethinkStore) DeleteMachine(m *metal.Machine) error {
	return rs.deleteEntity(rs.machineTable(), m)
}

// UpdateMachine replaces a machine in the database if the 'changed' field of
// the old value equals the 'changed' field of the recored in the database.
func (rs *RethinkStore) UpdateMachine(oldMachine *metal.Machine, newMachine *metal.Machine) error {
	return rs.updateEntity(rs.machineTable(), newMachine, oldMachine)
}

// FindWaitingMachine returns an available, not allocated, waiting and alive machine of given size within the given partition.
// TODO: the algorithm can be optimized / shortened by using a rethinkdb join command and then using .Sample(1)
// but current implementation should have a slightly better readability.
func (rs *RethinkStore) FindWaitingMachine(partitionid, sizeid string) (*metal.Machine, error) {
	q := *rs.machineTable()
	q = q.Filter(map[string]interface{}{
		"allocation":  nil,
		"partitionid": partitionid,
		"sizeid":      sizeid,
		"state": map[string]string{
			"value": string(metal.AvailableState),
		},
		"waiting": true,
	})

	var candidates metal.Machines
	err := rs.searchEntities(&q, &candidates)
	if err != nil {
		return nil, err
	}

	ecs, err := rs.ListProvisioningEventContainers()
	if err != nil {
		return nil, err
	}
	ecMap := ecs.ByID()

	var available metal.Machines
	for _, m := range candidates {
		ec, ok := ecMap[m.ID]
		if !ok {
			rs.SugaredLogger.Errorw("cannot find machine provisioning event container", "machine", m, "error", err)
			// fall through, so the rest of the machines is getting evaluated
			continue
		}
		if ec.Liveliness != metal.MachineLivelinessAlive {
			continue
		}
		available = append(available, m)
	}

	if available == nil || len(available) < 1 {
		return nil, errors.New("no machine available")
	}

	// pick a random machine from all available ones
	var idx int
	b, err := rand.Int(rand.Reader, big.NewInt(int64(len(available))))
	if err != nil {
		idx = int(b.Uint64())
	} else {
		mathrand.Seed(time.Now().UnixNano())
		idx = mathrand.Intn(len(available))
	}
	return &available[idx], nil
}
