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
		flowstate.Park(stateCtx),
	)))

	stateCtx.Current.SetLabel(`bar`, `barVal`)
	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Park(stateCtx),
	)))

	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Park(&flowstate.StateCtx{
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
	require.Equal(t, `fooVal`, actStates[0].Labels[`foo`])

	require.Equal(t, flowstate.StateID(`aTID`), actStates[1].ID)
	require.NotEmpty(t, actStates[1].Rev)
	require.Equal(t, `fooVal`, actStates[1].Labels[`foo`])
	require.Equal(t, `barVal`, actStates[1].Labels[`bar`])

	require.Greater(t, actStates[1].Rev, actStates[0].Rev)
}
