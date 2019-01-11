package datastore

import (
	"fmt"
	"time"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	humanize "github.com/dustin/go-humanize"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

// FindDevice returns the device with the given ID. If there is no
// such device a metal.NotFound will be returned.
func (rs *RethinkStore) FindDevice(id string) (*metal.Device, error) {
	res, err := rs.deviceTable().Get(id).Run(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot get device from database: %v", err)
	}
	defer res.Close()
	if res.IsNil() {
		return nil, metal.NotFound("no device with %q found", id)
	}

	var d metal.Device
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
	if d.SiteID != "" {
		st, err := rs.FindSite(d.SiteID)
		if err != nil {
			return nil, err
		}
		d.Site = *st
	}
	if d.Allocation != nil {
		if d.Allocation.ImageID != "" {
			f, err := rs.FindImage(d.Allocation.ImageID)
			if err != nil {
				return nil, fmt.Errorf("illegal imageid-id %q in device %q", d.Allocation.ImageID, id)
			}
			d.Allocation.Image = f
		}
	}
	return &d, nil
}

// SearchDevice will search devices, optionally by mac. If no mac is
// given all devices will be returned. If no devices are found you
// will get an empty list.
func (rs *RethinkStore) SearchDevice(mac string) ([]metal.Device, error) {
	q := *rs.deviceTable()
	if mac != "" {
		q = q.Filter(func(d r.Term) r.Term {
			return d.Field("macAddresses").Contains(mac)
		})
	}
	res, err := q.Run(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot search devices from database: %v", err)
	}
	defer res.Close()
	data := make([]metal.Device, 0)
	err = res.All(&data)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch results: %v", err)
	}
	return rs.fillDeviceList(data...)
}

// ListDevices returns all devices currently stored in the database.
func (rs *RethinkStore) ListDevices() ([]metal.Device, error) {
	return rs.SearchDevice("")
}

// CreateDevice creates a new device in the database as "unallocated new devices".
// If the given device has an allocation, the function returns an error because
// allocated devices cannot be created. If there is already a device with the
// given ID in the database it will be replaced the the given device.
func (rs *RethinkStore) CreateDevice(d *metal.Device) (*metal.Device, error) {
	d.Changed = time.Now()
	d.Created = d.Changed

	if d.Allocation != nil {
		return nil, fmt.Errorf("a device cannot be created when it is allocated: %q: %+v", d.ID, *d.Allocation)
	}
	d.SizeID = d.Size.ID
	d.SiteID = d.Site.ID
	res, err := rs.deviceTable().Insert(d, r.InsertOpts{
		Conflict: "replace",
	}).RunWrite(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot create device in database: %v", err)
	}
	if d.ID == "" {
		d.ID = res.GeneratedKeys[0]
	}
	return d, nil
}

// FindIPMI returns the IPMI data for the given device id.
func (rs *RethinkStore) FindIPMI(id string) (*metal.IPMI, error) {
	res, err := rs.ipmiTable().Get(id).Run(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot query ipmi data: %v", err)
	}
	if res.IsNil() {
		return nil, metal.NotFound("no impi for device %q found", id)
	}
	var ipmi metal.IPMI
	err = res.One(&ipmi)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch ipmi data: %v", err)
	}
	return &ipmi, nil
}

// UpsertIPMI inserts or updates the IPMI data for a given device id.
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

// DeleteDevice removes a device from the database.
func (rs *RethinkStore) DeleteDevice(id string) (*metal.Device, error) {
	d, err := rs.FindDevice(id)
	if err != nil {
		return nil, err
	}
	_, err = rs.deviceTable().Get(id).Delete().RunWrite(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot delete device from database: %v", err)
	}
	return d, nil
}

// UpdateDevice replaces a device in the database if the 'changed' field of
// the old value equals the 'changed' field of the recored in the database.
func (rs *RethinkStore) UpdateDevice(oldD *metal.Device, newD *metal.Device) error {
	_, err := rs.deviceTable().Get(oldD.ID).Replace(func(row r.Term) r.Term {
		return r.Branch(row.Field("changed").Eq(r.Expr(oldD.Changed)), newD, r.Error("the device was changed from another, please retry"))
	}).RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot update device: %v", err)
	}
	return nil
}

// AllocateDevice allocates a device in the database. It searches the 'waitTable'
// for a device which matches the criteria for site and size. If a device is
// found the system will allocate a CIDR, create an allocation and update the
// device in the database.
func (rs *RethinkStore) AllocateDevice(
	name string,
	description string,
	hostname string,
	projectid string,
	site *metal.Site,
	size *metal.Size,
	img *metal.Image,
	sshPubKeys []string,
	tenant string,
	cidrAllocator CidrAllocator,
) (*metal.Device, error) {
	available, err := rs.waitTable().Filter(map[string]interface{}{
		"allocation": nil,
		"siteid":     site.ID,
		"sizeid":     size.ID,
	}).Run(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot find free device: %v", err)
	}
	var res []metal.Device
	err = available.All(&res)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch results: %v", err)
	}
	if len(res) < 1 {
		return nil, ErrNoDeviceAvailable
	}

	old := res[0]
	//uuid, tenant, project, name, description, os
	cidr, err := cidrAllocator.Allocate(res[0].ID, tenant, projectid, name, description, img.Name)
	if err != nil {
		return nil, fmt.Errorf("cannot allocate at netbox: %v", err)
	}

	rs.fillDeviceList(res[0:1]...)
	alloc := &metal.DeviceAllocation{
		Created:     time.Now(),
		Name:        name,
		Hostname:    hostname,
		Tenant:      tenant,
		Project:     projectid,
		Description: description,
		Image:       img,
		ImageID:     img.ID,
		SSHPubKeys:  sshPubKeys,
		Cidr:        cidr,
	}
	res[0].Allocation = alloc
	res[0].Changed = time.Now()
	err = rs.UpdateDevice(&old, &res[0])
	if err != nil {
		cidrAllocator.Release(res[0].ID)
		return nil, fmt.Errorf("error when allocating device %q, %v", res[0].ID, err)
	}
	_, err = rs.waitTable().Get(res[0].ID).Update(res[0]).RunWrite(rs.session)
	if err != nil {
		cidrAllocator.Release(res[0].ID)
		rs.UpdateDevice(&res[0], &old)
		return nil, fmt.Errorf("cannot allocate device in DB: %v", err)
	}
	return &res[0], nil
}

