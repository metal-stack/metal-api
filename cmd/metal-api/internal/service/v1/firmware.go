package v1

type Firmware struct {
	Vendor      string
	Board       string
	BmcVersion  string
	BiosVersion string
	Revision    string
}

type FirmwaresResponse struct {
	Firmwares map[string]FirmwaresRevisions `json:"firmwares" description:"map of firmware revisions per kind"`
}

type FirmwaresRevisions struct {
	Kind      string                         `json:"kind" description:"the firmware kind to which the contained firmwares belong"`
	Revisions map[string]map[string][]string `json:"revisions" description:"list of firmwares per board per vendor"`
}

type MachineUpdateFirmwareRequest struct {
	Kind        string `json:"kind" description:"the firmware kind, i.e. 'bios' of 'bmc'"`
	Revision    string `json:"revision" description:"the update revision"`
	Description string `json:"description" description:"a description why the machine has been updated"`
}
