package states

import (
	"context"

	"github.com/looplab/fsm"
)

type noopState struct{}

func (_ noopState) OnEnter(ctx context.Context, e *fsm.Event) {}
func (_ noopState) OnLeave(ctx context.Context, e *fsm.Event) {}
