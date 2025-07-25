package testcases

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func WatchSinceLatest(t *testing.T, e flowstate.Engine, fr flowstate.FlowRegistry, d flowstate.Driver) {
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
	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Park(stateCtx),
	)))
	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Park(stateCtx),
	)))

	expRev := stateCtx.Committed.Rev

	w := flowstate.NewWatcher(e, time.Millisecond*100, flowstate.GetStatesByLabels(map[string]string{
		`foo`: `fooVal`,
	}).WithSinceLatest())
	defer w.Close()

	actStates := watchCollectStates(t, w, 1)

	require.Len(t, actStates, 1)
	require.Equal(t, flowstate.StateID(`aTID`), actStates[0].ID)
	require.Equal(t, expRev, actStates[0].Rev)
	require.Equal(t, `fooVal`, actStates[0].Labels[`foo`])
}
