package metal

// A Site represents a location.
type Site struct {
	Base
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
