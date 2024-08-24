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

	require.Equal(t, []flowstate.State{
		{
			ID:  "aTID",
			Rev: 1,
			Transition: flowstate.Transition{
				Annotations: map[string]string{
					`flowstate.state`: `paused`,
				},
			},
			Labels: map[string]string{
				`foo`: `fooVal`,
			},
		},
	}, watchCollectStates(t, lis2, 1))
}
