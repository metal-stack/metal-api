package issues

import (
	"fmt"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

const (
	TypeNonDistinctBMCIP Type = "bmc-no-distinct-ip"
)

type (
	issueNonDistinctBMCIP struct {
		details string
	}
)

func (i *issueNonDistinctBMCIP) Spec() *spec {
	return &spec{
		Type:        TypeNonDistinctBMCIP,
		Description: "BMC IP address is not distinct",
		RefURL:      "https://docs.metal-stack.io/stable/installation/troubleshoot/#bmc-no-distinct-ip",
	}
}

func (i *issueNonDistinctBMCIP) Evaluate(m metal.Machine, ec metal.ProvisioningEventContainer, c *Config) bool {
	if m.IPMI.Address == "" {
		return false
	}

	var (
		bmcIP    = m.IPMI.Address
		overlaps []string
	)

	for _, machineFromAll := range c.Machines {
		machineFromAll := machineFromAll

		if machineFromAll.ID == m.ID {
			continue
		}
		otherMachine := machineFromAll

		if otherMachine.IPMI.Address == "" {
			continue
		}

		if bmcIP == otherMachine.IPMI.Address {
			overlaps = append(overlaps, otherMachine.ID)
		}
	}

	if len(overlaps) == 0 {
		return false
	}

	i.details = fmt.Sprintf("BMC IP (%s) not unique, shared with %s", bmcIP, overlaps)

	return true
}

func (i *issueNonDistinctBMCIP) Details() string {
	return i.details
}
