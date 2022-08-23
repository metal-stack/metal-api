package fsm

import (
	"testing"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/fsm/states"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventsProperlyDefined(t *testing.T) {
	events := Events()
	allStates := states.AllStates(&states.StateConfig{})
	allStates[SelfTransitionState] = nil

	for _, e := range events {
		require.NotEmpty(t, e.Dst)
		assert.Contains(t, allStates, e.Dst)
		assert.NotEmpty(t, e.Src)
		assert.NotEmpty(t, e.Name)
	}
}
