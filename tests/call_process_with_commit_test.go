package tests

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
)

func TestCallProcessWithCommit(t *testing.T) {
	var nextStateCtx *flowstate.StateCtx
	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTID",
		},
	}

	endedCh := make(chan struct{})
	trkr := &tracker2{}

	br := &flowstate.MapFlowRegistry{}
	br.SetFlow("call", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		track2(stateCtx, trkr)

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
				flowstate.Stack(stateCtx, nextStateCtx),
				flowstate.Transit(nextStateCtx, `called`),
			),
			flowstate.Execute(nextStateCtx),
		); err != nil {
			return nil, err
		}

		return flowstate.Nop(stateCtx), nil
	}))
	br.SetFlow("called", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		track2(stateCtx, trkr)

		if err := e.Do(
			flowstate.Transit(stateCtx, `calledEnd`),
			flowstate.Execute(stateCtx),
		); err != nil {
			return nil, err
		}

		return flowstate.Nop(stateCtx), nil
	}))
	br.SetFlow("calledEnd", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		track2(stateCtx, trkr)
		if flowstate.Stacked(stateCtx) {
			callStateCtx := &flowstate.StateCtx{}

			if err := e.Do(
				flowstate.Commit(
					flowstate.Unstack(stateCtx, callStateCtx),
					flowstate.Resume(callStateCtx),
					flowstate.End(stateCtx),
				),
				flowstate.Execute(callStateCtx),
			); err != nil {
				return nil, err
			}

			return flowstate.Nop(stateCtx), nil
		}

		return flowstate.End(stateCtx), nil
	}))
	br.SetFlow("callEnd", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		track2(stateCtx, trkr)

		close(endedCh)

		return flowstate.End(stateCtx), nil
	}))

	d := &memdriver.Driver{}
	e := flowstate.NewEngine(d, br)

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
