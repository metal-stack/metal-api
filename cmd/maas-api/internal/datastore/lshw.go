package datastore

// A LshwInformation contains the required fields from the discovered information data. We only
// declare the fields which are needed, not a full LSHW model because we are not sure if the
// transported data is always identical over all hardware types.
type LshwInformation struct {
	Configuration struct {
		UUID string `json:"uuid"`
	} `json:"configuration"`
}

type LshwElement map[string]interface{}

func SearchNetworkEntries(data map[string]interface{}, result *[]LshwElement) {
	clzz, has := data["class"]
	if !has {
		return
	}
	if clzz == "network" {
		*result = append(*result, data)
	}
	child, has := data["children"]
	if has {
		childs := child.([]interface{})
		for i := range childs {
			cc := childs[i]
			SearchNetworkEntries(cc.(map[string]interface{}), result)
		}
	}
}
