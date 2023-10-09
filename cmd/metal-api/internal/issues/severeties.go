package issues

import "fmt"

const (
	// SeverityMinor is an issue that should be checked from time to time but has no bad effects for the user.
	SeverityMinor Severity = "minor"
	// SeverityMajor is an issue where user experience is affected or provider resources are wasted.
	// overall functionality is still maintained though. major issues should be resolved as soon as possible.
	SeverityMajor Severity = "major"
	// SeverityCritical is an issue that can lead to disfunction of the system and need to be handled as quickly as possible.
	SeverityCritical Severity = "critical"
)

type (
	Severity string
)

func AllSevereties() []Severity {
	return []Severity{
		SeverityMinor,
		SeverityMajor,
		SeverityCritical,
	}
}

func SeverityFromString(input string) (Severity, error) {
	switch Severity(input) {
	case SeverityCritical:
		return SeverityCritical, nil
	case SeverityMajor:
		return SeverityMajor, nil
	case SeverityMinor:
		return SeverityMinor, nil
	default:
		return "", fmt.Errorf("unknown issue severity: %s", input)
	}
}

func (s Severity) LowerThan(o Severity) bool {
	smap := map[Severity]int{
		SeverityCritical: 10,
		SeverityMajor:    5,
		SeverityMinor:    0,
	}

	return smap[s] < smap[o]
}
