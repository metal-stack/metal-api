package metal

// RegisterSwitch must be sent by a switch at least when it starts up.
type RegisterSwitch struct {
	ID          string `json:"id" description:"a unique ID" unique:"true"`
	Nics        Nics   `json:"nics" description:"the list of network interfaces on the switch"`
	PartitionID string `json:"partition_id" description:"the id of the partition in which this switch is located"`
	RackID      string `json:"rack_id" description:"the id of the rack in which this switch is located"`
}
