package issues

import (
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

const (
	TypeNoEventContainer Type = "no-event-container"
)

type (
	IssueNoEventContainer struct{}
)

func (i *IssueNoEventContainer) Spec() *spec {
	return &spec{
		Type:        TypeNoEventContainer,
		Severity:    SeverityMajor,
		Description: "machine has no event container",
		RefURL:      "https://docs.metal-stack.io/stable/installation/troubleshoot/#no-event-container",
	}
}

func (i *IssueNoEventContainer) Evaluate(m metal.Machine, ec metal.ProvisioningEventContainer, c *Config) bool {
	return ec.Base.ID == ""
}

func (i *IssueNoEventContainer) Details() string {
	return ""
}
