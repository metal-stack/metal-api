package v1

type Firmware struct {
	Vendor      string
	Board       string
	BmcVersion  string
	BiosVersion string
	Revision    string
}

type FirmwaresResponse struct {
	Revisions map[string]VendorRevisions `json:"revisions" description:"list of firmwares per board per vendor per kind"`
}

type VendorRevisions struct {
	VendorRevisions map[string]BoardRevisions
}

type BoardRevisions struct {
	BoardRevisions map[string][]string
}

type MachineUpdateFirmwareRequest struct {
	Kind        string `json:"kind" enum:"bios|bmc" description:"the firmware kind, i.e. [bios|bmc]"`
	Revision    string `json:"revision" description:"the update revision"`
	Description string `json:"description" description:"a description why the machine has been updated"`
}
