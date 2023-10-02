package issues

import "fmt"

const (
	// IssueSeverityMinor is an issue that should be checked from time to time but has no bad effects for the user.
	IssueSeverityMinor IssueSeverity = "minor"
	// IssueSeverityMajor is an issue where user experience is affected or provider resources are wasted.
	// overall functionality is still maintained though. major issues should be resolved as soon as possible.
	IssueSeverityMajor IssueSeverity = "major"
	// IssueSeverityCritical is an issue that can lead to disfunction of the system and need to be handled as quickly as possible.
	IssueSeverityCritical IssueSeverity = "critical"
)

type (
	IssueSeverity string
)

func AllSevereties() []IssueSeverity {
	return []IssueSeverity{
		IssueSeverityMinor,
		IssueSeverityMajor,
		IssueSeverityCritical,
	}
}

func SeverityFromString(input string) (IssueSeverity, error) {
	switch IssueSeverity(input) {
	case IssueSeverityCritical:
		return IssueSeverityCritical, nil
	case IssueSeverityMajor:
		return IssueSeverityMajor, nil
	case IssueSeverityMinor:
		return IssueSeverityMinor, nil
	default:
		return "", fmt.Errorf("unknown issue severity: %s", input)
	}
}

func (s IssueSeverity) LowerThan(o IssueSeverity) bool {
	smap := map[IssueSeverity]int{
		IssueSeverityCritical: 10,
		IssueSeverityMajor:    5,
		IssueSeverityMinor:    0,
	}

	return smap[s] < smap[o]
}
