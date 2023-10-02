package issues

import "fmt"

type (
	IssueType string
)

func AllIssueTypes() []IssueType {
	return []IssueType{
		IssueTypeNoPartition,
		IssueTypeLivelinessDead,
		IssueTypeLivelinessUnknown,
		IssueTypeLivelinessNotAvailable,
		IssueTypeFailedMachineReclaim,
		IssueTypeCrashLoop,
		IssueTypeLastEventError,
		IssueTypeBMCWithoutMAC,
		IssueTypeBMCWithoutIP,
		IssueTypeBMCInfoOutdated,
		IssueTypeASNUniqueness,
		IssueTypeNonDistinctBMCIP,
		IssueTypeNoEventContainer,
	}
}

func newIssueFromType(t IssueType) (issueImpl, error) {
	switch t {
	case IssueTypeNoPartition:
		return &IssueNoPartition{}, nil
	case IssueTypeLivelinessDead:
		return &IssueLivelinessDead{}, nil
	case IssueTypeLivelinessUnknown:
		return &IssueLivelinessUnknown{}, nil
	case IssueTypeLivelinessNotAvailable:
		return &IssueLivelinessNotAvailable{}, nil
	case IssueTypeFailedMachineReclaim:
		return &IssueFailedMachineReclaim{}, nil
	case IssueTypeCrashLoop:
		return &IssueCrashLoop{}, nil
	case IssueTypeLastEventError:
		return &IssueLastEventError{}, nil
	case IssueTypeBMCWithoutMAC:
		return &IssueBMCWithoutMAC{}, nil
	case IssueTypeBMCWithoutIP:
		return &IssueBMCWithoutIP{}, nil
	case IssueTypeBMCInfoOutdated:
		return &IssueBMCInfoOutdated{}, nil
	case IssueTypeASNUniqueness:
		return &IssueASNUniqueness{}, nil
	case IssueTypeNonDistinctBMCIP:
		return &IssueNonDistinctBMCIP{}, nil
	case IssueTypeNoEventContainer:
		return &IssueNoEventContainer{}, nil
	default:
		return nil, fmt.Errorf("unknown issue type: %s", t)
	}
}
