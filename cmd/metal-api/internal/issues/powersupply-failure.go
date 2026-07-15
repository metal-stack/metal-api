package issues

import (
	"fmt"
	"strings"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

const (
	TypePowerSupplyFailure Type = "powersupply-failure"
)

type (
	issuePowerSupplyFailure struct {
		details string
	}
)

func (i *issuePowerSupplyFailure) Spec() *spec {
	return &spec{
		Type:        TypePowerSupplyFailure,
		Severity:    SeverityMajor,
		Description: "machine has power supply failures",
		RefURL:      "https://docs.metal-stack.io/stable/installation/troubleshoot/#power-supply-failure",
	}
}

func (i *issuePowerSupplyFailure) Evaluate(m metal.Machine, ec metal.ProvisioningEventContainer, c *Config) bool {
	for _, ps := range m.IPMI.PowerSupplies {
		if strings.ToLower(ps.Status.Health) != "ok" || strings.ToLower(ps.Status.State) != "enabled" {
			i.details = fmt.Sprintf("Health:%s State:%s", ps.Status.Health, ps.Status.State)
			return true
		}
	}
	return false
}

func (i *issuePowerSupplyFailure) Details() string {
	return i.details
}
