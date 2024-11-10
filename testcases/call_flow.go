package testcases

import (
	"context"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func CallFlow(t TestingT, d flowstate.Doer, fr FlowRegistry) {
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

		if flowstate.Resumed(stateCtx.Current) {
			return flowstate.Transit(stateCtx, `callEnd`), nil
		}

		nextStateCtx = &flowstate.StateCtx{
			Current: flowstate.State{
				ID: "aTID",
			},
		}

		if err := e.Do(
			flowstate.Pause(stateCtx),
			flowstate.Serialize(stateCtx, nextStateCtx, `caller_state`),
			flowstate.Transit(nextStateCtx, `called`),
			flowstate.Execute(nextStateCtx),
		); err != nil {
			return nil, err
		}

		return flowstate.Noop(stateCtx), nil
	}))
	fr.SetFlow("called", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.Transit(stateCtx, `calledEnd`), nil
	}))
	fr.SetFlow("calledEnd", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		if stateCtx.Current.Annotations[`caller_state`] != "" {
			callStateCtx := &flowstate.StateCtx{}

			if err := e.Do(
				flowstate.Deserialize(stateCtx, callStateCtx, `caller_state`),
				flowstate.Resume(callStateCtx),
				flowstate.Execute(callStateCtx),
				flowstate.End(stateCtx),
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

	l, _ := newTestLogger(t)
	e, err := flowstate.NewEngine(d, l)
	require.NoError(t, err)
	defer func() {
		sCtx, sCtxCancel := context.WithTimeout(context.Background(), time.Second*5)
		defer sCtxCancel()

		require.NoError(t, e.Shutdown(sCtx))
	}()

	require.NoError(t, e.Do(flowstate.Transit(stateCtx, `call`)))
	require.NoError(t, e.Execute(stateCtx))

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
