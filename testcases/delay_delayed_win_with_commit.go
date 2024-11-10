package testcases

import (
	"context"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func Delay_DelayedWin_WithCommit(t TestingT, d flowstate.Doer, fr FlowRegistry) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	trkr := &Tracker{}

	fr.SetFlow("first", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		if flowstate.Delayed(stateCtx.Current) {
			return flowstate.Commit(
				flowstate.Transit(stateCtx, `second`),
			), nil
		}

		if err := e.Do(
			flowstate.Delay(stateCtx, time.Millisecond*200),
		); err != nil {
			return nil, err
		}

		time.Sleep(time.Millisecond * 400)

		return flowstate.Commit(
			flowstate.Transit(stateCtx, `third`),
		), nil
	}))
	fr.SetFlow("second", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.Commit(
			flowstate.End(stateCtx),
		), nil
	}))
	fr.SetFlow("third", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.Commit(
			flowstate.End(stateCtx),
		), nil
	}))

	l, _ := newTestLogger(t)
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
	stateCtx.Current.CopyTo(&stateCtx.Committed)

	require.NoError(t, e.Do(flowstate.Transit(stateCtx, `first`)))
	require.NoError(t, e.Execute(stateCtx))

	// no third in list
	trkr.WaitVisitedEqual(t, []string{`first`, `first`, `second`}, time.Millisecond*500)
}
