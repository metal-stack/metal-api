package metal

// A Site represents a location.
type Site struct {
	Base
	BootConfiguration BootConfiguration `json:"bootconfig"`
}

// BootConfiguration defines the metal-hammer initrd, kernel and commandline
type BootConfiguration struct {
	ImageURL    string `json:"imageurl" description:"the url to download the initrd for the boot image" rethinkdb:"imageurl"`
	KernelURL   string `json:"kernelurl" description:"the url to download the kernel for the boot image" rethinkdb:"kernelurl"`
	CommandLine string `json:"commandline" description:"the cmdline to the kernel for the boot image" rethinkdb:"commandline"`
}

// Sites is a list of sites.
type Sites []Site

// SiteMap is an indexed map of sites
type SiteMap map[string]Site

// ByID creates an indexed map of sites whre the id is the index.
func (sz Sites) ByID() SiteMap {
	res := make(SiteMap)
	for i, s := range sz {
		res[s.ID] = sz[i]
	}
	return res
}
