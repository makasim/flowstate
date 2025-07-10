package testcases

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func CallFlowWithCommit(t *testing.T, e flowstate.Engine, d flowstate.Driver) {
	endedCh := make(chan struct{})
	trkr := &Tracker{}

	mustSetFlow(d, "call", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
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
	mustSetFlow(d, "called", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		if err := e.Do(
			flowstate.Transit(stateCtx, `calledEnd`),
			flowstate.Execute(stateCtx),
		); err != nil {
			return nil, err
		}

		return flowstate.Noop(stateCtx), nil
	}))
	mustSetFlow(d, "calledEnd", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
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
	mustSetFlow(d, "callEnd", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		close(endedCh)

		return flowstate.End(stateCtx), nil
	}))

	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTID",
		},
	}
	err := e.Do(flowstate.Commit(
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
