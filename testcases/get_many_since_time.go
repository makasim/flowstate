package testcases

import (
	"context"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func GetManySinceTime(t TestingT, d flowstate.Doer, _ FlowRegistry) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	l, _ := NewTestLogger(t)
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
	require.Greater(t, stateCtx.Committed.CommittedAtUnixMilli, int64(0))

	cmd := flowstate.GetStatesByLabels(map[string]string{
		`foo`: `fooVal`,
	}).WithSinceTime(time.UnixMilli(stateCtx.Committed.CommittedAtUnixMilli + 5000))
	require.NoError(t, e.Do(cmd))

	res, err := cmd.Result()
	require.NoError(t, err)

	require.Len(t, res.States, 0)
	require.False(t, res.More)

	cmd1 := flowstate.GetStatesByLabels(map[string]string{
		`foo`: `fooVal`,
	}).WithSinceTime(time.UnixMilli(stateCtx.Committed.CommittedAtUnixMilli - 1000))
	require.NoError(t, e.Do(cmd1))

	res1, err := cmd1.Result()
	require.NoError(t, err)

	require.Len(t, res1.States, 1)
	require.False(t, res.More)

	actStates := res1.States
	require.Equal(t, flowstate.StateID(`aTID`), actStates[0].ID)
	require.Equal(t, int64(1), actStates[0].Rev)
	require.Equal(t, `paused`, actStates[0].Transition.Annotations[`flowstate.state`])
	require.Equal(t, `fooVal`, actStates[0].Labels[`foo`])
}
