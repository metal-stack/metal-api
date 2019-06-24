package datastore

import (
	"fmt"
	v1 "git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/service/v1"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

// FindMachine returns the machine with the given ID. If there is no
// such machine a metal.NotFound will be returned.
func (rs *RethinkStore) FindMachine(id string) (*metal.Machine, error) {
	var m metal.Machine
	err := rs.findEntityByID(rs.machineTable(), &m, id)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// ListMachines returns all machines.
func (rs *RethinkStore) ListMachines() ([]metal.Machine, error) {
	ms := make([]metal.Machine, 0)
	err := rs.listEntities(rs.machineTable(), &ms)
	return ms, err
}

// FindMachines returns the machines filtered by the given search filter.
func (rs *RethinkStore) FindMachines(props *v1.FindMachinesRequest) ([]metal.Machine, error) {
	q := *rs.machineTable()

	if props.ID != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("id").Eq(*props.ID)
		})
	}

	if props.Name != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("name").Eq(*props.Name)
		})
	}

	if props.PartitionID != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("partitionid").Eq(*props.PartitionID)
		})
	}

	if props.SizeID != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("sizeid").Eq(*props.SizeID)
		})
	}

	if props.RackID != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("rackid").Eq(*props.RackID)
		})
	}

	if props.Liveliness != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("liveliness").Eq(*props.Liveliness)
		})
	}

	for _, tag := range props.Tags {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("tags").Contains(r.Expr(tag))
		})
	}

	if props.AllocationName != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("allocation").Field("name").Eq(*props.AllocationName)
		})
	}

	if props.AllocationTenant != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("allocation").Field("tenant").Eq(*props.AllocationTenant)
		})
	}

	if props.AllocationProject != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("allocation").Field("project").Eq(*props.AllocationProject)
		})
	}

	if props.AllocationImageID != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("allocation").Field("imageid").Eq(*props.AllocationImageID)
		})
	}

	if props.AllocationHostname != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("allocation").Field("hostname").Eq(*props.AllocationHostname)
		})
	}

	if props.AllocationSucceeded != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("allocation").Field("succeeded").Eq(*props.AllocationSucceeded)
		})
	}

	for _, id := range props.NetworkIDs {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("networks").Field("networkid").Contains(r.Expr(id))
		})
	}

	for _, prefix := range props.NetworkPrefixes {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("networks").Field("prefixes").Contains(r.Expr(prefix))
		})
	}

	for _, ip := range props.NetworkIPs {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("networks").Field("ips").Contains(r.Expr(ip))
		})
	}

	for _, destPrefix := range props.NetworkDestinationPrefixes {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("networks").Field("destinationprefixes").Contains(r.Expr(destPrefix))
		})
	}

	for _, vrf := range props.NetworkVrfs {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("networks").Field("vrf").Contains(r.Expr(vrf))
		})
	}

	if props.NetworkPrimary != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("networks").Field("primary").Eq(*props.NetworkPrimary)
		})
	}

	for _, asn := range props.NetworkASNs {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("networks").Field("asn").Contains(r.Expr(asn))
		})
	}

	if props.NetworkNat != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("networks").Field("nat").Eq(*props.NetworkNat)
		})
	}

	if props.NetworkUnderlay != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("networks").Field("underlay").Eq(*props.NetworkUnderlay)
		})
	}

	if props.HardwareMemory != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("hardware").Field("memory").Eq(*props.HardwareMemory)
		})
	}

	if props.HardwareCPUCores != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("hardware").Field("cpu_cores").Eq(*props.HardwareCPUCores)
		})
	}

	for _, mac := range props.NicsMacAddresses {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("hardware").Field("network_interfaces").Map(func(nic r.Term) r.Term {
				return nic.Field("macAddress")
			}).Contains(r.Expr(mac))
		})
	}

	for _, name := range props.NicsNames {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("hardware").Field("network_interfaces").Map(func(nic r.Term) r.Term {
				return nic.Field("name")
			}).Contains(r.Expr(name))
		})
	}

	for _, vrf := range props.NicsVrfs {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("hardware").Field("network_interfaces").Map(func(nic r.Term) r.Term {
				return nic.Field("vrf")
			}).Contains(r.Expr(vrf))
		})
	}

	for _, mac := range props.NicsNeighborMacAddresses {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("hardware").Field("network_interfaces").Field("neighbors").Map(func(nic r.Term) r.Term {
				return nic.Field("macAddress")
			}).Contains(r.Expr(mac))
		})
	}

	for _, name := range props.NicsNames {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("hardware").Field("network_interfaces").Field("neighbors").Map(func(nic r.Term) r.Term {
				return nic.Field("name")
			}).Contains(r.Expr(name))
		})
	}

	for _, vrf := range props.NicsVrfs {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("hardware").Field("network_interfaces").Field("neighbors").Map(func(nic r.Term) r.Term {
				return nic.Field("vrf")
			}).Contains(r.Expr(vrf))
		})
	}

	for _, name := range props.DiskNames {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("block_devices").Field("name").Contains(r.Expr(name))
		})
	}

	for _, size := range props.DiskSizes {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("block_devices").Field("vrf").Contains(r.Expr(size))
		})
	}

	if props.StateValue != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("state_value").Eq(*props.StateValue)
		})
	}

	if props.IpmiAddress != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("ipmi").Field("address").Eq(*props.IpmiAddress)
		})
	}

	if props.IpmiMacAddress != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("ipmi").Field("mac").Eq(*props.IpmiMacAddress)
		})
	}

	if props.IpmiUser != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("ipmi").Field("user").Eq(*props.IpmiUser)
		})
	}

	if props.IpmiInterface != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("ipmi").Field("interface").Eq(*props.IpmiInterface)
		})
	}

	if props.FruChassisPartNumber != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("ipmi").Field("fru").Field("chassis_part_number").Eq(*props.FruChassisPartNumber)
		})
	}

	if props.FruChassisPartSerial != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("ipmi").Field("fru").Field("chassis_part_serial").Eq(*props.FruChassisPartSerial)
		})
	}

	if props.FruBoardMfg != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("ipmi").Field("fru").Field("board_mfg").Eq(*props.FruBoardMfg)
		})
	}

	if props.FruBoardMfgSerial != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("ipmi").Field("fru").Field("board_mfg_serial").Eq(*props.FruBoardMfgSerial)
		})
	}

	if props.FruBoardPartNumber != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("ipmi").Field("fru").Field("board_part_number").Eq(*props.FruBoardPartNumber)
		})
	}

	if props.FruProductManufacturer != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("ipmi").Field("fru").Field("product_manufacturer").Eq(*props.FruProductManufacturer)
		})
	}

	if props.FruProductPartNumber != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("ipmi").Field("fru").Field("product_part_number").Eq(*props.FruProductPartNumber)
		})
	}

	if props.FruProductSerial != nil {
		q = q.Filter(func(row r.Term) r.Term {
			return row.Field("ipmi").Field("fru").Field("product_serial").Eq(*props.FruProductSerial)
		})
	}

	var ms []metal.Machine
	err := rs.searchEntities(&q, &ms)
	if err != nil {
		return nil, err
	}

	return ms, nil
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
func (rs *RethinkStore) WaitForMachineAllocation(m *metal.Machine) (*metal.Machine, error) {
	type responseType struct {
		NewVal metal.Machine `rethinkdb:"new_val"`
		OldVal metal.Machine `rethinkdb:"old_val"`
	}
	var response responseType
	err := rs.listenForEntityChange(rs.waitTable(), m, response)
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
	})

	var available []metal.Machine
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
	m, err := rs.FindMachine(available[0].ID)
	if err != nil {
		return nil, err
	}

	return m, nil
}
