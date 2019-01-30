package metal

// AllocateDevice must be sent by a client who wants to allocate a new device.
type AllocateDevice struct {
	Name        string   `json:"name" description:"the new name for the allocated device" optional:"true"`
	Tenant      string   `json:"tenant" description:"the name of the owning tenant"`
	Hostname    string   `json:"hostname" description:"the hostname for the allocated device"`
	Description string   `json:"description" description:"the description for the allocated device" optional:"true"`
	ProjectID   string   `json:"projectid" description:"the project id to assign this device to"`
	SiteID      string   `json:"siteid" description:"the site id to assign this device to"`
	SizeID      string   `json:"sizeid" description:"the size id to assign this device to"`
	ImageID     string   `json:"imageid" description:"the image id to assign this device to"`
	SSHPubKeys  []string `json:"ssh_pub_keys" description:"the public ssh keys to access the device with"`
	UserData    string   `json:"user_data,omitempty" description:"cloud-init.io compatible userdata" optional:"true" rethinkdb:"userdata"`
}

// RegisterDevice must be sent by a device, when it boots with our image and
// reports its capabilities.
type RegisterDevice struct {
	UUID     string         `json:"uuid" description:"the product uuid of the device to register"`
	SiteID   string         `json:"siteid" description:"the site id to register this device with"`
	RackID   string         `json:"rackid" description:"the rack id where this device is connected to"`
	Hardware DeviceHardware `json:"hardware" description:"the hardware of this device"`
	IPMI     IPMI           `json:"ipmi" description:"the ipmi access infos"`
}

// PhoneHomeRequest is sent by a regular phone home of a device.
type PhoneHomeRequest struct {
	PhoneHomeToken string `json:"phone_home_token" description:"the jwt that was issued for the device"`
}

// An ReportAllocation is sent to the api after a device was successfully
// allocated and provisioned.
type ReportAllocation struct {
	Success         bool   `json:"success" description:"signals if the allocation was successful" optional:"false"`
	ErrorMessage    string `json:"errormessage" description:"contains an errormessage when there was no success" optional:"true"`
	ConsolePassword string `json:"console_password" description:"the console password which was generated while provisioning" optional:"false"`
}

// RegisterSwitch must be sent by a switch at least when it starts up.
type RegisterSwitch struct {
	ID     string `json:"id" description:"a unique ID" unique:"true"`
	Nics   Nics   `json:"nics" description:"the list of network interfaces on the switch"`
	SiteID string `json:"site_id" description:"the id of the site in which this switch is located"`
	RackID string `json:"rack_id" description:"the id of the rack in which this switch is located"`
}
