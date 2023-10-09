package issues

import "fmt"

type (
	IssueType string
)

func AllIssueTypes() []IssueType {
	return []IssueType{
		TypeNoPartition,
		TypeLivelinessDead,
		TypeLivelinessUnknown,
		TypeLivelinessNotAvailable,
		TypeFailedMachineReclaim,
		TypeCrashLoop,
		TypeLastEventError,
		TypeBMCWithoutMAC,
		TypeBMCWithoutIP,
		TypeBMCInfoOutdated,
		TypeASNUniqueness,
		TypeNonDistinctBMCIP,
		TypeNoEventContainer,
	}
}

func NewIssueFromType(t IssueType) (issue, error) {
	switch t {
	case TypeNoPartition:
		return &IssueNoPartition{}, nil
	case TypeLivelinessDead:
		return &IssueLivelinessDead{}, nil
	case TypeLivelinessUnknown:
		return &IssueLivelinessUnknown{}, nil
	case TypeLivelinessNotAvailable:
		return &IssueLivelinessNotAvailable{}, nil
	case TypeFailedMachineReclaim:
		return &IssueFailedMachineReclaim{}, nil
	case TypeCrashLoop:
		return &IssueCrashLoop{}, nil
	case TypeLastEventError:
		return &IssueLastEventError{}, nil
	case TypeBMCWithoutMAC:
		return &IssueBMCWithoutMAC{}, nil
	case TypeBMCWithoutIP:
		return &IssueBMCWithoutIP{}, nil
	case TypeBMCInfoOutdated:
		return &IssueBMCInfoOutdated{}, nil
	case TypeASNUniqueness:
		return &IssueASNUniqueness{}, nil
	case TypeNonDistinctBMCIP:
		return &IssueNonDistinctBMCIP{}, nil
	case TypeNoEventContainer:
		return &IssueNoEventContainer{}, nil
	default:
		return nil, fmt.Errorf("unknown issue type: %s", t)
	}
}
