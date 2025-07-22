package testcases

import (
	"fmt"
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func Queue(t *testing.T, e flowstate.Engine, fr flowstate.FlowRegistry, d flowstate.Driver) {
	trkr := &Tracker{
		IncludeTaskID: true,
		IncludeState:  true,
	}

	mustSetFlow(fr, "queue", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		if flowstate.Resumed(stateCtx.Current) {
			return flowstate.Transit(stateCtx, `dequeued`), nil
		}

		stateCtx.Current.SetLabel("queue", "theName")

		return flowstate.Commit(
			flowstate.Pause(stateCtx),
		), nil
	}))
	mustSetFlow(fr, "enqueue", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		w := flowstate.NewWatcher(e, time.Millisecond*100, flowstate.GetStatesByLabels(map[string]string{
			"queue": "theName",
		}))
		defer w.Close()

		for {
			select {
			case queuedState := <-w.Next():
				queuedStateCtx := queuedState.CopyToCtx(&flowstate.StateCtx{})

				delete(queuedStateCtx.Current.Labels, "queue")

				if err := e.Do(
					flowstate.Commit(
						flowstate.Resume(queuedStateCtx),
						flowstate.End(stateCtx),
					),
					flowstate.Execute(queuedStateCtx),
				); err != nil {
					return nil, err
				}

				return flowstate.Noop(stateCtx), nil
			}
		}
	}))
	mustSetFlow(fr, "dequeued", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.Commit(
			flowstate.End(stateCtx),
		), nil
	}))

	for i := 0; i < 3; i++ {
		stateCtx := &flowstate.StateCtx{
			Current: flowstate.State{
				ID: flowstate.StateID(fmt.Sprintf("aTID%d", i)),
			},
		}

		err := e.Do(flowstate.Commit(
			flowstate.Transit(stateCtx, `queue`),
		))
		require.NoError(t, err)

		err = e.Execute(stateCtx)
		require.NoError(t, err)
	}

	enqueueStateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "enqueueTID",
		},
	}
	err := e.Do(flowstate.Commit(
		flowstate.Transit(enqueueStateCtx, `enqueue`),
	))
	require.NoError(t, err)

	err = e.Execute(enqueueStateCtx)
	require.NoError(t, err)

	trkr.WaitVisitedEqual(t, []string{
		"queue:aTID0",
		"queue:aTID1",
		"queue:aTID2",
		"enqueue:enqueueTID",
		"queue:resumed:aTID0",
		"dequeued:aTID0",
	}, time.Second)
}
