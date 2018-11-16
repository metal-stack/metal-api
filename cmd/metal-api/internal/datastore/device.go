package datastore

import (
	"fmt"
	"time"

	"git.f-i-ts.de/cloud-native/metal/metal-api/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

func (rs *RethinkStore) FindDevice(id string) (*metal.Device, error) {
	res, err := rs.deviceTable().Get(id).Run(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot get device from database: %v", err)
	}
	defer res.Close()
	var d metal.Device
	err = res.One(&d)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch results: %v", err)
	}
	if d.SizeID != "" {
		s, err := rs.FindSize(d.SizeID)
		if err != nil {
			return nil, fmt.Errorf("illegal size-id %q in device %q", d.SizeID, id)
		}
		d.Size = s
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

func (rs *RethinkStore) SearchDevice(projectid string, mac string, free *bool) ([]metal.Device, error) {
	q := *rs.deviceTable()
	if projectid != "" {
		q = q.Filter(map[string]interface{}{
			"project": projectid,
		})
	}
	if mac != "" {
		q = q.Filter(func(d r.Term) r.Term {
			return d.Field("macAddresses").Contains(mac)
		})
	}
	if free != nil {
		q = q.Filter(func(d r.Term) r.Term {
			if *free {
				return d.Field("project").Eq("")
			}
			return d.Field("project").Ne("")
		})
	}
	res, err := q.Run(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannt search devices from database: %v", err)
	}
	defer res.Close()
	data := make([]metal.Device, 0)
	err = res.All(&data)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch results: %v", err)
	}
	return rs.fillDeviceList(data...)
}

func (rs *RethinkStore) ListDevices() ([]metal.Device, error) {
	res, err := rs.deviceTable().Run(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot list devices from database: %v", err)
	}
	defer res.Close()
	data := make([]metal.Device, 0)
	err = res.All(&data)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch results: %v", err)
	}
	return rs.fillDeviceList(data...)
}

func (rs *RethinkStore) CreateDevice(d *metal.Device) error {
	d.Changed = time.Now()
	d.Created = d.Changed

	if d.Allocation != nil {
		return fmt.Errorf("a device cannot be created when it is allocated: %q: %+v", d.ID, *d.Allocation)
	}
	d.SizeID = d.Size.ID
	d.SiteID = d.Site.ID
	res, err := rs.deviceTable().Insert(d, r.InsertOpts{
		Conflict: "replace",
	}).RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot create device in database: %v", err)
	}
	if d.ID == "" {
		d.ID = res.GeneratedKeys[0]
	}
	return nil
}

func (rs *RethinkStore) FindIPMI(id string) (*metal.IPMI, error) {
	res, err := rs.ipmiTable().Get(id).Run(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot query ipmi data: %v", err)
	}
	if res.IsNil() {
		return nil, ErrNotFound
	}
	var ipmi metal.IPMI
	err = res.One(&ipmi)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch ipmi data: %v", err)
	}
	return &ipmi, nil
}

func (rs *RethinkStore) UpsertIpmi(id string, ipmi *metal.IPMI) error {
	ipmi.ID = id
	_, err := rs.ipmiTable().Insert(ipmi, r.InsertOpts{
		Conflict: "replace",
	}).RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot create ipmi in database: %v", err)
	}
	return nil
}

func (rs *RethinkStore) DeleteDevice(id string) (*metal.Device, error) {
	d, err := rs.FindDevice(id)
	if err != nil {
		return nil, fmt.Errorf("cannot find device with id %q: %v", id, err)
	}
	_, err = rs.deviceTable().Get(id).Delete().RunWrite(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot delete device from database: %v", err)
	}
	return d, nil
}

func (rs *RethinkStore) UpdateDevice(oldD *metal.Device, newD *metal.Device) error {
	_, err := rs.deviceTable().Get(oldD.ID).Replace(func(row r.Term) r.Term {
		return r.Branch(row.Field("changed").Eq(r.Expr(oldD.Changed)), newD, r.Error("the device was changed from another, please retry"))
	}).RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot update device: %v", err)
	}
	return nil
}

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
	cidr, err := cidrAllocator(res[0].ID, tenant, projectid, name, description, img.Name)
	if err != nil {
		return nil, fmt.Errorf("cannot allocate at netbox: %v", err)
	}

	rs.fillDeviceList(res[0:1]...)
	alloc := &metal.DeviceAllocation{
		Name:        name,
		Hostname:    hostname,
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
		return nil, fmt.Errorf("error when allocating device %q, %v", res[0].ID, err)
	}
	_, err = rs.waitTable().Get(res[0].ID).Update(res[0]).RunWrite(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot allocate device in DB: %v", err)
	}
	return &res[0], nil
}

func (rs *RethinkStore) FreeDevice(id string) (*metal.Device, error) {
	device, err := rs.FindDevice(id)
	if err != nil {
		return nil, fmt.Errorf("cannot free device: %v", err)
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

func (rs *RethinkStore) RegisterDevice(
	id string,
	site metal.Site,
	sz metal.Size,
	hardware metal.DeviceHardware,
	ipmi metal.IPMI) (*metal.Device, error) {

	device, err := rs.FindDevice(id)
	if err != nil {
		device = &metal.Device{
			ID:       id,
			Size:     &sz,
			Site:     site,
			SiteID:   site.ID,
			Hardware: hardware,
		}
		err = rs.CreateDevice(device)
		if err != nil {
			return nil, err
		}
	} else {
		old := *device
		device.Hardware = hardware
		device.Site = site
		device.SiteID = site.ID
		device.Size = &sz

		err = rs.UpdateDevice(&old, device)
		if err != nil {
			return nil, err
		}
	}
	err = rs.UpsertIpmi(id, &ipmi)
	if err != nil {
		return nil, err
	}

	return device, nil
}

func (rs *RethinkStore) Wait(id string, alloc Allocator) error {
	dev, err := rs.FindDevice(id)
	if err != nil {
		return fmt.Errorf("cannot wait for unknown device: %v", err)
	}
	a := make(chan metal.Device)

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

func (rs *RethinkStore) fillDeviceList(data ...metal.Device) ([]metal.Device, error) {
	allsz, err := rs.ListSizes()
	if err != nil {
		return nil, fmt.Errorf("cannot query all sizes: %v", err)
	}
	szmap := metal.Sizes(allsz).ByID()

	allimg, err := rs.ListImages()
	if err != nil {
		return nil, fmt.Errorf("cannot query all images: %v", err)
	}
	imgmap := metal.Images(allimg).ByID()

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
	}
	return res, nil
}
