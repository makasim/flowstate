package testcases

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func GetManySinceTime(t *testing.T, e flowstate.Engine, fr flowstate.FlowRegistry, d flowstate.Driver) {
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
	require.Greater(t, stateCtx.Committed.CommittedAt.UnixMilli(), int64(0))

	cmd := flowstate.GetStatesByLabels(map[string]string{
		`foo`: `fooVal`,
	}).WithSinceTime(time.UnixMilli(stateCtx.Committed.CommittedAt.UnixMilli() + 5000))
	require.NoError(t, e.Do(cmd))

	res := cmd.MustResult()
	require.Len(t, res.States, 0)
	require.False(t, res.More)

	cmd1 := flowstate.GetStatesByLabels(map[string]string{
		`foo`: `fooVal`,
	}).WithSinceTime(time.UnixMilli(stateCtx.Committed.CommittedAt.UnixMilli() - 1000))
	require.NoError(t, e.Do(cmd1))

	res1 := cmd1.MustResult()
	require.False(t, res.More)

	actStates := filterSystemStates(res1.States)
	require.Len(t, actStates, 1)

	require.Equal(t, flowstate.StateID(`aTID`), actStates[0].ID)
	require.NotEmpty(t, actStates[0].Rev)
	require.Equal(t, `paused`, actStates[0].Transition.Annotations[`flowstate.state`])
	require.Equal(t, `fooVal`, actStates[0].Labels[`foo`])
}
