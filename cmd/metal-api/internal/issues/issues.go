package issues

import (
	"sort"
	"time"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
)

type (
	// Config contains configuration parameters for finding machine issues
	Config struct {
		// Machines are the machines to evaluate issues for
		Machines metal.Machines
		// EventContainers are the event containers of the machines to evaluate issues for
		// if not provided the machines will have a no-event-container issue
		EventContainers metal.ProvisioningEventContainers
		// Severity filters issues for the given severity
		Severity Severity
		// Only includes only the given issue types
		Only []Type
		// Omit omits the given issue types, this has precedence over only
		Omit []Type
		// LastErrorThreshold specifies for how long in the past the last event error is counted as an error
		LastErrorThreshold time.Duration
	}

	// Issue formulates an issue of a machine
	Issue struct {
		// Type specifies the issue type (id)
		Type Type
		// Severity specifies the severity of an issue
		Severity Severity
		// Description provides an issue description
		Description string
		// RefURL provides a link to a more detailed issue description in the metal-stack documentation
		RefURL string
		// Details may contain additional details on an evaluated issue
		Details string
	}

	// Issues is a list of issues
	Issues []Issue

	// MachineWithIssues summarizes a machine with issues
	MachineWithIssues struct {
		Machine *metal.Machine
		Issues  Issues
	}
	// MachineIssues is map of a machine response to a list of machine issues
	MachineIssues []*MachineWithIssues

	// MachineIssuesMap is a map of machine issues with the machine id as a map key
	MachineIssuesMap map[string]*MachineWithIssues

	issue interface {
		// Evaluate decides whether a given machine has the machine issue.
		// the third argument contains additional information that may be required for the issue evaluation
		Evaluate(m metal.Machine, ec metal.ProvisioningEventContainer, c *Config) bool
		// Spec returns the issue spec of this issue.
		Spec() *spec
		// Details returns additional information on the issue after the evaluation.
		Details() string
	}

	// spec defines the specification of an issue.
	spec struct {
		Type        Type
		Severity    Severity
		Description string
		RefURL      string
	}
)

func All() Issues {
	var res Issues

	for _, t := range AllIssueTypes() {
		i, err := NewIssueFromType(t)
		if err != nil {
			continue
		}

		res = append(res, toIssue(i))
	}

	return res
}

func toIssue(i issue) Issue {
	return Issue{
		Type:        i.Spec().Type,
		Severity:    i.Spec().Severity,
		Description: i.Spec().Description,
		RefURL:      i.Spec().RefURL,
		Details:     i.Details(),
	}
}

func Find(c *Config) (MachineIssuesMap, error) {
	if c.LastErrorThreshold == 0 {
		c.LastErrorThreshold = DefaultLastErrorThreshold()
	}

	res := MachineIssuesMap{}

	ecs := c.EventContainers.ByID()

	for _, m := range c.Machines {
		m := m

		ec, ok := ecs[m.ID]
		if !ok {
			res.add(m, toIssue(&issueNoEventContainer{}))
			continue
		}

		for _, t := range AllIssueTypes() {
			if !c.includeIssue(t) {
				continue
			}

			i, err := NewIssueFromType(t)
			if err != nil {
				return nil, err
			}

			if i.Evaluate(m, ec, c) {
				res.add(m, toIssue(i))
			}
		}
	}

	return res, nil
}

func (mis MachineIssues) Get(id string) *MachineWithIssues {
	for _, m := range mis {
		m := m

		if m.Machine == nil {
			continue
		}

		if m.Machine.ID == id {
			return m
		}
	}

	return nil
}

func (c *Config) includeIssue(t Type) bool {
	issue, err := NewIssueFromType(t)
	if err != nil {
		return false
	}

	if issue.Spec().Severity.LowerThan(c.Severity) {
		return false
	}

	for _, o := range c.Omit {
		if t == o {
			return false
		}
	}

	if len(c.Only) > 0 {
		for _, o := range c.Only {
			if t == o {
				return true
			}
		}
		return false
	}

	return true
}

func (mim MachineIssuesMap) add(m metal.Machine, issue Issue) {
	machineWithIssues, ok := mim[m.ID]
	if !ok {
		machineWithIssues = &MachineWithIssues{
			Machine: &m,
		}
	}
	machineWithIssues.Issues = append(machineWithIssues.Issues, issue)
	mim[m.ID] = machineWithIssues
}

func (mim MachineIssuesMap) ToList() MachineIssues {
	var res MachineIssues

	for _, machineWithIssues := range mim {
		res = append(res, &MachineWithIssues{
			Machine: machineWithIssues.Machine,
			Issues:  machineWithIssues.Issues,
		})
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].Machine.ID < res[j].Machine.ID
	})

	return res
}
