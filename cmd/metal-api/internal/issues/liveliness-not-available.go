package issues

import "github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"

const (
	IssueTypeLivelinessNotAvailable IssueType = "liveliness-not-available"
)

type (
	IssueLivelinessNotAvailable struct{}
)

func (i *IssueLivelinessNotAvailable) Spec() *spec {
	return &spec{
		Type:        IssueTypeLivelinessNotAvailable,
		Severity:    IssueSeverityMinor,
		Description: "the machine liveliness is not available",
	}
}

func (i *IssueLivelinessNotAvailable) Evaluate(m metal.Machine, ec metal.ProvisioningEventContainer, c *IssueConfig) bool {
	allowed := map[metal.MachineLiveliness]bool{
		metal.MachineLivelinessAlive:   true,
		metal.MachineLivelinessDead:    true,
		metal.MachineLivelinessUnknown: true,
	}

	return !allowed[ec.Liveliness]
}

func (i *IssueLivelinessNotAvailable) Details() string {
	return ""
}
