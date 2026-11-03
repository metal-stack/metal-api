package issues

import "fmt"

type (
	Type string
)

func AllIssueTypes() []Type {
	return []Type{
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
		TypePowerSupplyFailure,
	}
}

func NewIssueFromType(t Type) (issue, error) {
	switch t {
	case TypeNoPartition:
		return &issueNoPartition{}, nil
	case TypeLivelinessDead:
		return &issueLivelinessDead{}, nil
	case TypeLivelinessUnknown:
		return &issueLivelinessUnknown{}, nil
	case TypeLivelinessNotAvailable:
		return &issueLivelinessNotAvailable{}, nil
	case TypeFailedMachineReclaim:
		return &issueFailedMachineReclaim{}, nil
	case TypeCrashLoop:
		return &issueCrashLoop{}, nil
	case TypeLastEventError:
		return &issueLastEventError{}, nil
	case TypeBMCWithoutMAC:
		return &issueBMCWithoutMAC{}, nil
	case TypeBMCWithoutIP:
		return &issueBMCWithoutIP{}, nil
	case TypeBMCInfoOutdated:
		return &issueBMCInfoOutdated{}, nil
	case TypeASNUniqueness:
		return &issueASNUniqueness{}, nil
	case TypeNonDistinctBMCIP:
		return &issueNonDistinctBMCIP{}, nil
	case TypeNoEventContainer:
		return &issueNoEventContainer{}, nil
	case TypePowerSupplyFailure:
		return &issuePowerSupplyFailure{}, nil
	default:
		return nil, fmt.Errorf("unknown issue type: %s", t)
	}
}
