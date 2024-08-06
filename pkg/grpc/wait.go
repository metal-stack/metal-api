package helper

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// WaitForAllocation can be used to call the wait method continuously until an allocation was made.
// This is made for the metal-hammer and located here for better testability.
func WaitForAllocation(ctx context.Context, log *slog.Logger, service v1.BootServiceClient, machineID string, timeout time.Duration) error {
	req := &v1.BootServiceWaitRequest{
		MachineId: machineID,
	}

	for {
		stream, err := service.Wait(ctx, req)
		if err != nil {
			log.Error("failed waiting for allocation", "retry after", timeout, "error", err)

			time.Sleep(timeout)
			continue
		}

		for {
			_, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				log.Info("machine has been requested for allocation", "machineID", machineID)
				return nil
			}

			if err != nil {
				if e, ok := status.FromError(err); ok {
					log.Error("got error from wait call", "code", e.Code(), "message", e.Message(), "details", e.Details())
					switch e.Code() { // nolint:exhaustive
					case codes.Unimplemented:
						return fmt.Errorf("metal-api breaking change detected, rebooting: %w", err)
					}
				}

				log.Error("failed stream receiving during waiting for allocation", "retry after", timeout, "error", err)

				time.Sleep(timeout)
				break
			}

			log.Info("wait for allocation...")
		}
	}
}
