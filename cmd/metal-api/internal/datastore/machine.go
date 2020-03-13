package datastore

import (
	"context"
	"fmt"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

// MachineSearchQuery can be used to search machines.
type MachineSearchQuery struct {
	ID          *string  `json:"id"`
	Name        *string  `json:"name"`
	PartitionID *string  `json:"partition_id"`
	SizeID      *string  `json:"sizeid"`
	RackID      *string  `json:"rackid"`
	Liveliness  *string  `json:"liveliness"`
	Tags        []string `json:"tags"`

	// allocation
	AllocationName      *string `json:"allocation_name"`
	AllocationProject   *string `json:"allocation_project"`
	AllocationImageID   *string `json:"allocation_image_id"`
	AllocationHostname  *string `json:"allocation_hostname"`
	AllocationSucceeded *bool   `json:"allocation_succeeded"`

	// network
	NetworkIDs                 []string `json:"network_ids"`
	NetworkPrefixes            []string `json:"network_prefixes"`
	NetworkIPs                 []string `json:"network_ips"`
	NetworkDestinationPrefixes []string `json:"network_destination_prefixes"`
	NetworkVrfs                []int64  `json:"network_vrfs"`
	NetworkPrivate             *bool    `json:"network_private"`
	NetworkASNs                []int64  `json:"network_asns"`
	NetworkNat                 *bool    `json:"network_nat"`
	NetworkUnderlay            *bool    `json:"network_underlay"`

	// hardware
	HardwareMemory   *int64 `json:"hardware_memory"`
	HardwareCPUCores *int64 `json:"hardware_cpu_cores"`

	// nics
	NicsMacAddresses         []string `json:"nics_mac_addresses"`
	NicsNames                []string `json:"nics_names"`
	NicsVrfs                 []string `json:"nics_vrfs"`
	NicsNeighborMacAddresses []string `json:"nics_neighbor_mac_addresses"`
	NicsNeighborNames        []string `json:"nics_neighbor_names"`
	NicsNeighborVrfs         []string `json:"nics_neighbor_vrfs"`

	// disks
	DiskNames []string `json:"disk_names"`
	DiskSizes []int64  `json:"disk_sizes"`

	// state
	StateValue *string `json:"state_value"`

	// ipmi
	IpmiAddress    *string `json:"ipmi_address"`
	IpmiMacAddress *string `json:"ipmi_mac_address"`
	IpmiUser       *string `json:"ipmi_user"`
	IpmiInterface  *string `json:"ipmi_interface"`

	// fru
	FruChassisPartNumber   *string `json:"fru_chassis_part_number"`
	FruChassisPartSerial   *string `json:"fru_chassis_part_serial"`
	FruBoardMfg            *string `json:"fru_board_mfg"`
	FruBoardMfgSerial      *string `json:"fru_board_mfg_serial"`
	FruBoardPartNumber     *string `json:"fru_board_part_number"`
	FruProductManufacturer *string `json:"fru_product_manufacturer"`
	FruProductPartNumber   *string `json:"fru_product_part_number"`
	FruProductSerial       *string `json:"fru_product_serial"`
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

	if p.Liveliness != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("liveliness").Eq(*p.Liveliness)
		})
	}

	for _, tag := range p.Tags {
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
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("allocation").Field("networks").Map(func(nw r.Term) r.Term {
				return nw.Field("networkid")
			}).Contains(r.Expr(id))
		})
	}

	for _, prefix := range p.NetworkPrefixes {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("allocation").Field("networks").Map(func(nw r.Term) r.Term {
				return nw.Field("prefixes")
			}).Contains(r.Expr(prefix))
		})
	}

	for _, ip := range p.NetworkIPs {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("allocation").Field("networks").Map(func(nw r.Term) r.Term {
				return nw.Field("ips")
			}).Contains(r.Expr(ip))
		})
	}

	for _, destPrefix := range p.NetworkDestinationPrefixes {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("allocation").Field("networks").Map(func(nw r.Term) r.Term {
				return nw.Field("destinationprefixes")
			}).Contains(r.Expr(destPrefix))
		})
	}

	for _, vrf := range p.NetworkVrfs {
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
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("hardware").Field("network_interfaces").Map(func(nic r.Term) r.Term {
				return nic.Field("macAddress")
			}).Contains(r.Expr(mac))
		})
	}

	for _, name := range p.NicsNames {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("hardware").Field("network_interfaces").Map(func(nic r.Term) r.Term {
				return nic.Field("name")
			}).Contains(r.Expr(name))
		})
	}

	for _, vrf := range p.NicsVrfs {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("hardware").Field("network_interfaces").Map(func(nic r.Term) r.Term {
				return nic.Field("vrf")
			}).Contains(r.Expr(vrf))
		})
	}

	for _, mac := range p.NicsNeighborMacAddresses {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("hardware").Field("network_interfaces").Map(func(nic r.Term) r.Term {
				return nic.Field("neighbors").Map(func(neigh r.Term) r.Term {
					return neigh.Field("macAddress")
				})
			}).Contains(r.Expr(mac))
		})
	}

	for _, name := range p.NicsNames {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("hardware").Field("network_interfaces").Map(func(nic r.Term) r.Term {
				return nic.Field("neighbors").Map(func(neigh r.Term) r.Term {
					return neigh.Field("name")
				})
			}).Contains(r.Expr(name))
		})
	}

	for _, vrf := range p.NicsVrfs {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("hardware").Field("network_interfaces").Map(func(nic r.Term) r.Term {
				return nic.Field("neighbors").Map(func(neigh r.Term) r.Term {
					return neigh.Field("vrf")
				})
			}).Contains(r.Expr(vrf))
		})
	}

	for _, name := range p.DiskNames {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("block_devices").Map(func(bd r.Term) r.Term {
				return bd.Field("name")
			}).Contains(r.Expr(name))
		})
	}

	for _, size := range p.DiskSizes {
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

// InsertWaitingMachine adds a machine to the wait table.
func (rs *RethinkStore) InsertWaitingMachine(m *metal.Machine) error {
	// does not prohibit concurrent wait calls for the same UUID
	return rs.upsertEntity(rs.waitTable(), m)
}

// RemoveWaitingMachine removes a machine from the wait table.
func (rs *RethinkStore) RemoveWaitingMachine(m *metal.Machine) error {
	return rs.deleteEntity(rs.waitTable(), m)
}

// UpdateWaitingMachine updates a machine in the wait table with the given machine
func (rs *RethinkStore) UpdateWaitingMachine(m *metal.Machine) error {
	_, err := rs.waitTable().Get(m.ID).Update(m).RunWrite(rs.session)
	return err
}

// WaitForMachineAllocation listens on changes on the wait table for a given machine and returns the changed machine.
func (rs *RethinkStore) WaitForMachineAllocation(ctx context.Context, m *metal.Machine) (*metal.Machine, error) {
	type responseType struct {
		NewVal metal.Machine `rethinkdb:"new_val" json:"new_val"`
		OldVal metal.Machine `rethinkdb:"old_val" json:"old_val"`
	}
	var response responseType
	err := rs.listenForEntityChange(ctx, rs.waitTable(), m, response)
	if err != nil {
		return nil, err
	}

	if response.NewVal.ID == "" {
		// the machine was taken out of the wait table and not allocated
		return nil, fmt.Errorf("machine %q was taken out of the wait table", m.ID)
	}

	// the machine was really allocated!
	return &response.NewVal, nil
}

// FindAvailableMachine returns an available machine that momentarily also sits in the wait table.
func (rs *RethinkStore) FindAvailableMachine(partitionid, sizeid string) (*metal.Machine, error) {
	q := *rs.waitTable()
	q = q.Filter(map[string]interface{}{
		"allocation":  nil,
		"partitionid": partitionid,
		"sizeid":      sizeid,
		"state": map[string]string{
			"value": string(metal.AvailableState),
		},
	}).Sample(1)

	var available metal.Machines
	err := rs.searchEntities(&q, &available)
	if err != nil {
		return nil, err
	}

	if len(available) < 1 {
		return nil, fmt.Errorf("no machine available")
	}

	// we actually return the machine from the machine table, not from the wait table
	// otherwise we will get in trouble with update operations on the machine table because
	// we have mixed timestamps with the entity from the wait table...
	m, err := rs.FindMachineByID(available[0].ID)
	if err != nil {
		return nil, err
	}

	return m, nil
}
