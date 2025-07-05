package testcases

import (
	"context"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func GetManyLatestOnly(t TestingT, d flowstate.Driver) {
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
		},
	}
	stateCtx1 := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "anotherTID",
		},
	}

	for i := 0; i < 3; i++ {
		require.NoError(t, e.Do(flowstate.Commit(
			flowstate.Pause(stateCtx),
		)))
		require.NoError(t, e.Do(flowstate.Commit(
			flowstate.Pause(stateCtx1),
		)))
	}

	cmd := flowstate.GetStatesByLabels(nil).WithLatestOnly()
	require.NoError(t, e.Do(cmd))

	res := cmd.MustResult()
	require.Len(t, res.States, 2)
	require.False(t, res.More)

	actStates := res.States
	require.Equal(t, flowstate.StateID(`aTID`), actStates[0].ID)
	require.Equal(t, int64(5), actStates[0].Rev)

	require.Equal(t, flowstate.StateID(`anotherTID`), actStates[1].ID)
	require.Equal(t, int64(6), actStates[1].Rev)
}
