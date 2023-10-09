package issues

import (
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-lib/pkg/pointer"
)

const (
	TypeFailedMachineReclaim Type = "failed-machine-reclaim"
)

type (
	IssueFailedMachineReclaim struct{}
)

func (i *IssueFailedMachineReclaim) Spec() *spec {
	return &spec{
		Type:        TypeFailedMachineReclaim,
		Severity:    SeverityCritical,
		Description: "machine phones home but not allocated",
		RefURL:      "https://docs.metal-stack.io/stable/installation/troubleshoot/#failed-machine-reclaim",
	}
}

func (i *IssueFailedMachineReclaim) Evaluate(m metal.Machine, ec metal.ProvisioningEventContainer, c *Config) bool {
	if ec.FailedMachineReclaim {
		return true
	}

	// compatibility: before the provisioning FSM was renewed, this state could be detected the following way
	// we should keep this condition
	if m.Allocation == nil && pointer.FirstOrZero(ec.Events).Event == metal.ProvisioningEventPhonedHome {
		return true
	}

	return false
}

func (i *IssueFailedMachineReclaim) Details() string {
	return ""
}
