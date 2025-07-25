package testcases

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func WatchSinceRev(t *testing.T, e flowstate.Engine, fr flowstate.FlowRegistry, d flowstate.Driver) {
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

	w := flowstate.NewWatcher(e, time.Millisecond*100, flowstate.GetStatesByLabels(map[string]string{
		`foo`: `fooVal`,
	}).WithSinceRev(sinceRev))
	defer w.Close()

	actStates := watchCollectStates(t, w, 2)

	require.Len(t, actStates, 2)
	require.Equal(t, flowstate.StateID(`aTID`), actStates[0].ID)
	require.Greater(t, actStates[0].Rev, sinceRev)
	require.Equal(t, `fooVal`, actStates[0].Labels[`foo`])

	require.Equal(t, flowstate.StateID(`aTID`), actStates[1].ID)
	require.Greater(t, actStates[1].Rev, sinceRev)
	require.Equal(t, `fooVal`, actStates[1].Labels[`foo`])

	require.Greater(t, actStates[1].Rev, actStates[0].Rev)
}
