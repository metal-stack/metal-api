package issues

import "github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"

const (
	IssueTypeBMCWithoutIP IssueType = "bmc-without-ip"
)

type (
	IssueBMCWithoutIP struct{}
)

func (i *IssueBMCWithoutIP) Spec() *issueSpec {
	return &issueSpec{
		Type:        IssueTypeBMCWithoutIP,
		Severity:    IssueSeverityMajor,
		Description: "BMC has no ip address",
		RefURL:      "https://docs.metal-stack.io/stable/installation/troubleshoot/#bmc-without-ip",
	}
}

func (i *IssueBMCWithoutIP) Evaluate(m metal.Machine, ec metal.ProvisioningEventContainer, c *IssueConfig) bool {
	return m.IPMI.Address == ""
}

func (i *IssueBMCWithoutIP) Details() string {
	return ""
}
