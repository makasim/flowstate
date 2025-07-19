package testcases

import (
	"testing"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func GetManyLabels(t *testing.T, e flowstate.Engine, _ flowstate.FlowRegistry, _ flowstate.Driver) {
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

	stateCtx.Current.SetLabel(`bar`, `barVal`)
	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Pause(stateCtx),
	)))

	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Pause(&flowstate.StateCtx{
			Current: flowstate.State{
				ID: "anotherTID",
				Labels: map[string]string{
					`foo`: `barVal`,
				},
			},
		}),
	)))

	cmd := flowstate.GetStatesByLabels(map[string]string{
		`foo`: `fooVal`,
	})
	require.NoError(t, e.Do(cmd))

	res := cmd.MustResult()
	require.False(t, res.More)

	actStates := filterSystemStates(res.States)
	require.Len(t, actStates, 2)

	require.Equal(t, flowstate.StateID(`aTID`), actStates[0].ID)
	require.NotEmpty(t, actStates[0].Rev)
	require.Equal(t, `paused`, actStates[0].Transition.Annotations[`flowstate.state`])
	require.Equal(t, `fooVal`, actStates[0].Labels[`foo`])

	require.Equal(t, flowstate.StateID(`aTID`), actStates[1].ID)
	require.NotEmpty(t, actStates[1].Rev)
	require.Equal(t, `paused`, actStates[1].Transition.Annotations[`flowstate.state`])
	require.Equal(t, `fooVal`, actStates[1].Labels[`foo`])
	require.Equal(t, `barVal`, actStates[1].Labels[`bar`])

	require.Greater(t, actStates[1].Rev, actStates[0].Rev)
}
