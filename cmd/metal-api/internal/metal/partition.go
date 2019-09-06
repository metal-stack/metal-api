package metal

// A Partition represents a location.
type Partition struct {
	Base
	BootConfiguration          BootConfiguration `rethinkdb:"bootconfig"`
	MgmtServiceAddress         string            `rethinkdb:"mgmtserviceaddr"`
	PrivateNetworkPrefixLength int               `rethinkdb:"privatenetworkprefixlength"`
}

// BootConfiguration defines the metal-hammer initrd, kernel and commandline
type BootConfiguration struct {
	ImageURL    string `rethinkdb:"imageurl"`
	KernelURL   string `rethinkdb:"kernelurl"`
	CommandLine string `rethinkdb:"commandline"`
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
