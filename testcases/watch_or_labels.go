package testcases

import (
	"testing"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func WatchORLabels(t *testing.T, e flowstate.Engine, fr flowstate.FlowRegistry, d flowstate.Driver) {
	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Pause(&flowstate.StateCtx{
			Current: flowstate.State{
				ID: "aTID",
				Labels: map[string]string{
					`foo`: `fooVal`,
				},
			},
		}),
	)))

	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Pause(&flowstate.StateCtx{
			Current: flowstate.State{
				ID: "anotherTID",
				Labels: map[string]string{
					`bar`: `barVal`,
				},
			},
		}),
	)))

	w := flowstate.NewWatcher(e, flowstate.GetStatesByLabels(map[string]string{
		`foo`: `fooVal`,
	}).WithORLabels(map[string]string{
		`bar`: `barVal`,
	}))
	defer w.Close()

	actStates := watchCollectStates(t, w, 2)

	require.Len(t, actStates, 2)
	require.Equal(t, flowstate.StateID(`aTID`), actStates[0].ID)
	require.NotEmpty(t, actStates[0].Rev)
	require.Equal(t, `paused`, actStates[0].Transition.Annotations[`flowstate.state`])
	require.Equal(t, `fooVal`, actStates[0].Labels[`foo`])
	require.Equal(t, ``, actStates[0].Labels[`bar`])

	require.Equal(t, flowstate.StateID(`anotherTID`), actStates[1].ID)
	require.NotEmpty(t, actStates[1].Rev)
	require.Equal(t, `paused`, actStates[1].Transition.Annotations[`flowstate.state`])
	require.Equal(t, ``, actStates[1].Labels[`foo`])
	require.Equal(t, `barVal`, actStates[1].Labels[`bar`])

	require.Greater(t, actStates[1].Rev, actStates[0].Rev)
}
