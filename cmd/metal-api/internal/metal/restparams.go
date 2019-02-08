package metal

// AllocateMachine must be sent by a client who wants to allocate a new machine.
type AllocateMachine struct {
	Name        string   `json:"name" description:"the new name for the allocated machine" optional:"true"`
	Tenant      string   `json:"tenant" description:"the name of the owning tenant"`
	Hostname    string   `json:"hostname" description:"the hostname for the allocated machine"`
	Description string   `json:"description" description:"the description for the allocated machine" optional:"true"`
	ProjectID   string   `json:"projectid" description:"the project id to assign this machine to"`
	PartitionID string   `json:"partitionid" description:"the partition id to assign this machine to"`
	SizeID      string   `json:"sizeid" description:"the size id to assign this machine to"`
	ImageID     string   `json:"imageid" description:"the image id to assign this machine to"`
	SSHPubKeys  []string `json:"ssh_pub_keys" description:"the public ssh keys to access the machine with"`
	UserData    string   `json:"user_data,omitempty" description:"cloud-init.io compatible userdata must be base64 encoded." optional:"true" rethinkdb:"userdata"`
}

// RegisterMachine must be sent by a machine, when it boots with our image and
// reports its capabilities.
type RegisterMachine struct {
	UUID        string          `json:"uuid" description:"the product uuid of the machine to register"`
	PartitionID string          `json:"partitionid" description:"the partition id to register this machine with"`
	RackID      string          `json:"rackid" description:"the rack id where this machine is connected to"`
	Hardware    MachineHardware `json:"hardware" description:"the hardware of this machine"`
	IPMI        IPMI            `json:"ipmi" description:"the ipmi access infos"`
}

// PhoneHomeRequest is sent by a regular phone home of a machine.
type PhoneHomeRequest struct {
	PhoneHomeToken string `json:"phone_home_token" description:"the jwt that was issued for the machine"`
}

// An ReportAllocation is sent to the api after a machine was successfully
// allocated and provisioned.
type ReportAllocation struct {
	Success         bool   `json:"success" description:"signals if the allocation was successful" optional:"false"`
	ErrorMessage    string `json:"errormessage" description:"contains an errormessage when there was no success" optional:"true"`
	ConsolePassword string `json:"console_password" description:"the console password which was generated while provisioning" optional:"false"`
}

// RegisterSwitch must be sent by a switch at least when it starts up.
type RegisterSwitch struct {
	ID          string `json:"id" description:"a unique ID" unique:"true"`
	Nics        Nics   `json:"nics" description:"the list of network interfaces on the switch"`
	PartitionID string `json:"partition_id" description:"the id of the partition in which this switch is located"`
	RackID      string `json:"rack_id" description:"the id of the rack in which this switch is located"`
}
