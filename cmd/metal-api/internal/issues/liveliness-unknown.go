package issues

import "github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"

const (
	TypeLivelinessUnknown Type = "liveliness-unknown"
)

type (
	IssueLivelinessUnknown struct{}
)

func (i *IssueLivelinessUnknown) Spec() *spec {
	return &spec{
		Type:        TypeLivelinessUnknown,
		Severity:    SeverityMajor,
		Description: "the machine is not sending LLDP alive messages anymore",
		RefURL:      "https://docs.metal-stack.io/stable/installation/troubleshoot/#liveliness-unknown",
	}
}

func (i *IssueLivelinessUnknown) Evaluate(m metal.Machine, ec metal.ProvisioningEventContainer, c *Config) bool {
	return ec.Liveliness.Is(string(metal.MachineLivelinessUnknown))
}

func (i *IssueLivelinessUnknown) Details() string {
	return ""
}
