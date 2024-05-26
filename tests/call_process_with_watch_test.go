package tests

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
)

func TestCallProcessWithWatch(t *testing.T) {
	var nextStateCtx *flowstate.StateCtx
	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTID",
		},
	}

	endedCh := make(chan struct{})

	trkr := &tracker2{}

	br := &flowstate.MapFlowRegistry{}
	br.SetFlow("call", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx) (flowstate.Command, error) {
		track2(stateCtx, trkr)

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
					flowstate.Execute(nextStateCtx),
				),
			); err != nil {
				return nil, err
			}
		}

		w, err := e.Watch(stateCtx.Committed.Rev, map[string]string{
			`theWatchLabel`: string(stateCtx.Current.ID),
		})
		if err != nil {
			return nil, err
		}
		defer w.Close()

		for {
			select {
			// todo: case <-stateCtx.Done()
			case nextStateCtx := <-w.Watch():
				if !flowstate.Ended(nextStateCtx) {
					continue
				}

				delete(stateCtx.Current.Annotations, `called`)

				return flowstate.Commit(
					flowstate.Transit(stateCtx, `callEnd`),
				), nil
			}
		}

	}))
	br.SetFlow("called", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx) (flowstate.Command, error) {
		track2(stateCtx, trkr)
		return flowstate.Transit(stateCtx, `calledEnd`), nil
	}))
	br.SetFlow("calledEnd", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx) (flowstate.Command, error) {
		track2(stateCtx, trkr)

		return flowstate.Commit(
			flowstate.End(stateCtx),
		), nil
	}))
	br.SetFlow("callEnd", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx) (flowstate.Command, error) {
		track2(stateCtx, trkr)

		close(endedCh)

		return flowstate.Commit(
			flowstate.End(stateCtx),
		), nil
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
		`callEnd`,
	}, trkr.Visited())
}
