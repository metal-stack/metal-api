package metal

type Site struct {
	Base
}

type Sites []Site
type SiteMap map[string]Site

func (sz Sites) ByID() SiteMap {
	res := make(SiteMap)
	for i, s := range sz {
		res[s.ID] = sz[i]
	}
	return res
}
