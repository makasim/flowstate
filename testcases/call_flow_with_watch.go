package testcases

import (
	"context"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func CallFlowWithWatch(t TestingT, d flowstate.Driver, fr FlowRegistry) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	var nextStateCtx *flowstate.StateCtx
	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTID",
		},
	}

	endedCh := make(chan struct{})

	trkr := &Tracker{}

	fr.SetFlow("call", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
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
				return flowstate.Noop(stateCtx), nil
			case nextState := <-w.Next():
				nextStateCtx := nextState.CopyToCtx(&flowstate.StateCtx{})

				if !flowstate.Ended(nextStateCtx.Current) {
					continue
				}

				delete(stateCtx.Current.Annotations, `called`)

				return flowstate.Commit(
					flowstate.Transit(stateCtx, `callEnd`),
				), nil
			}
		}

	}))
	fr.SetFlow("called", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.Transit(stateCtx, `calledEnd`), nil
	}))
	fr.SetFlow("calledEnd", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		return flowstate.Commit(
			flowstate.End(stateCtx),
		), nil
	}))
	fr.SetFlow("callEnd", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		close(endedCh)

		return flowstate.Commit(
			flowstate.End(stateCtx),
		), nil
	}))

	l, _ := NewTestLogger(t)
	e, err := flowstate.NewEngine(d, l)
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
		`callEnd`,
	}, trkr.Visited())
}
