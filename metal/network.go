package metal

// Nic information.
type Nic struct {
	MacAddress string `json:"mac"  description:"the mac address of this network interface" rethinkdb:"macAddress"`
	Name       string `json:"name"  description:"the name of this network interface" rethinkdb:"name"`
	Neighbors  []Nic  `json:"neighbors" description:"the neighbors visible to this network interface"`
}
