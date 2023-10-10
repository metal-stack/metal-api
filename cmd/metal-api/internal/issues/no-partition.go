package issues

import "github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"

const (
	TypeNoPartition Type = "no-partition"
)

type (
	issueNoPartition struct{}
)

func (i *issueNoPartition) Spec() *spec {
	return &spec{
		Type:        TypeNoPartition,
		Severity:    SeverityMajor,
		Description: "machine with no partition",
		RefURL:      "https://docs.metal-stack.io/stable/installation/troubleshoot/#no-partition",
	}
}

func (i *issueNoPartition) Evaluate(m metal.Machine, ec metal.ProvisioningEventContainer, c *Config) bool {
	return m.PartitionID == ""
}

func (i *issueNoPartition) Details() string {
	return ""
}
