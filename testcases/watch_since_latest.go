package testcases

import (
	"context"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func WatchSinceLatest(t TestingT, d flowstate.Doer, _ FlowRegistry) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	l, _ := newTestLogger(t)
	e, err := flowstate.NewEngine(d, l)
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
	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Pause(stateCtx),
	)))
	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Pause(stateCtx),
	)))

	lis, err := flowstate.DoWatch(e, flowstate.Watch(map[string]string{
		`foo`: `fooVal`,
	}).WithSinceLatest())
	require.NoError(t, err)
	defer lis.Close()

	actStates := watchCollectStates(t, lis, 1)

	require.Len(t, actStates, 1)
	require.Equal(t, flowstate.StateID(`aTID`), actStates[0].ID)
	require.Equal(t, int64(3), actStates[0].Rev)
	require.Equal(t, `paused`, actStates[0].Transition.Annotations[`flowstate.state`])
	require.Equal(t, `fooVal`, actStates[0].Labels[`foo`])
}
