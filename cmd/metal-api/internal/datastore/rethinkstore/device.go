package rethinkstore

import (
	"fmt"
	"time"

	"git.f-i-ts.de/cloud-native/maas/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/maas/metal-api/cmd/metal-api/internal/ipam"
	"git.f-i-ts.de/cloud-native/maas/metal-api/pkg/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

func (rs *RethinkStore) FindDevice(id string) (*metal.Device, error) {
	res, err := rs.deviceTable.Get(id).Run(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot get device from database: %v", err)
	}
	defer res.Close()
	var d metal.Device
	err = res.One(&d)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch results: %v", err)
	}
	if d.FacilityID != "" {
		f, err := rs.FindFacility(d.FacilityID)
		if err != nil {
			return nil, fmt.Errorf("illegal facility-id %q in device %q", d.FacilityID, id)
		}
		d.Facility = *f
	}
	if d.SizeID != "" {
		s, err := rs.FindSize(d.SizeID)
		if err != nil {
			return nil, fmt.Errorf("illegal size-id %q in device %q", d.SizeID, id)
		}
		d.Size = s
	}
	if d.ImageID != "" {
		f, err := rs.FindImage(d.ImageID)
		if err != nil {
			return nil, fmt.Errorf("illegal imageid-id %q in device %q", d.ImageID, id)
		}
		d.Image = f
	}
	return &d, nil
}

func (rs *RethinkStore) SearchDevice(projectid string, mac string, free *bool) ([]metal.Device, error) {
	q := rs.deviceTable
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
	return rs.fillDeviceList(data)
}

func (rs *RethinkStore) ListDevices() ([]metal.Device, error) {
	res, err := rs.deviceTable.Run(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot list devices from database: %v", err)
	}
	defer res.Close()
	data := make([]metal.Device, 0)
	err = res.All(&data)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch results: %v", err)
	}
	return rs.fillDeviceList(data)
}

func (rs *RethinkStore) CreateDevice(d *metal.Device) error {
	d.Changed = time.Now()
	d.Created = d.Changed

	if d.Image != nil {
		d.ImageID = d.Image.ID
	}
	d.SizeID = d.Size.ID
	d.FacilityID = d.Facility.ID
	res, err := rs.deviceTable.Insert(d).RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot create device in database: %v", err)
	}
	if d.ID == "" {
		d.ID = res.GeneratedKeys[0]
	}
	return nil
}

func (rs *RethinkStore) DeleteDevice(id string) (*metal.Device, error) {
	d, err := rs.FindDevice(id)
	if err != nil {
		return nil, fmt.Errorf("cannot find device with id %q: %v", id, err)
	}
	_, err = rs.deviceTable.Get(id).Delete().RunWrite(rs.session)
	if err != nil {
		return nil, fmt.Errorf("cannot delete device from database: %v", err)
	}
	return d, nil
}

func (rs *RethinkStore) UpdateDevice(oldD *metal.Device, newD *metal.Device) error {
	_, err := rs.deviceTable.Get(oldD.ID).Replace(func(row r.Term) r.Term {
		return r.Branch(row.Field("changed").Eq(r.Expr(oldD.Changed)), newD, r.Error("the device was changed from another, please retry"))
	}).RunWrite(rs.session)
	if err != nil {
		return fmt.Errorf("cannot update size: %v", err)
	}
	return nil
}

