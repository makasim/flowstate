package testcases

import (
	"fmt"
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func Queue(t *testing.T, e *flowstate.Engine, fr flowstate.FlowRegistry, d flowstate.Driver) {
	trkr := &Tracker{
		IncludeTaskID: true,
	}

	mustSetFlow(fr, "enqueue", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		stateCtx.Current.SetLabel("queue", "theName")

		return flowstate.Commit(
			flowstate.Park(stateCtx),
		), nil
	}))
	mustSetFlow(fr, "queue", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		w := flowstate.NewWatcher(e, flowstate.GetStatesByLabels(map[string]string{
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
						flowstate.Transit(queuedStateCtx, `dequeued`),
						flowstate.Park(stateCtx),
					),
					flowstate.Execute(queuedStateCtx),
				); err != nil {
					return nil, err
				}

				return flowstate.Noop(), nil
			}
		}
	}))
	mustSetFlow(fr, "dequeued", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.Commit(
			flowstate.Park(stateCtx),
		), nil
	}))

	for i := 0; i < 3; i++ {
		stateCtx := &flowstate.StateCtx{
			Current: flowstate.State{
				ID: flowstate.StateID(fmt.Sprintf("aTID%d", i)),
			},
		}

		err := e.Do(flowstate.Commit(
			flowstate.Transit(stateCtx, `enqueue`),
		))
		require.NoError(t, err)

		err = e.Execute(stateCtx)
		require.NoError(t, err)
	}

	enqueueStateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "workerTID",
		},
	}
	err := e.Do(flowstate.Commit(
		flowstate.Transit(enqueueStateCtx, `queue`),
	))
	require.NoError(t, err)

	err = e.Execute(enqueueStateCtx)
	require.NoError(t, err)

	trkr.WaitVisitedEqual(t, []string{
		"enqueue:aTID0",
		"enqueue:aTID1",
		"enqueue:aTID2",
		"queue:workerTID",
		"dequeued:aTID0",
	}, time.Second)
}
