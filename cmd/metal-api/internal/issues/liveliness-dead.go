package issues

import "github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"

const (
	IssueTypeLivelinessDead IssueType = "liveliness-dead"
)

type (
	IssueLivelinessDead struct{}
)

func (i *IssueLivelinessDead) Spec() *issueSpec {
	return &issueSpec{
		Type:        IssueTypeLivelinessDead,
		Severity:    IssueSeverityMajor,
		Description: "the machine is not sending events anymore",
		RefURL:      "https://docs.metal-stack.io/stable/installation/troubleshoot/#liveliness-dead",
	}
}

func (i *IssueLivelinessDead) Evaluate(m metal.Machine, ec metal.ProvisioningEventContainer, c *IssueConfig) bool {
	return ec.Liveliness.Is(string(metal.MachineLivelinessDead))
}

func (i *IssueLivelinessDead) Details() string {
	return ""
}
