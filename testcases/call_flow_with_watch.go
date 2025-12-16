package testcases

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func CallFlowWithWatch(t *testing.T, e *flowstate.Engine, fr flowstate.FlowRegistry, d flowstate.Driver) {
	var nextStateCtx *flowstate.StateCtx
	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTID",
		},
	}

	endedCh := make(chan struct{})

	trkr := &Tracker{}

	mustSetFlow(fr, "call", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		if stateCtx.Current.Annotations[`called`] == `` {
			nextStateCtx = &flowstate.StateCtx{
				Current: flowstate.State{
					ID: "aNextTID",
				},
			}
			nextStateCtx.Committed = nextStateCtx.Current
			nextStateCtx.Current.SetLabel("theWatchLabel", string(stateCtx.Current.ID))

			stateCtx.Current.SetAnnotation("called", `true`)

			if err := e.Do(
				flowstate.Commit(
					flowstate.Transit(stateCtx, `call`),
					flowstate.Transit(nextStateCtx, `called`),
				),
				flowstate.Execute(nextStateCtx),
			); err != nil {
				return nil, err
			}
		}

		w := flowstate.NewWatcher(e, flowstate.GetStatesByLabels(map[string]string{
			`theWatchLabel`: string(stateCtx.Current.ID),
		}).WithSinceRev(stateCtx.Committed.Rev))
		defer w.Close()

		for {
			select {
			case <-stateCtx.Done():
				return flowstate.Noop(), nil
			case nextState := <-w.Next():
				nextStateCtx := nextState.CopyToCtx(&flowstate.StateCtx{})

				if !flowstate.Parked(nextStateCtx.Current) {
					continue
				}

				delete(stateCtx.Current.Annotations, `called`)

				return flowstate.Commit(
					flowstate.Transit(stateCtx, `callEnd`),
				), nil
			}
		}

	}))
	mustSetFlow(fr, "called", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.Transit(stateCtx, `calledEnd`), nil
	}))
	mustSetFlow(fr, "calledEnd", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		return flowstate.Commit(
			flowstate.Park(stateCtx),
		), nil
	}))
	mustSetFlow(fr, "callEnd", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		close(endedCh)

		return flowstate.Commit(
			flowstate.Park(stateCtx),
		), nil
	}))

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
		`callEnd`,
	}, trkr.Visited())
}
