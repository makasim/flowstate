package testcases

import (
	"context"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func WatchSinceTime(t TestingT, d flowstate.Doer, _ FlowRegistry) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	e, err := flowstate.NewEngine(d)
	defer func() {
		sCtx, sCtxCancel := context.WithTimeout(context.Background(), time.Second*5)
		defer sCtxCancel()

		require.NoError(t, e.Shutdown(sCtx))
	}()

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
	require.Greater(t, stateCtx.Committed.CommittedAtUnixMilli, int64(0))

	lis, err := flowstate.DoWatch(e, flowstate.Watch(map[string]string{
		`foo`: `fooVal`,
	}).WithSinceTime(time.UnixMilli(stateCtx.Committed.CommittedAtUnixMilli+5000)))
	require.NoError(t, err)
	defer lis.Close()

	require.Equal(t, []flowstate.State(nil), watchCollectStates(t, lis, 0))

	lis2, err := flowstate.DoWatch(e, flowstate.Watch(map[string]string{
		`foo`: `fooVal`,
	}).WithSinceTime(time.UnixMilli(stateCtx.Committed.CommittedAtUnixMilli-1000)))
	require.NoError(t, err)
	defer lis2.Close()

	actStates := watchCollectStates(t, lis2, 1)

	require.Len(t, actStates, 1)
	require.Equal(t, flowstate.StateID(`aTID`), actStates[0].ID)
	require.Equal(t, int64(1), actStates[0].Rev)
	require.Equal(t, `paused`, actStates[0].Transition.Annotations[`flowstate.state`])
	require.Equal(t, `fooVal`, actStates[0].Labels[`foo`])
}
