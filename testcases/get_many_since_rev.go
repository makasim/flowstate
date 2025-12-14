package testcases

import (
	"testing"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func GetManySinceRev(t *testing.T, e *flowstate.Engine, fr flowstate.FlowRegistry, d flowstate.Driver) {
	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTID",
			Labels: map[string]string{
				`foo`: `fooVal`,
			},
		},
	}
	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Park(stateCtx),
	)))
	sinceRev := stateCtx.Committed.Rev

	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Park(stateCtx),
	)))
	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Park(stateCtx),
	)))

	// filter all since zero revision
	cmd := flowstate.GetStatesByLabels(map[string]string{
		`foo`: `fooVal`,
	}).WithSinceRev(0)
	require.NoError(t, e.Do(cmd))
	require.Len(t, filterSystemStates(cmd.MustResult().States), 3)

	// filter since specific revision
	cmd = flowstate.GetStatesByLabels(map[string]string{
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
	require.Equal(t, `fooVal`, actStates[0].Labels[`foo`])

	require.Equal(t, flowstate.StateID(`aTID`), actStates[1].ID)
	require.Greater(t, actStates[1].Rev, sinceRev)
	require.Equal(t, `fooVal`, actStates[1].Labels[`foo`])

	require.Greater(t, actStates[1].Rev, actStates[0].Rev)

	// filter since the latest revision
	cmd = flowstate.GetStatesByLabels(map[string]string{
		`foo`: `fooVal`,
	}).WithSinceRev(actStates[1].Rev)
	require.NoError(t, e.Do(cmd))
	require.Len(t, filterSystemStates(cmd.MustResult().States), 0)
}
