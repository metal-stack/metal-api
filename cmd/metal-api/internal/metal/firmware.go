package metal

type FirmwareKind = string

const (
	FirmwareBIOS FirmwareKind = "bios"
	FirmwareBMC  FirmwareKind = "bmc"
)

var FirmwareKinds = []string{
	FirmwareBIOS,
	FirmwareBMC,
}
