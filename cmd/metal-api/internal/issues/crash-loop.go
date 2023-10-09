package issues

import (
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-lib/pkg/pointer"
)

const (
	IssueTypeCrashLoop IssueType = "crashloop"
)

type (
	IssueCrashLoop struct{}
)

func (i *IssueCrashLoop) Spec() *spec {
	return &spec{
		Type:        IssueTypeCrashLoop,
		Severity:    IssueSeverityMajor,
		Description: "machine is in a provisioning crash loop (⭕)",
		RefURL:      "https://docs.metal-stack.io/stable/installation/troubleshoot/#crashloop",
	}
}

func (i *IssueCrashLoop) Evaluate(m metal.Machine, ec metal.ProvisioningEventContainer, c *IssueConfig) bool {
	if ec.CrashLoop {
		if pointer.FirstOrZero(ec.Events).Event == metal.ProvisioningEventWaiting {
			// Machine which are waiting are not considered to have issues
		} else {
			return true
		}
	}
	return false
}

func (i *IssueCrashLoop) Details() string {
	return ""
}
