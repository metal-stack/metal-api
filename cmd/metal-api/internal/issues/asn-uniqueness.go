package issues

import (
	"fmt"
	"slices"
	"strings"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

const (
	TypeASNUniqueness Type = "asn-not-unique"
)

type (
	issueASNUniqueness struct {
		details string
	}
)

func (i *issueASNUniqueness) Spec() *spec {
	return &spec{
		Type:        TypeASNUniqueness,
		Severity:    SeverityMinor,
		Description: "The ASN is not unique (only impact on firewalls)",
		RefURL:      "https://docs.metal-stack.io/stable/installation/troubleshoot/#asn-not-unique",
	}
}

func (i *issueASNUniqueness) Evaluate(m metal.Machine, ec metal.ProvisioningEventContainer, c *Config) bool {
	var (
		machineASNs  = map[uint32]metal.Machines{}
		overlaps     []string
		isNoFirewall = func(m metal.Machine) bool {
			return m.Allocation == nil || m.Allocation.Role != metal.RoleFirewall
		}
	)

	if isNoFirewall(m) {
		return false
	}

	for _, n := range m.Allocation.MachineNetworks {

		if n.ASN == 0 {
			continue
		}

		machineASNs[n.ASN] = nil
	}

	for _, machineFromAll := range c.Machines {

		if machineFromAll.ID == m.ID {
			continue
		}
		otherMachine := machineFromAll

		if isNoFirewall(otherMachine) {
			continue
		}

		for _, n := range otherMachine.Allocation.MachineNetworks {

			if n.ASN == 0 {
				continue
			}

			_, ok := machineASNs[n.ASN]
			if !ok {
				continue
			}

			machineASNs[n.ASN] = append(machineASNs[n.ASN], otherMachine)
		}
	}

	var asnList []uint32
	for asn := range machineASNs {
		asnList = append(asnList, asn)
	}
	slices.Sort(asnList)

	for _, asn := range asnList {

		overlappingMachines, ok := machineASNs[asn]
		if !ok || len(overlappingMachines) == 0 {
			continue
		}

		var sharedIDs []string
		for _, m := range overlappingMachines {
			sharedIDs = append(sharedIDs, m.ID)
		}

		overlaps = append(overlaps, fmt.Sprintf("- ASN (%d) not unique, shared with %s", asn, sharedIDs))
	}

	if len(overlaps) == 0 {
		return false
	}

	slices.Sort(overlaps)

	i.details = strings.Join(overlaps, "\n")

	return true
}

func (i *issueASNUniqueness) Details() string {
	return i.details
}
