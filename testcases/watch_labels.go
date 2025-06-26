package testcases

import (
	"context"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func WatchLabels(t TestingT, d flowstate.Driver, _ FlowRegistry) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	l, _ := NewTestLogger(t)
	e, err := flowstate.NewEngine(d, l)
	require.NoError(t, err)

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

	w := flowstate.NewWatcher(e, flowstate.GetStatesByLabels(map[string]string{
		`foo`: `fooVal`,
	}))
	defer w.Close()

	actStates := watchCollectStates(t, w, 2)

	require.Len(t, actStates, 2)
	require.Equal(t, flowstate.StateID(`aTID`), actStates[0].ID)
	require.Equal(t, int64(1), actStates[0].Rev)
	require.Equal(t, `paused`, actStates[0].Transition.Annotations[`flowstate.state`])
	require.Equal(t, `fooVal`, actStates[0].Labels[`foo`])

	require.Equal(t, flowstate.StateID(`aTID`), actStates[1].ID)
	require.Equal(t, int64(2), actStates[1].Rev)
	require.Equal(t, `paused`, actStates[1].Transition.Annotations[`flowstate.state`])
	require.Equal(t, `fooVal`, actStates[1].Labels[`foo`])
	require.Equal(t, `barVal`, actStates[1].Labels[`bar`])
}

func watchCollectStates(t TestingT, w *flowstate.Watcher, limit int) []flowstate.State {
	var states []flowstate.State

	timeoutT := time.NewTimer(time.Second)
	defer timeoutT.Stop()

loop:
	for {
		select {
		case s := <-w.Next():
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
