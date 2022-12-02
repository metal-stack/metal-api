package metal

// A Partition represents a location.
type Partition struct {
	Base
	BootConfiguration          BootConfiguration `rethinkdb:"bootconfig" json:"bootconfig"`
	MgmtServiceAddress         string            `rethinkdb:"mgmtserviceaddr" json:"mgmtserviceaddr"`
	PrivateNetworkPrefixLength uint8             `rethinkdb:"privatenetworkprefixlength" json:"privatenetworkprefixlength"`
	WaitingPoolMinSize         string            `rethinkdb:"waitingpoolminsize" json:"waitingpoolminsize"`
	WaitingPoolMaxSize         string            `rethinkdb:"waitingpoolmaxsize" json:"waitingpoolmaxsize"`
}

// BootConfiguration defines the metal-hammer initrd, kernel and commandline
type BootConfiguration struct {
	ImageURL    string `rethinkdb:"imageurl" json:"imageurl"`
	KernelURL   string `rethinkdb:"kernelurl" json:"kernelurl"`
	CommandLine string `rethinkdb:"commandline" json:"commandline"`
}

// Partitions is a list of partitions.
type Partitions []Partition

// PartitionMap is an indexed map of partitions
type PartitionMap map[string]Partition

// ByID creates an indexed map of partitions whre the id is the index.
func (sz Partitions) ByID() PartitionMap {
	res := make(PartitionMap)
	for i, s := range sz {
		res[s.ID] = sz[i]
	}
	return res
}
