package v1

type Firmware struct {
	Vendor      string
	Board       string
	BmcVersion  string
	BiosVersion string
}

type Firmwares struct {
	Kind            string            `json:"kind" description:"the firmware kind to which the contained firmwares belong"`
	VendorFirmwares []VendorFirmwares `json:"vendor_firmwares" description:"list of firmwares per vendor"`
}

type VendorFirmwares struct {
	Vendor         string           `json:"vendor" description:"the vendor to which the contained firmwares belong"`
	BoardFirmwares []BoardFirmwares `json:"board_firmwares" description:"list of firmwares per board"`
}

type BoardFirmwares struct {
	Board     string   `json:"board" description:"the board to which the contained firmwares belong"`
	Revisions []string `json:"revisions" description:"list of firmwares revisions"`
}

type MachineUpdateFirmware struct {
	Kind        string `json:"kind" description:"the firmware kind, i.e. 'bios' of 'bmc'"`
	Revision    string `json:"revision" description:"the update revision"`
	Description string `json:"description" description:"a description why the machine has been updated"`
}
