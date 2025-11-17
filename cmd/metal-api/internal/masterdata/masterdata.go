package masterdata

import (
	"context"
	"fmt"

	v1 "github.com/metal-stack/masterdata-api/api/v1"
	mdm "github.com/metal-stack/masterdata-api/pkg/client"
	"github.com/metal-stack/metal-lib/pkg/healthstatus"
)

type MasterdataHealthClient struct {
	mdc mdm.Client
}

func NewMasterdataHealthClient(mdc mdm.Client) *MasterdataHealthClient {
	return &MasterdataHealthClient{mdc: mdc}
}

func (mhc *MasterdataHealthClient) ServiceName() string {
	return "masterdata-api"
}

func (mhc *MasterdataHealthClient) Check(ctx context.Context) (healthstatus.HealthResult, error) {
	version, err := mhc.mdc.Version().Get(ctx, &v1.GetVersionRequest{})

	if err != nil {
		return healthstatus.HealthResult{
			Status: healthstatus.HealthStatusUnhealthy,
		}, err
	}

	return healthstatus.HealthResult{
		Status:  healthstatus.HealthStatusHealthy,
		Message: fmt.Sprintf("connected to masterdata-api service version: %s rev: %s", version.Version, version.Revision),
	}, nil
}