func (rs *RethinkStore) AllocateDevice(name string, description string, hostname string, projectid string, facilityid string, sizeid string, imageid string, sshPubKey string) (*metal.Device, error) {
	image, err := rs.FindImage(imageid)
	if err != nil {
		return nil, fmt.Errorf("image with id %q not found", imageid)
	}
	available, err := rs.waitTable.Filter(map[string]interface{}{
		"project":    "",
		"facilityid": facilityid,
		"sizeid":     sizeid,
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
		return nil, datastore.ErrNoDeviceAvailable
	}
	ip, err := ipam.AllocateIP()
	if err != nil {
		return nil, err
	}

	old := res[0]
	rs.fillDeviceList(res[0:1])
	res[0].Name = name
	res[0].Hostname = hostname
	res[0].Project = projectid
	res[0].Description = description
	res[0].Image = image
	res[0].ImageID = image.ID
	res[0].SSHPubKey = sshPubKey
	res[0].IP = ip
	res[0].Changed = time.Now()
	err = rs.UpdateDevice(&old, &res[0])
	if err != nil {
		return nil, fmt.Errorf("error when allocating device %q, %v", res[0].ID, err)
	}
	_, err = rs.waitTable.Get(res[0].ID).Update(res[0]).RunWrite(rs.session)
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
	old := *device
	ipam.FreeIP(device.IP)
	device.Name, device.Project, device.Description, device.IP, device.Hostname, device.SSHPubKey = "", "", "", "", "", ""
	err = rs.UpdateDevice(&old, device)
	if err != nil {
		return nil, fmt.Errorf("cannot clear device data: %v", err)
	}
	return device, nil
}

func (rs *RethinkStore) RegisterDevice(id string, facilityid string, hardware metal.DeviceHardware) (*metal.Device, error) {
	fc, err := rs.FindFacility(facilityid)
	if err != nil {
		return nil, fmt.Errorf("facility with id %q not found", facilityid)
	}

	sz := rs.determineSizeFromHardware(hardware)

	device, err := rs.FindDevice(id)
	if err != nil {
		device = &metal.Device{
			ID:       id,
			Size:     sz,
			Facility: *fc,
			Hardware: hardware,
		}
		err = rs.CreateDevice(device)
		if err != nil {
			return nil, err
		}
		return device, nil
	}
	old := *device
	device.Hardware = hardware
	device.Facility = *fc
	device.Size = sz

	err = rs.UpdateDevice(&old, device)
	if err != nil {
		return nil, err
	}

	return device, nil
}

func (rs *RethinkStore) Wait(id string, alloc datastore.Allocator) error {
	dev, err := rs.FindDevice(id)
	if err != nil {
		return fmt.Errorf("cannot wait for unknown device: %v", err)
	}
	if dev.Project != "" {
		return fmt.Errorf("device is already allocated, needs to be released first")
	}
	res, err := rs.waitTable.Insert(dev).Run(rs.session)
	if err != nil {
		return fmt.Errorf("cannot create device in wait table: %v", err)
	}
	defer func() {
		rs.waitTable.Get(id).Delete().Run(rs.session)
		res.Close()
	}()
	a := make(datastore.Allocation)
	go func() {
		ch, err := rs.waitTable.Get(id).Changes().Run(rs.session)
		if err != nil {
			rs.Error("cannot wait for allocation", "error", err)
			// simply return so this device will not be allocated
			// the normal timeout-behaviour of the allocator will
			// occur without an allocation
			return
		}

		var response metal.Device
		for ch.Next(&response) {
			res, err := rs.fillDeviceList([]metal.Device{response})
			if err != nil {
				rs.Logger.Error("Device could not be populated", "error", err, "id", response.ID)
				continue
			}
			a <- res[0]
			return
		}

	}()
	alloc(a)
	return fmt.Errorf("cannot fetch changed device")
}

func (rs *RethinkStore) fillDeviceList(data []metal.Device) ([]metal.Device, error) {
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
	allfacs, err := rs.ListFacilities()
	if err != nil {
		return nil, fmt.Errorf("cannot query all facilities: %v", err)
	}
	facmap := metal.Facilities(allfacs).ByID()

	for i, d := range data {
		data[i].Facility = facmap[d.FacilityID]
		size := szmap[d.SizeID]
		data[i].Size = &size
		if d.ImageID != "" {
			img := imgmap[d.ImageID]
			data[i].Image = &img
		}
	}
	return data, nil
}
