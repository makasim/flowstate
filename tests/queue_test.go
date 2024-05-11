package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
)

func TestQueue(t *testing.T) {
	pQueue := flowstate.Process{
		ID:  "queuePID",
		Rev: 1,
		Nodes: []flowstate.Node{
			{
				ID:         "queueNID",
				BehaviorID: "queue",
			},
			{
				ID:         "dequeuedNID",
				BehaviorID: "dequeued",
			},
		},
		Transitions: []flowstate.Transition{
			{
				ID:   "queue",
				ToID: "queueNID",
			},
			{
				ID:     "dequeued",
				FromID: `queueNID`,
				ToID:   "dequeuedNID",
			},
		},
	}

	p := flowstate.Process{
		ID:  "enqueuePID",
		Rev: 2,
		Nodes: []flowstate.Node{
			{
				ID:         "enqueueNID",
				BehaviorID: "enqueue",
			},
		},
		Transitions: []flowstate.Transition{
			{
				ID:   "enqueue",
				ToID: `enqueueNID`,
			},
		},
	}

	trkr := &tracker{
		IncludeTaskID: true,
		IncludeState:  true,
	}

	br := &flowstate.MapBehaviorRegistry{}
	br.SetBehavior("queue", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(taskCtx, trkr)
		if flowstate.Resumed(taskCtx) {
			return flowstate.Transit(taskCtx, `dequeued`), nil
		}

		taskCtx.Current.SetLabel("queue", "theName")

		return flowstate.Commit(
			flowstate.Pause(taskCtx, taskCtx.Current.Transition.ID),
		), nil
	}))
	br.SetBehavior("enqueue", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(taskCtx, trkr)

		w, err := taskCtx.Engine.Watch(0, map[string]string{
			"queue": "theName",
		})
		if err != nil {
			return nil, err
		}
		defer w.Close()

		for {
			select {
			case queuedTaskCtx := <-w.Watch():
				delete(queuedTaskCtx.Current.Labels, "queue")

				return flowstate.Commit(
					flowstate.Resume(queuedTaskCtx),
					flowstate.End(taskCtx),
				), nil
			}
		}
	}))
	br.SetBehavior("dequeued", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(taskCtx, trkr)
		return flowstate.Commit(
			flowstate.End(taskCtx),
		), nil
	}))

	d := &memdriver.Driver{}
	e := flowstate.NewEngine(d, br)

	for i := 0; i < 3; i++ {
		taskCtx := &flowstate.TaskCtx{
			Current: flowstate.Task{
				ID:         flowstate.TaskID(fmt.Sprintf("aTID%d", i)),
				Rev:        0,
				ProcessID:  pQueue.ID,
				ProcessRev: pQueue.Rev,
			},
			Process: pQueue,
		}

		err := e.Do(flowstate.Commit(
			flowstate.Transit(taskCtx, `queue`),
		))
		require.NoError(t, err)

		err = e.Execute(taskCtx)
		require.NoError(t, err)
	}

	enqueueTaskCtx := &flowstate.TaskCtx{
		Current: flowstate.Task{
			ID:         "enqueueTID",
			ProcessID:  p.ID,
			ProcessRev: p.Rev,
		},
		Process: p,
	}
	err := e.Do(flowstate.Commit(
		flowstate.Transit(enqueueTaskCtx, `enqueue`),
	))
	require.NoError(t, err)

	err = e.Execute(enqueueTaskCtx)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 500)

	require.Equal(t, []flowstate.TransitionID{
		"queue:aTID0",
		"queue:aTID1",
		"queue:aTID2",
		"enqueue:enqueueTID",
		"queue:resumed:aTID0",
		"dequeued:aTID0",
	}, trkr.Visited())
}
