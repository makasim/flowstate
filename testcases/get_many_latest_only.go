package testcases

import (
	"testing"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func GetManyLatestOnly(t *testing.T, e flowstate.Engine, d flowstate.Driver) {
	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTID",
		},
	}
	stateCtx1 := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "anotherTID",
		},
	}

	for i := 0; i < 3; i++ {
		require.NoError(t, e.Do(flowstate.Commit(
			flowstate.Pause(stateCtx),
		)))
		require.NoError(t, e.Do(flowstate.Commit(
			flowstate.Pause(stateCtx1),
		)))
	}

	// guard
	require.NotEmpty(t, stateCtx.Committed.Rev)
	expRev0 := stateCtx.Committed.Rev

	// guard
	require.NotEmpty(t, stateCtx1.Committed.Rev)
	expRev1 := stateCtx1.Committed.Rev

	cmd := flowstate.GetStatesByLabels(nil).WithLatestOnly()
	require.NoError(t, e.Do(cmd))

	res := cmd.MustResult()
	require.False(t, res.More)

	actStates := filterSystemStates(res.States)
	require.Len(t, actStates, 2)

	require.Equal(t, flowstate.StateID(`aTID`), actStates[0].ID)
	require.Equal(t, expRev0, actStates[0].Rev)

	require.Equal(t, flowstate.StateID(`anotherTID`), actStates[1].ID)
	require.Equal(t, expRev1, actStates[1].Rev)

	require.Greater(t, actStates[1].Rev, actStates[0].Rev)
}
