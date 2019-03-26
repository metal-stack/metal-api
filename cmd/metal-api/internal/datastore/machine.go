package datastore

import (
	"crypto/sha256"
	"fmt"
	"strconv"
	"time"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	humanize "github.com/dustin/go-humanize"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

// FindMachine returns the machine with the given ID. If there is no
// such machine a metal.NotFound will be returned.
func (rs *RethinkStore) FindMachine(id string) (*metal.Machine, error) {
	res, err := rs.machineTable().Get(id).Run(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot get machine from database: %v", err)
	}
	defer res.Close()
	if res.IsNil() {
		return nil, metal.NotFound("no machine with %q found", id)
	}

	var d metal.Machine
	err = res.One(&d)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch results: %v", err) // Not Reachable?
	}
	if d.SizeID != "" {
		s, err := rs.FindSize(d.SizeID)
		if err != nil {
			return nil, err
		}
		d.Size = s
	}
	if d.PartitionID != "" {
		st, err := rs.FindPartition(d.PartitionID)
		if err != nil {
			return nil, err
		}
		d.Partition = *st
	}
	if d.Allocation != nil {
		if d.Allocation.ImageID != "" {
			f, err := rs.FindImage(d.Allocation.ImageID)
			if err != nil {
				return nil, fmt.Errorf("illegal imageid-id %q in machine %q", d.Allocation.ImageID, id)
			}
			d.Allocation.Image = f
		}
	}
	return &d, nil
}

// SearchMachine will search machines, optionally by mac. If no mac is
// given all machines will be returned. If no machines are found you
// will get an empty list.
func (rs *RethinkStore) SearchMachine(mac string) ([]metal.Machine, error) {
	q := *rs.machineTable()
	if mac != "" {
		q = q.Filter(func(d r.Term) r.Term {
			return d.Field("macAddresses").Contains(mac)
		})
	}
	res, err := q.Run(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot search machines from database: %v", err)
	}
	defer res.Close()
	data := make([]metal.Machine, 0)
	err = res.All(&data)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch results: %v", err)
	}
	return rs.fillMachineList(data...)
}

// ListMachines returns all machines currently stored in the database.
func (rs *RethinkStore) ListMachines() ([]metal.Machine, error) {
	return rs.SearchMachine("")
}

// CreateMachine creates a new machine in the database as "unallocated new machines".
// If the given machine has an allocation, the function returns an error because
// allocated machines cannot be created. If there is already a machine with the
// given ID in the database it will be replaced the the given machine.
func (rs *RethinkStore) CreateMachine(d *metal.Machine) (*metal.Machine, error) {
	d.Changed = time.Now()
	d.Created = d.Changed

	if d.Allocation != nil {
		return nil, fmt.Errorf("a machine cannot be created when it is allocated: %q: %+v", d.ID, *d.Allocation)
	}
	d.SizeID = d.Size.ID
	d.PartitionID = d.Partition.ID
	res, err := rs.machineTable().Insert(d, r.InsertOpts{
		Conflict: "replace",
	}).RunWrite(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot create machine in database: %v", err)
	}
	if d.ID == "" {
		d.ID = res.GeneratedKeys[0]
	}
	return d, nil
}

// FindIPMI returns the IPMI data for the given machine id.
func (rs *RethinkStore) FindIPMI(id string) (*metal.IPMI, error) {
	res, err := rs.ipmiTable().Get(id).Run(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot query ipmi data: %v", err)
	}
	if res.IsNil() {
		return nil, metal.NotFound("no impi for machine %q found", id)
	}
	var ipmi metal.IPMI
	err = res.One(&ipmi)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch ipmi data: %v", err)
	}
	return &ipmi, nil
}

// UpsertIPMI inserts or updates the IPMI data for a given machine id.
func (rs *RethinkStore) UpsertIPMI(id string, ipmi *metal.IPMI) error {
	ipmi.ID = id
	_, err := rs.ipmiTable().Insert(ipmi, r.InsertOpts{
		Conflict: "replace",
	}).RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot create ipmi in database: %v", err)
	}
	return nil
}

// DeleteMachine removes a machine from the database.
func (rs *RethinkStore) DeleteMachine(id string) (*metal.Machine, error) {
	d, err := rs.FindMachine(id)
	if err != nil {
		return nil, err
	}
	_, err = rs.machineTable().Get(id).Delete().RunWrite(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot delete machine from database: %v", err)
	}
	return d, nil
}

// UpdateMachine replaces a machine in the database if the 'changed' field of
// the old value equals the 'changed' field of the recored in the database.
func (rs *RethinkStore) UpdateMachine(oldD *metal.Machine, newD *metal.Machine) error {
	_, err := rs.machineTable().Get(oldD.ID).Replace(func(row r.Term) r.Term {
		return r.Branch(row.Field("changed").Eq(r.Expr(oldD.Changed)), newD, r.Error("the machine was changed from another, please retry"))
	}).RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot update machine: %v", err)
	}
	return nil
}

