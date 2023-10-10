package issues

import (
	"fmt"
	"sort"
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
		n := n

		if n.ASN == 0 {
			continue
		}

		machineASNs[n.ASN] = nil
	}

	for _, machineFromAll := range c.Machines {
		machineFromAll := machineFromAll

		if machineFromAll.ID == m.ID {
			continue
		}
		otherMachine := machineFromAll

		if isNoFirewall(otherMachine) {
			continue
		}

		for _, n := range otherMachine.Allocation.MachineNetworks {
			n := n

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
		asn := asn
		asnList = append(asnList, asn)
	}
	sort.Slice(asnList, func(i, j int) bool {
		return asnList[i] < asnList[j]
	})

	for _, asn := range asnList {
		asn := asn

		overlappingMachines, ok := machineASNs[asn]
		if !ok || len(overlappingMachines) == 0 {
			continue
		}

		var sharedIDs []string
		for _, m := range overlappingMachines {
			m := m
			sharedIDs = append(sharedIDs, m.ID)
		}

		overlaps = append(overlaps, fmt.Sprintf("- ASN (%d) not unique, shared with %s", asn, sharedIDs))
	}

	if len(overlaps) == 0 {
		return false
	}

	sort.Slice(overlaps, func(i, j int) bool {
		return overlaps[i] < overlaps[j]
	})

	i.details = strings.Join(overlaps, "\n")

	return true
}

func (i *issueASNUniqueness) Details() string {
	return i.details
}
