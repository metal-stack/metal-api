package issues

import "github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"

const (
	IssueTypeNoPartition IssueType = "no-partition"
)

type (
	IssueNoPartition struct{}
)

func (i *IssueNoPartition) Spec() *issueSpec {
	return &issueSpec{
		Type:        IssueTypeNoPartition,
		Severity:    IssueSeverityMajor,
		Description: "machine with no partition",
		RefURL:      "https://docs.metal-stack.io/stable/installation/troubleshoot/#no-partition",
	}
}

func (i *IssueNoPartition) Evaluate(m metal.Machine, ec metal.ProvisioningEventContainer, c *IssueConfig) bool {
	return m.PartitionID == ""
}

func (i *IssueNoPartition) Details() string {
	return ""
}
