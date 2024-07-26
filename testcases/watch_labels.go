package testcases

import (
	"context"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func WatchLabels(t TestingT, d flowstate.Doer, _ flowRegistry) {
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

	stateCtx.Current.SetLabel(`bar`, `barVal`)
	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Pause(stateCtx),
	)))

	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Pause(&flowstate.StateCtx{
			Current: flowstate.State{
				ID: "anotherTID",
				Labels: map[string]string{
					`foo`: `barVal`,
				},
			},
		}),
	)))

	lis, err := flowstate.DoWatch(e, flowstate.Watch(map[string]string{
		`foo`: `fooVal`,
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
			ID:  "aTID",
			Rev: 2,
			Transition: flowstate.Transition{
				Annotations: map[string]string{
					`flowstate.state`: `paused`,
				},
			},
			Labels: map[string]string{
				`foo`: `fooVal`,
				`bar`: `barVal`,
			},
		},
	}, watchCollectStates(t, lis, 2))
}

func watchCollectStates(t TestingT, lis flowstate.WatchListener, limit int) []flowstate.State {
	var states []flowstate.State

	timeoutT := time.NewTimer(time.Second)
	defer timeoutT.Stop()

loop:
	for {
		select {
		case s := <-lis.Listen():
			assert.Greater(t, s.CommittedAtUnixMilli, int64(0))
			// reset commited time to 0 to make test case deterministic
			s.CommittedAtUnixMilli = 0

			states = append(states, s)
			if len(states) >= limit {
				break loop
			}
		case <-timeoutT.C:
			if limit == 0 {
				return states
			}

			t.Errorf("timeout watch collect states")
			return states
		}
	}

	return states
}
