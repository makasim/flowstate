package testcases

import (
	"testing"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func GetOneByIDAndRev(t *testing.T, e flowstate.Engine, fr flowstate.FlowRegistry, d flowstate.Driver) {
	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTID",
		},
	}

	stateCtx.Current.SetAnnotation("v", "1")
	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Park(stateCtx),
	)))
	expectedStateCtx := stateCtx.CopyTo(&flowstate.StateCtx{})
	expectedStateCtx.Transitions = nil

	stateCtx.Current.SetAnnotation("v", "2")
	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Park(stateCtx),
	)))

	stateCtx.Current.SetAnnotation("v", "3")
	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Park(stateCtx),
	)))

	foundStateCtx := &flowstate.StateCtx{}

	require.NoError(t, e.Do(flowstate.GetStateByID(foundStateCtx, `aTID`, expectedStateCtx.Committed.Rev)))
	require.Equal(t, expectedStateCtx, foundStateCtx)
}
