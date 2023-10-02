package issues

import (
	"fmt"
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

const (
	IssueTypeLastEventError IssueType = "last-event-error"
)

type (
	IssueLastEventError struct {
		details string
	}
)

func DefaultLastErrorThreshold() time.Duration {
	return 7 * 24 * time.Hour
}

func (i *IssueLastEventError) Spec() *issueSpec {
	return &issueSpec{
		Type:        IssueTypeLastEventError,
		Severity:    IssueSeverityMinor,
		Description: "the machine had an error during the provisioning lifecycle",
		RefURL:      "https://docs.metal-stack.io/stable/installation/troubleshoot/#last-event-error",
	}
}

func (i *IssueLastEventError) Evaluate(m metal.Machine, ec metal.ProvisioningEventContainer, c *IssueConfig) bool {
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

func (i *IssueLastEventError) Details() string {
	return i.details
}
