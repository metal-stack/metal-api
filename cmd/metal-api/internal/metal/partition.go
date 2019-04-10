package metal

// A Partition represents a location.
type Partition struct {
	Base
	BootConfiguration  BootConfiguration `json:"bootconfig"`
	MgmtServiceAddress string            `json:"mgmtserviceaddress"`
}

// BootConfiguration defines the metal-hammer initrd, kernel and commandline
type BootConfiguration struct {
	ImageURL    string `json:"imageurl" description:"the url to download the initrd for the boot image" rethinkdb:"imageurl"`
	KernelURL   string `json:"kernelurl" description:"the url to download the kernel for the boot image" rethinkdb:"kernelurl"`
	CommandLine string `json:"commandline" description:"the cmdline to the kernel for the boot image" rethinkdb:"commandline"`
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