func generateVrfID(i string) (uint, error) {
	sha := sha256.Sum256([]byte(i))
	// cut four bytes of hash
	hexTrunc := fmt.Sprintf("%x", sha)[:4]
	hash, err := strconv.ParseUint(hexTrunc, 16, 16)
	if err != nil {
		return 0, err
	}
	return uint(hash), err
}

func (rs *RethinkStore) findVrf(f map[string]interface{}) (*metal.Vrf, error) {
	q := *rs.vrfTable()
	q = q.Filter(f)
	res, err := q.Run(rs.session)
	defer res.Close()
	if res.IsNil() {
		return nil, nil
	}
	var vrf *metal.Vrf
	err = res.One(vrf)
	if err != nil {
		return nil, err
	}
	return vrf, nil
}

func (rs *RethinkStore) reserveNewVrf(tenant string) (*metal.Vrf, error) {
	var hashInput = tenant
	for {
		id, err := generateVrfID(hashInput)
		if err != nil {
			return nil, err
		}
		vrf, err := rs.findVrf(map[string]interface{}{"id": id})
		if err != nil {
			return nil, err
		}
		if vrf != nil {
			hashInput += "salt"
			continue
		}
		vrf = &metal.Vrf{
			ID:     id,
			Tenant: tenant,
		}
		_, err = rs.vrfTable().Insert(vrf).RunWrite(rs.session)
		if err != nil {
			return nil, err
		}
		return vrf, nil
	}
}

// AllocateMachine allocates a machine in the database. It searches the 'waitTable'
// for a machine which matches the criteria for partition and size. If a machine is
// found the system will allocate a CIDR, create an allocation and update the
// machine in the database.
func (rs *RethinkStore) AllocateMachine(
	uuid string,
	name string,
	description string,
	hostname string,
	projectid string,
	part *metal.Partition, size *metal.Size,
	img *metal.Image,
	sshPubKeys []string,
	tags []string,
	userData string,
	tenant string,
	cidrAllocator CidrAllocator,
) (*metal.Machine, error) {
	query := rs.waitTable().Filter(map[string]interface{}{
		"allocation": nil,
		"id":         uuid,
	}).Filter(func(row r.Term) r.Term {
		return row.Field("state").Field("value").Ne(string(metal.AvailableState))
	})
	if uuid == "" {
		query = rs.waitTable().Filter(map[string]interface{}{
			"allocation":  nil,
			"partitionid": part.ID,
			"sizeid":      size.ID,
			"state": map[string]interface{}{
				"value": "",
			},
		})
	}
	available, err := query.Run(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot find free machine: %v", err)
	}
	var res []metal.Machine
	err = available.All(&res)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch results: %v", err)
	}
	if len(res) < 1 {
		return nil, ErrNoMachineAvailable
	}

	old := res[0]
	var vrf *metal.Vrf
	vrf, err = rs.findVrf(map[string]interface{}{"tenant": tenant})
	if err != nil {
		return nil, fmt.Errorf("cannot find vrf for tenant: %v", err)
	}

	if vrf == nil {
		vrf, err = rs.reserveNewVrf(tenant)
		if err != nil {
			return nil, fmt.Errorf("cannot reserve new vrf for tenant: %v", err)
		}
	}

	cidr, err := cidrAllocator.Allocate(res[0].ID, tenant, vrf.ID, projectid, name, description, img.Name)
	if err != nil {
		return nil, fmt.Errorf("cannot allocate at netbox: %v", err)
	}

	rs.fillMachineList(res[0:1]...)
	alloc := &metal.MachineAllocation{
		Created:     time.Now(),
		Name:        name,
		Hostname:    hostname,
		Tenant:      tenant,
		Project:     projectid,
		Description: description,
		Image:       img,
		ImageID:     img.ID,
		SSHPubKeys:  sshPubKeys,
		UserData:    userData,
		Cidr:        cidr,
		Vrf:         vrf.ID,
	}
	res[0].Allocation = alloc
	res[0].Changed = time.Now()

	tagSet := make(map[string]bool)
	tagList := append(res[0].Tags, tags...)
	for _, t := range tagList {
		tagSet[t] = true
	}
	newTags := []string{}
	for k := range tagSet {
		newTags = append(newTags, k)
	}
	res[0].Tags = newTags
	err = rs.UpdateMachine(&old, &res[0])
	if err != nil {
		cidrAllocator.Release(res[0].ID)
		return nil, fmt.Errorf("error when allocating machine %q, %v", res[0].ID, err)
	}
	_, err = rs.waitTable().Get(res[0].ID).Update(res[0]).RunWrite(rs.session)
	if err != nil {
		cidrAllocator.Release(res[0].ID)
		rs.UpdateMachine(&res[0], &old)
		return nil, fmt.Errorf("cannot allocate machine in DB: %v", err)
	}
	return &res[0], nil
}

// FreeMachine removes an allocation from a given machine.
func (rs *RethinkStore) FreeMachine(id string) (*metal.Machine, error) {
	machine, err := rs.FindMachine(id)
	if err != nil {
		return nil, err
	}
	if machine.Allocation == nil {
		return nil, fmt.Errorf("machine is not allocated")
	}
	old := *machine
	machine.Allocation = nil
	err = rs.UpdateMachine(&old, machine)
	if err != nil {
		return nil, fmt.Errorf("cannot clear machine data: %v", err)
	}
	return machine, nil
}

