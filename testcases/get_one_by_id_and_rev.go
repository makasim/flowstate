package testcases

import (
	"context"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func GetOneByIDAndRev(t TestingT, d flowstate.Driver, _ FlowRegistry) {
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

	stateCtx.Current.SetAnnotation("v", "1")
	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.CommitStateCtx(stateCtx),
	)))
	expectedStateCtx := stateCtx.CopyTo(&flowstate.StateCtx{})

	stateCtx.Current.SetAnnotation("v", "2")
	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.CommitStateCtx(stateCtx),
	)))

	stateCtx.Current.SetAnnotation("v", "3")
	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.CommitStateCtx(stateCtx),
	)))

	foundStateCtx := &flowstate.StateCtx{}

	require.NoError(t, e.Do(flowstate.GetStateByID(foundStateCtx, `aTID`, expectedStateCtx.Committed.Rev)))
	require.Equal(t, expectedStateCtx, foundStateCtx)
}
