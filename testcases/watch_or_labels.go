package testcases

import (
	"context"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func WatchORLabels(t TestingT, d flowstate.Doer, _ FlowRegistry) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	e, err := flowstate.NewEngine(d)
	defer func() {
		sCtx, sCtxCancel := context.WithTimeout(context.Background(), time.Second*5)
		defer sCtxCancel()

		require.NoError(t, e.Shutdown(sCtx))
	}()

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

	lis, err := flowstate.DoWatch(e, flowstate.Watch(map[string]string{
		`foo`: `fooVal`,
	}).WithORLabels(map[string]string{
		`bar`: `barVal`,
	}))
	require.NoError(t, err)
	defer lis.Close()

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
		{
			ID:  "anotherTID",
			Rev: 2,
			Transition: flowstate.Transition{
				Annotations: map[string]string{
					`flowstate.state`: `paused`,
				},
			},
			Labels: map[string]string{
				`bar`: `barVal`,
			},
		},
	}, watchCollectStates(t, lis, 2))
}
