package issues

import "github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"

const (
	IssueTypeLivelinessUnknown IssueType = "liveliness-unknown"
)

type (
	IssueLivelinessUnknown struct{}
)

func (i *IssueLivelinessUnknown) Spec() *issueSpec {
	return &issueSpec{
		Type:        IssueTypeLivelinessUnknown,
		Severity:    IssueSeverityMajor,
		Description: "the machine is not sending LLDP alive messages anymore",
		RefURL:      "https://docs.metal-stack.io/stable/installation/troubleshoot/#liveliness-unknown",
	}
}

func (i *IssueLivelinessUnknown) Evaluate(m metal.Machine, ec metal.ProvisioningEventContainer, c *IssueConfig) bool {
	return ec.Liveliness.Is(string(metal.MachineLivelinessUnknown))
}

func (i *IssueLivelinessUnknown) Details() string {
	return ""
}
