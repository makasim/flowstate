package testcases

import (
	"context"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func WatchSinceRev(t TestingT, d flowstate.Doer, _ FlowRegistry) {
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
	sinceRev := stateCtx.Committed.Rev

	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Pause(stateCtx),
	)))
	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Pause(stateCtx),
	)))
	lis, err := flowstate.DoWatch(e, flowstate.Watch(map[string]string{
		`foo`: `fooVal`,
	}).WithSinceRev(sinceRev))
	require.NoError(t, err)
	defer lis.Close()

	require.Equal(t, []flowstate.State{
		{
			ID:  "aTID",
			Rev: 2,
			Transition: flowstate.Transition{
				Annotations: map[string]string{
					`flowstate.state`: `paused`,
				},
			},
			Labels: map[string]string{
				`foo`: `fooVal`,
			},
		},
		{
			ID:  "aTID",
			Rev: 3,
			Transition: flowstate.Transition{
				Annotations: map[string]string{
					`flowstate.state`: `paused`,
				},
			},
			Labels: map[string]string{
				`foo`: `fooVal`,
			},
		},
	}, watchCollectStates(t, lis, 2))
}
