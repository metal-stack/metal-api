package issues

import "github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"

const (
	TypeBMCWithoutMAC Type = "bmc-without-mac"
)

type (
	issueBMCWithoutMAC struct{}
)

func (i *issueBMCWithoutMAC) Spec() *spec {
	return &spec{
		Type:        TypeBMCWithoutMAC,
		Severity:    SeverityMajor,
		Description: "BMC has no mac address",
		RefURL:      "https://docs.metal-stack.io/stable/installation/troubleshoot/#bmc-without-mac",
	}
}

func (i *issueBMCWithoutMAC) Evaluate(m metal.Machine, ec metal.ProvisioningEventContainer, c *Config) bool {
	return m.IPMI.MacAddress == ""
}

func (i *issueBMCWithoutMAC) Details() string {
	return ""
}