// FreeDevice removes an allocation from a given device.
func (rs *RethinkStore) FreeDevice(id string) (*metal.Device, error) {
	device, err := rs.FindDevice(id)
	if err != nil {
		return nil, err
	}
	if device.Allocation == nil {
		return nil, fmt.Errorf("device is not allocated")
	}
	old := *device
	device.Allocation = nil
	err = rs.UpdateDevice(&old, device)
	if err != nil {
		return nil, fmt.Errorf("cannot clear device data: %v", err)
	}
	return device, nil
}

// RegisterDevice creates or updates a device in the database. It also creates
// an IPMI data record for this device.
// Maby it would be good to give a Device As Parameter, insted of all Atributes of a Device.
func (rs *RethinkStore) RegisterDevice(
	id string,
	site metal.Site,
	rackid string,
	sz metal.Size,
	hardware metal.DeviceHardware,
	ipmi metal.IPMI) (*metal.Device, error) {

	device, err := rs.FindDevice(id)
	name := fmt.Sprintf("%d-core/%s", hardware.CPUCores, humanize.Bytes(hardware.Memory))
	descr := fmt.Sprintf("a device with %d core(s) and %s of RAM", hardware.CPUCores, humanize.Bytes(hardware.Memory))
	if err != nil {
		if metal.IsNotFound(err) {
			device = &metal.Device{
				Base: metal.Base{
					ID:          id,
					Name:        name,
					Description: descr,
				},
				Size:     &sz,
				Site:     site,
				SiteID:   site.ID,
				RackID:   rackid,
				Hardware: hardware,
			}
			_, err = rs.CreateDevice(device)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else {
		old := *device
		device.Hardware = hardware
		device.Site = site
		device.SiteID = site.ID
		device.Size = &sz
		device.RackID = rackid
		device.Name = name
		device.Description = descr
		err = rs.UpdateDevice(&old, device)
		if err != nil {
			return nil, err
		}
	}
	err = rs.UpsertIPMI(id, &ipmi)
	if err != nil {
		return nil, err
	}

	return device, nil
}

// Wait inserts the device with the given ID in the waittable, so
// this device is ready for allocation. After this, this function waits
// for an update of this record in the waittable, which is a signal that
// this device is allocated. This allocation will be signaled via the
// given allocator in a separate goroutine. The allocator is a function
// which will receive a channel and the caller has to select on this
// channel to get a result. Using a channel allows the caller of this
// function to implement timeouts to not wait forever.
// The user of this function will block until this device is allocated.
func (rs *RethinkStore) Wait(id string, alloc Allocator) error {
	dev, err := rs.FindDevice(id)
	if err != nil {
		return err
	}
	a := make(chan metal.Device)

	// the device IS already allocated, so notify this allocation back.
	if dev.Allocation != nil {
		go func() {
			a <- *dev
		}()
		alloc(a)
		return nil
	}

	// does not prohibit concurrent wait calls for the same UUID
	_, err = rs.waitTable().Insert(dev, r.InsertOpts{
		Conflict: "replace",
	}).RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot insert device into wait table: %v", err)
	}
	defer func() {
		rs.waitTable().Get(id).Delete().RunWrite(rs.session)
	}()

	go func() {
		ch, err := rs.waitTable().Get(id).Changes().Run(rs.session)
		if err != nil {
			rs.Error("cannot wait for allocation", "error", err)
			// simply return so this device will not be allocated
			// the normal timeout-behaviour of the allocator will
			// occur without an allocation
			return
		}
		type responseType struct {
			NewVal metal.Device `rethinkdb:"new_val"`
		}
		var response responseType
		for ch.Next(&response) {
			if response.NewVal.ID == "" {
				// the entry was deleted, no wait any more
				break
			}
			res, err := rs.fillDeviceList(response.NewVal)
			if err != nil {
				rs.Error("device could not be populated", "error", err, "id", response.NewVal.ID)
				break
			}
			a <- res[0]
			break
		}
		rs.Info("stop waiting for changes", "id", id)
		close(a)
	}()
	alloc(a)
	return nil
}

// fillDeviceList fills the output fields of a device which are not directly
// stored in the database.
func (rs *RethinkStore) fillDeviceList(data ...metal.Device) ([]metal.Device, error) {
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

	all, err := rs.ListSites()
	if err != nil {
		return nil, err
	}
	sitemap := metal.Sites(all).ByID()

	res := make([]metal.Device, len(data), len(data))
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
		res[i].Site = sitemap[d.SiteID]
	}
	return res, nil
}