// RegisterMachine creates or updates a machine in the database. It also creates
// an IPMI data record for this machine.
// Maby it would be good to give a machine As Parameter, insted of all Attributes of a machine.
func (rs *RethinkStore) RegisterMachine(
	id string,
	part metal.Partition, rackid string,
	sz metal.Size,
	hardware metal.MachineHardware,
	ipmi metal.IPMI,
	tags []string) (*metal.Machine, error) {

	machine, err := rs.FindMachine(id)
	name := fmt.Sprintf("%d-core/%s", hardware.CPUCores, humanize.Bytes(hardware.Memory))
	descr := fmt.Sprintf("a machine with %d core(s) and %s of RAM", hardware.CPUCores, humanize.Bytes(hardware.Memory))
	if err != nil {
		if metal.IsNotFound(err) {
			machine = &metal.Machine{
				Base: metal.Base{
					ID:          id,
					Name:        name,
					Description: descr,
				},
				Size:        &sz,
				Partition:   part,
				PartitionID: part.ID,
				RackID:      rackid,
				Hardware:    hardware,
				Tags:        tags,
			}
			machine, err = rs.CreateMachine(machine)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else {
		old := *machine
		machine.Hardware = hardware
		machine.Partition = part
		machine.PartitionID = part.ID
		machine.Size = &sz
		machine.RackID = rackid
		machine.Name = name
		machine.Description = descr
		machine.Tags = tags
		err = rs.UpdateMachine(&old, machine)
		if err != nil {
			return nil, err
		}
	}
	err = rs.UpsertIPMI(id, &ipmi)
	if err != nil {
		return nil, err
	}

	return machine, nil
}

// Wait inserts the machine with the given ID in the waittable, so
// this machine is ready for allocation. After this, this function waits
// for an update of this record in the waittable, which is a signal that
// this machine is allocated. This allocation will be signaled via the
// given allocator in a separate goroutine. The allocator is a function
// which will receive a channel and the caller has to select on this
// channel to get a result. Using a channel allows the caller of this
// function to implement timeouts to not wait forever.
// The user of this function will block until this machine is allocated.
func (rs *RethinkStore) Wait(id string, alloc Allocator) error {
	m, err := rs.FindMachine(id)
	if err != nil {
		return err
	}
	a := make(chan MachineAllocation)

	// the machine IS already allocated, so notify this allocation back.
	if m.Allocation != nil {
		go func() {
			a <- MachineAllocation{Machine: m}
		}()
		return alloc(a)
	}

	// does not prohibit concurrent wait calls for the same UUID
	_, err = rs.waitTable().Insert(m, r.InsertOpts{
		Conflict: "replace",
	}).RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot insert machine into wait table: %v", err)
	}
	defer func() {
		rs.waitTable().Get(id).Delete().RunWrite(rs.session)
	}()

	go func() {
		ch, err := rs.waitTable().Get(id).Changes().Run(rs.session)
		if err != nil {
			rs.Error("cannot wait for allocation", "error", err)
			// simply return so this machine will not be allocated
			// the normal timeout-behaviour of the allocator will
			// occur without an allocation
			return
		}
		type responseType struct {
			NewVal metal.Machine `rethinkdb:"new_val"`
			OldVal metal.Machine `rethinkdb:"old_val"`
		}
		var response responseType
		for ch.Next(&response) {
			rs.Infow("machine changed", "response", response)
			if response.NewVal.ID == "" {
				// the entry was deleted, no wait any more
				a <- MachineAllocation{Err: fmt.Errorf("machine %s not available any more", id)}
				break
			}
			res, err := rs.fillMachineList(response.NewVal)
			if err != nil {
				rs.Errorw("machine could not be populated", "error", err, "id", response.NewVal.ID)
				break
			}
			a <- MachineAllocation{Machine: &res[0]}
			break
		}
		rs.Infow("stop waiting for changes", "id", id)
		close(a)
	}()
	return alloc(a)
}

// fillMachineList fills the output fields of a machine which are not directly
// stored in the database.
func (rs *RethinkStore) fillMachineList(data ...metal.Machine) ([]metal.Machine, error) {
	allsz, err := rs.ListSizes()
	if err != nil {
		return nil, err
	}
	szmap := metal.Sizes(allsz).ByID()

	allimg, err := rs.ListImages()
	if err != nil {
		return nil, err
	}
	imgmap := metal.Images(allimg).ByID()

	all, err := rs.ListPartitions()
	if err != nil {
		return nil, err
	}
	partmap := metal.Partitions(all).ByID()

	res := make([]metal.Machine, len(data), len(data))
	for i, d := range data {
		res[i] = d
		size := szmap[d.SizeID]
		res[i].Size = &size
		if d.Allocation != nil {
			if d.Allocation.ImageID != "" {
				img := imgmap[d.Allocation.ImageID]
				res[i].Allocation.Image = &img
			}
		}
		res[i].Partition = partmap[d.PartitionID]
	}
	return res, nil
}
