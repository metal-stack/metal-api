package metal

type MacAddress string

// Nic information.
type Nic struct {
	MacAddress MacAddress `json:"mac"  description:"the mac address of this network interface" rethinkdb:"macAddress"`
	Name       string     `json:"name"  description:"the name of this network interface" rethinkdb:"name"`
	Neighbors  Nics       `json:"neighbors" description:"the neighbors visible to this network interface"`
}

type Nics []Nic

func (nics Nics) ByMac() map[MacAddress]*Nic {
	res := make(map[MacAddress]*Nic)
	for i, n := range nics {
		res[n.MacAddress] = &nics[i]
	}
	return res
}
