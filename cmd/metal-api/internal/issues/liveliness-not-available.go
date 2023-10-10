package issues

import "github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"

const (
	TypeLivelinessNotAvailable Type = "liveliness-not-available"
)

type (
	issueLivelinessNotAvailable struct{}
)

func (i *issueLivelinessNotAvailable) Spec() *spec {
	return &spec{
		Type:        TypeLivelinessNotAvailable,
		Severity:    SeverityMinor,
		Description: "the machine liveliness is not available",
	}
}

func (i *issueLivelinessNotAvailable) Evaluate(m metal.Machine, ec metal.ProvisioningEventContainer, c *Config) bool {
	allowed := map[metal.MachineLiveliness]bool{
		metal.MachineLivelinessAlive:   true,
		metal.MachineLivelinessDead:    true,
		metal.MachineLivelinessUnknown: true,
	}

	return !allowed[ec.Liveliness]
}

func (i *issueLivelinessNotAvailable) Details() string {
	return ""
}
