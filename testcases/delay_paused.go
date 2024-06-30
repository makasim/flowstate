package testcases

import (
	"context"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func Delay_Paused(t TestingT, d flowstate.Doer, fr flowRegistry) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	trkr := &Tracker{}

	fr.SetFlow("first", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		if flowstate.Delayed(stateCtx.Current) {
			return flowstate.Transit(stateCtx, `second`), nil
		}
		if flowstate.Paused(stateCtx.Current) {
			return flowstate.Noop(stateCtx), nil
		}

		if err := e.Do(flowstate.Commit(
			flowstate.Pause(stateCtx),
			flowstate.Delay(stateCtx, time.Millisecond*100),
		)); err != nil {
			return nil, err
		}
		return flowstate.Noop(stateCtx), nil
	}))
	fr.SetFlow("second", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.Commit(
			flowstate.End(stateCtx),
		), nil
	}))

	e, err := flowstate.NewEngine(d)
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

	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Transit(stateCtx, `first`)),
	))
	require.NoError(t, e.Execute(stateCtx))

	trkr.WaitVisitedEqual(t, []string{`first`, `first`, `second`}, time.Second*2)

}
