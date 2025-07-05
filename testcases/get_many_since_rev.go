package testcases

import (
	"context"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func GetManySinceRev(t TestingT, d flowstate.Driver) {
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
	sinceRev := stateCtx.Committed.Rev

	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Pause(stateCtx),
	)))
	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Pause(stateCtx),
	)))

	cmd := flowstate.GetStatesByLabels(map[string]string{
		`foo`: `fooVal`,
	}).WithSinceRev(sinceRev)
	require.NoError(t, e.Do(cmd))

	res := cmd.MustResult()
	require.Len(t, res.States, 2)
	require.False(t, res.More)

	actStates := res.States
	require.Len(t, actStates, 2)
	require.Equal(t, flowstate.StateID(`aTID`), actStates[0].ID)
	require.Equal(t, int64(2), actStates[0].Rev)
	require.Equal(t, `paused`, actStates[0].Transition.Annotations[`flowstate.state`])
	require.Equal(t, `fooVal`, actStates[0].Labels[`foo`])

	require.Equal(t, flowstate.StateID(`aTID`), actStates[1].ID)
	require.Equal(t, int64(3), actStates[1].Rev)
	require.Equal(t, `paused`, actStates[1].Transition.Annotations[`flowstate.state`])
	require.Equal(t, `fooVal`, actStates[1].Labels[`foo`])
}
