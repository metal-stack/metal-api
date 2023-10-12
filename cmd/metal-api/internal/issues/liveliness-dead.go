package issues

import "github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"

const (
	TypeLivelinessDead Type = "liveliness-dead"
)

type (
	issueLivelinessDead struct{}
)

func (i *issueLivelinessDead) Spec() *spec {
	return &spec{
		Type:        TypeLivelinessDead,
		Severity:    SeverityMajor,
		Description: "the machine is not sending events anymore",
		RefURL:      "https://docs.metal-stack.io/stable/installation/troubleshoot/#liveliness-dead",
	}
}

func (i *issueLivelinessDead) Evaluate(m metal.Machine, ec metal.ProvisioningEventContainer, c *Config) bool {
	return ec.Liveliness == metal.MachineLivelinessDead
}

func (i *issueLivelinessDead) Details() string {
	return ""
}
