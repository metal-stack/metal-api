package states

import (
	"github.com/looplab/fsm"
)

type noopState struct{}

func (_ noopState) OnTransition(e *fsm.Event) {}
func (_ noopState) OnLeave(e *fsm.Event)      {}
