package testcases

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func WatchSinceTime(t *testing.T, e *flowstate.Engine, fr flowstate.FlowRegistry, d flowstate.Driver) {
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
	require.Greater(t, stateCtx.Committed.CommittedAt.UnixMilli(), int64(0))

	// time.Millisecond*100
	w := e.Watch(flowstate.GetStatesByLabels(map[string]string{
		`foo`: `fooVal`,
	}).WithSinceTime(time.UnixMilli(stateCtx.Committed.CommittedAt.UnixMilli() + 5000)))
	defer w.Close()

	require.Equal(t, []flowstate.State(nil), watchCollectStates(t, w, 0))

	// time.Millisecond*100
	w2 := e.Watch(flowstate.GetStatesByLabels(map[string]string{
		`foo`: `fooVal`,
	}).WithSinceTime(time.UnixMilli(stateCtx.Committed.CommittedAt.UnixMilli() - 1000)))
	defer w2.Close()

	actStates := watchCollectStates(t, w2, 1)
	require.Len(t, actStates, 1)

	require.Equal(t, flowstate.StateID(`aTID`), actStates[0].ID)
	require.NotEmpty(t, actStates[0].Rev)
	require.Equal(t, `fooVal`, actStates[0].Labels[`foo`])
}
