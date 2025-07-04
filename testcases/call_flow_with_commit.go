package testcases

import (
	"context"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func CallFlowWithCommit(t TestingT, d flowstate.Driver, fr FlowRegistry) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	endedCh := make(chan struct{})
	trkr := &Tracker{}

	fr.SetFlow("call", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		if flowstate.Resumed(stateCtx.Current) {
			return flowstate.Transit(stateCtx, `callEnd`), nil
		}

		nextStateCtx := &flowstate.StateCtx{
			Current: flowstate.State{
				ID: "aNextTID",
			},
		}

		if err := e.Do(
			flowstate.Commit(
				flowstate.Pause(stateCtx),
				flowstate.Serialize(stateCtx, nextStateCtx, `caller_state`),
				flowstate.Transit(nextStateCtx, `called`),
			),
			flowstate.Execute(nextStateCtx),
		); err != nil {
			return nil, err
		}

		return flowstate.Noop(stateCtx), nil
	}))
	fr.SetFlow("called", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		if err := e.Do(
			flowstate.Transit(stateCtx, `calledEnd`),
			flowstate.Execute(stateCtx),
		); err != nil {
			return nil, err
		}

		return flowstate.Noop(stateCtx), nil
	}))
	fr.SetFlow("calledEnd", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		if stateCtx.Current.Annotations[`caller_state`] != "" {
			callStateCtx := &flowstate.StateCtx{}

			if err := e.Do(
				flowstate.Commit(
					flowstate.Deserialize(stateCtx, callStateCtx, `caller_state`),
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
	fr.SetFlow("callEnd", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		close(endedCh)

		return flowstate.End(stateCtx), nil
	}))

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
