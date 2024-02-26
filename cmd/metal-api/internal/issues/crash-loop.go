package issues

import (
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-lib/pkg/pointer"
)

const (
	TypeCrashLoop Type = "crashloop"
)

type (
	issueCrashLoop struct{}
)

func (i *issueCrashLoop) Spec() *spec {
	return &spec{
		Type:        TypeCrashLoop,
		Severity:    SeverityMajor,
		Description: "machine is in a provisioning crash loop (⭕)",
		RefURL:      "https://docs.metal-stack.io/stable/installation/troubleshoot/#crashloop",
	}
}

func (i *issueCrashLoop) Evaluate(m metal.Machine, ec metal.ProvisioningEventContainer, c *Config) bool {
	if ec.CrashLoop {
		if pointer.FirstOrZero(ec.Events).Event == metal.ProvisioningEventWaiting {
			// Machine which are waiting are not considered to have issues
		} else {
			return true
		}
	}
	return false
}

func (i *issueCrashLoop) Details() string {
	return ""
}
