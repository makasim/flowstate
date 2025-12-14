package testcases

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func WatchLabels(t *testing.T, e *flowstate.Engine, fr flowstate.FlowRegistry, d flowstate.Driver) {
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

	stateCtx.Current.SetLabel(`bar`, `barVal`)
	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Park(stateCtx),
	)))

	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Park(&flowstate.StateCtx{
			Current: flowstate.State{
				ID: "anotherTID",
				Labels: map[string]string{
					`foo`: `barVal`,
				},
			},
		}),
	)))

	w := flowstate.NewWatcher(e, time.Millisecond*100, flowstate.GetStatesByLabels(map[string]string{
		`foo`: `fooVal`,
	}))
	defer w.Close()

	actStates := watchCollectStates(t, w, 2)

	require.Len(t, actStates, 2)
	require.Equal(t, flowstate.StateID(`aTID`), actStates[0].ID)
	require.NotEmpty(t, actStates[0].Rev)
	require.Equal(t, `fooVal`, actStates[0].Labels[`foo`])

	require.Equal(t, flowstate.StateID(`aTID`), actStates[1].ID)
	require.NotEmpty(t, actStates[1].Rev)
	require.Equal(t, `fooVal`, actStates[1].Labels[`foo`])
	require.Equal(t, `barVal`, actStates[1].Labels[`bar`])

	require.Greater(t, actStates[1].Rev, actStates[0].Rev)
}

func watchCollectStates(t *testing.T, w *flowstate.Watcher, limit int) []flowstate.State {
	var states []flowstate.State

	timeoutT := time.NewTimer(time.Second)
	defer timeoutT.Stop()

loop:
	for {
		select {
		case s := <-w.Next():
			if len(filterSystemStates([]flowstate.State{s})) == 0 {
				continue
			}

			assert.Greater(t, s.CommittedAt.UnixMilli(), int64(0))
			// reset commited time to 0 to make test case deterministic
			s.CommittedAt = time.Time{}

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
