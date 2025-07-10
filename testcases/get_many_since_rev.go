package testcases

import (
	"testing"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func GetManySinceRev(t *testing.T, e flowstate.Engine, d flowstate.Driver) {
	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTID",
			Labels: map[string]string{
				`foo`: `fooVal`,
			},
		},
	}
	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Pause(stateCtx),
	)))
	sinceRev := stateCtx.Committed.Rev

	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Pause(stateCtx),
	)))
	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Pause(stateCtx),
	)))

	cmd := flowstate.GetStatesByLabels(map[string]string{
		`foo`: `fooVal`,
	}).WithSinceRev(sinceRev)
	require.NoError(t, e.Do(cmd))

	res := cmd.MustResult()
	require.False(t, res.More)

	actStates := filterSystemStates(res.States)
	require.Len(t, actStates, 2)

	require.Len(t, actStates, 2)
	require.Equal(t, flowstate.StateID(`aTID`), actStates[0].ID)
	require.Greater(t, actStates[0].Rev, sinceRev)
	require.Equal(t, `paused`, actStates[0].Transition.Annotations[`flowstate.state`])
	require.Equal(t, `fooVal`, actStates[0].Labels[`foo`])

	require.Equal(t, flowstate.StateID(`aTID`), actStates[1].ID)
	require.Greater(t, actStates[1].Rev, sinceRev)
	require.Equal(t, `paused`, actStates[1].Transition.Annotations[`flowstate.state`])
	require.Equal(t, `fooVal`, actStates[1].Labels[`foo`])

	require.Greater(t, actStates[1].Rev, actStates[0].Rev)
}
