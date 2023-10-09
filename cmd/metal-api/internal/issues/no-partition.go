package issues

import "github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"

const (
	TypeNoPartition Type = "no-partition"
)

type (
	IssueNoPartition struct{}
)

func (i *IssueNoPartition) Spec() *spec {
	return &spec{
		Type:        TypeNoPartition,
		Severity:    SeverityMajor,
		Description: "machine with no partition",
		RefURL:      "https://docs.metal-stack.io/stable/installation/troubleshoot/#no-partition",
	}
}

func (i *IssueNoPartition) Evaluate(m metal.Machine, ec metal.ProvisioningEventContainer, c *Config) bool {
	return m.PartitionID == ""
}

func (i *IssueNoPartition) Details() string {
	return ""
}
