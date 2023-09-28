package v1

import "google.golang.org/protobuf/types/known/timestamppb"

type (
	TenantGetHistoryRequest struct {
		At *timestamppb.Timestamp
	}
)
