package states

import (
	"github.com/looplab/fsm"
)

type noopState struct{}

func (_ noopState) OnEnter(e *fsm.Event) {}
func (_ noopState) OnLeave(e *fsm.Event) {}
