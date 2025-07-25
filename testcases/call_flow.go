package testcases

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func CallFlow(t *testing.T, e flowstate.Engine, fr flowstate.FlowRegistry, d flowstate.Driver) {
	var nextStateCtx *flowstate.StateCtx
	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTID",
		},
	}

	endedCh := make(chan struct{})
	trkr := &Tracker{}

	mustSetFlow(fr, "call", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		if stateCtx.Current.Annotation(`state`) == `resumed` {
			return flowstate.Transit(stateCtx, `callEnd`), nil
		}

		nextStateCtx = &flowstate.StateCtx{
			Current: flowstate.State{
				ID: "aTID",
			},
		}

		if err := e.Do(
			flowstate.Park(stateCtx),
			flowstate.Stack(nextStateCtx, stateCtx, `caller_state`),
			flowstate.Transit(nextStateCtx, `called`),
			flowstate.Execute(nextStateCtx),
		); err != nil {
			return nil, err
		}

		return flowstate.Noop(), nil
	}))
	mustSetFlow(fr, "called", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.Transit(stateCtx, `calledEnd`), nil
	}))
	mustSetFlow(fr, "calledEnd", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		if stateCtx.Current.Annotations[`caller_state`] != "" {
			callStateCtx := &flowstate.StateCtx{}

			if err := e.Do(
				flowstate.Unstack(stateCtx, callStateCtx, `caller_state`),
				flowstate.Transit(callStateCtx, `call`).WithAnnotation(`state`, `resumed`),
				flowstate.Execute(callStateCtx),
				flowstate.Park(stateCtx),
			); err != nil {
				return nil, err
			}

			return flowstate.Noop(), nil
		}

		return flowstate.Park(stateCtx), nil
	}))

	mustSetFlow(fr, "callEnd", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		close(endedCh)

		return flowstate.Park(stateCtx), nil
	}))

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
