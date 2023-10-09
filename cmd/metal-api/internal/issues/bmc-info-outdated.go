package issues

import (
	"fmt"
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

const (
	TypeBMCInfoOutdated Type = "bmc-info-outdated"
)

type (
	IssueBMCInfoOutdated struct {
		details string
	}
)

func (i *IssueBMCInfoOutdated) Details() string {
	return i.details
}

func (i *IssueBMCInfoOutdated) Evaluate(m metal.Machine, ec metal.ProvisioningEventContainer, c *Config) bool {
	if m.IPMI.LastUpdated.IsZero() {
		i.details = "machine ipmi has never been set"
		return true
	}

	lastUpdated := time.Since(m.IPMI.LastUpdated)

	if lastUpdated > 20*time.Minute {
		i.details = fmt.Sprintf("last updated %s ago", lastUpdated.String())
		return true
	}

	return false
}

func (*IssueBMCInfoOutdated) Spec() *spec {
	return &spec{
		Type:        TypeBMCInfoOutdated,
		Severity:    SeverityMajor,
		Description: "BMC has not been updated from either metal-hammer or metal-bmc",
		RefURL:      "https://docs.metal-stack.io/stable/installation/troubleshoot/#bmc-info-outdated",
	}
}
