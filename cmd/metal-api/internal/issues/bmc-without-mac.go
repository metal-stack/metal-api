package issues

import "github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"

const (
	IssueTypeBMCWithoutMAC IssueType = "bmc-without-mac"
)

type (
	IssueBMCWithoutMAC struct{}
)

func (i *IssueBMCWithoutMAC) Spec() *issueSpec {
	return &issueSpec{
		Type:        IssueTypeBMCWithoutMAC,
		Severity:    IssueSeverityMajor,
		Description: "BMC has no mac address",
		RefURL:      "https://docs.metal-stack.io/stable/installation/troubleshoot/#bmc-without-mac",
	}
}

func (i *IssueBMCWithoutMAC) Evaluate(m metal.Machine, ec metal.ProvisioningEventContainer, c *IssueConfig) bool {
	return m.IPMI.MacAddress == ""
}

func (i *IssueBMCWithoutMAC) Details() string {
	return ""
}
