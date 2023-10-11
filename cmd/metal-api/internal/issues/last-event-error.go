package issues

import (
	"fmt"
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

const (
	TypeLastEventError Type = "last-event-error"
)

type (
	issueLastEventError struct {
		details string
	}
)

func DefaultLastErrorThreshold() time.Duration {
	return 7 * 24 * time.Hour
}

func (i *issueLastEventError) Spec() *spec {
	return &spec{
		Type:        TypeLastEventError,
		Severity:    SeverityMinor,
		Description: "the machine had an error during the provisioning lifecycle",
		RefURL:      "https://docs.metal-stack.io/stable/installation/troubleshoot/#last-event-error",
	}
}

func (i *issueLastEventError) Evaluate(m metal.Machine, ec metal.ProvisioningEventContainer, c *Config) bool {
	if c.LastErrorThreshold == 0 {
		return false
	}

	if ec.LastErrorEvent != nil {
		timeSince := time.Since(time.Time(ec.LastErrorEvent.Time))
		if timeSince < c.LastErrorThreshold {
			i.details = fmt.Sprintf("occurred %s ago", timeSince.String())
			return true
		}
	}

	return false
}

func (i *issueLastEventError) Details() string {
	return i.details
}
