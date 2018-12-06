package metal

// An Image describes an image which could be used for provisioning.
type Image struct {
	Base
	URL string `json:"url" modelDescription:"An image that can be put on a device."  description:"the url to this image" rethinkdb:"url"`
}

type Images []Image
type ImageMap map[string]Image

func (ii Images) ByID() ImageMap {
	res := make(ImageMap)
	for i, f := range ii {
		res[f.ID] = ii[i]
	}
	return res
}
