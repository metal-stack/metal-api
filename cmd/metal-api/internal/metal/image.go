package metal

// An Image describes an image which could be used for provisioning.
type Image struct {
	Base
	URL string `json:"url" modelDescription:"An image that can be put on a machine."  description:"the url to this image" rethinkdb:"url"`
}

// Images is a collection of images.
type Images []Image

// ImageMap is an indexed map for images.
type ImageMap map[string]Image

// ByID creates an indexed map from an image collection.
func (ii Images) ByID() ImageMap {
	res := make(ImageMap)
	for i, f := range ii {
		res[f.ID] = ii[i]
	}
	return res
}
