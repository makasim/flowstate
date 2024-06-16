package usecase

import (
	"context"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/exptcmd"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func CallProcessWithCommit(t TestingT, d flowstate.Doer, fr flowRegistry) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	var nextStateCtx *flowstate.StateCtx
	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTID",
		},
	}

	endedCh := make(chan struct{})
	trkr := &Tracker{}

	fr.SetFlow("call", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		if flowstate.Resumed(stateCtx) {
			return flowstate.Transit(stateCtx, `callEnd`), nil
		}

		nextStateCtx = &flowstate.StateCtx{
			Current: flowstate.State{
				ID: "aNextTID",
			},
		}

		if err := e.Do(
			flowstate.Commit(
				flowstate.Pause(stateCtx, stateCtx.Current.Transition.ToID),
				exptcmd.Stack(stateCtx, nextStateCtx),
				flowstate.Transit(nextStateCtx, `called`),
			),
			flowstate.Execute(nextStateCtx),
		); err != nil {
			return nil, err
		}

		return flowstate.Noop(stateCtx), nil
	}))
	fr.SetFlow("called", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		if err := e.Do(
			flowstate.Transit(stateCtx, `calledEnd`),
			flowstate.Execute(stateCtx),
		); err != nil {
			return nil, err
		}

		return flowstate.Noop(stateCtx), nil
	}))
	fr.SetFlow("calledEnd", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		if exptcmd.Stacked(stateCtx) {
			callStateCtx := &flowstate.StateCtx{}

			if err := e.Do(
				flowstate.Commit(
					exptcmd.Unstack(stateCtx, callStateCtx),
					flowstate.Resume(callStateCtx),
					flowstate.End(stateCtx),
				),
				flowstate.Execute(callStateCtx),
			); err != nil {
				return nil, err
			}

			return flowstate.Noop(stateCtx), nil
		}

		return flowstate.End(stateCtx), nil
	}))
	fr.SetFlow("callEnd", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		close(endedCh)

		return flowstate.End(stateCtx), nil
	}))

	e, err := flowstate.NewEngine(d)
	require.NoError(t, err)
	defer func() {
		sCtx, sCtxCancel := context.WithTimeout(context.Background(), time.Second*5)
		defer sCtxCancel()

		require.NoError(t, e.Shutdown(sCtx))
	}()

	err = e.Do(flowstate.Commit(
		flowstate.Transit(stateCtx, `call`),
	))
	require.NoError(t, err)

	err = e.Execute(stateCtx)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		select {
		case <-endedCh:
			return true
		default:
			return false
		}
	}, time.Second*5, time.Millisecond*50)

	require.Equal(t, []string{
		`call`,
		`called`,
		`calledEnd`,
		`call`,
		`callEnd`,
	}, trkr.Visited())
}
