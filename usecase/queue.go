package usecase

import (
	"fmt"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func Queue(t TestingT, d flowstate.Doer, fr flowRegistry) {
	trkr := &Tracker{
		IncludeTaskID: true,
		IncludeState:  true,
	}

	fr.SetFlow("queue", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		if flowstate.Resumed(stateCtx) {
			return flowstate.Transit(stateCtx, `dequeued`), nil
		}

		stateCtx.Current.SetLabel("queue", "theName")

		return flowstate.Commit(
			flowstate.Pause(stateCtx, stateCtx.Current.Transition.ToID),
		), nil
	}))
	fr.SetFlow("enqueue", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		w, err := e.Watch(0, map[string]string{
			"queue": "theName",
		})
		if err != nil {
			return nil, err
		}
		defer w.Close()

		for {
			select {
			case queuedStateCtx := <-w.Watch():
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
	fr.SetFlow("dequeued", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.Commit(
			flowstate.End(stateCtx),
		), nil
	}))

	e, err := flowstate.NewEngine(d)
	require.NoError(t, err)

	if d1, ok := d.(initer); ok {
		require.NoError(t, d1.Init(e))
	}

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
	err = e.Do(flowstate.Commit(
		flowstate.Transit(enqueueStateCtx, `enqueue`),
	))
	require.NoError(t, err)

	err = e.Execute(enqueueStateCtx)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 500)

	require.Equal(t, []string{
		"queue:aTID0",
		"queue:aTID1",
		"queue:aTID2",
		"enqueue:enqueueTID",
		"queue:resumed:aTID0",
		"dequeued:aTID0",
	}, trkr.Visited())
}
