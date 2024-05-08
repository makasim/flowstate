package tests

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
)

func TestCallProcessWithWatch(t *testing.T) {
	p := flowstate.Process{
		ID:  "simplePID",
		Rev: 1,
		Nodes: []flowstate.Node{
			{
				ID:         "callNID",
				BehaviorID: "call",
			},
			{
				ID:         "calledNID",
				BehaviorID: "called",
			},
			{
				ID:         "endNID",
				BehaviorID: "end",
			},
		},
		Transitions: []flowstate.Transition{
			{
				ID:     "callTID",
				FromID: "",
				ToID:   "callNID",
			},
			{
				ID:     "calledTID",
				FromID: "",
				ToID:   "calledNID",
			},
			{
				ID:     "callEndTID",
				FromID: "callNID",
				ToID:   "endNID",
			},
			{
				ID:     "calledEndTID",
				FromID: "calledNID",
				ToID:   "endNID",
			},
		},
	}

	var nextTaskCtx *flowstate.TaskCtx
	taskCtx := &flowstate.TaskCtx{
		Current: flowstate.Task{
			ID:         "aTID",
			Rev:        0,
			ProcessID:  p.ID,
			ProcessRev: p.Rev,

			Transition: p.Transitions[0],
		},
		Process: p,
		Node:    p.Nodes[0],
	}
	taskCtx.Committed = taskCtx.Current

	trkr := &tracker{}

	br := &flowstate.MapBehaviorRegistry{}
	br.SetBehavior("call", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(taskCtx, trkr)

		if taskCtx.Current.Annotations[`called`] == `` {
			nextTaskCtx = &flowstate.TaskCtx{
				Current: flowstate.Task{
					ID:         "aNextTID",
					Rev:        0,
					ProcessID:  p.ID,
					ProcessRev: p.Rev,

					Transition: p.Transitions[1],
				},
				Process: p,
				Node:    p.Nodes[1],
			}
			nextTaskCtx.Committed = nextTaskCtx.Current
			nextTaskCtx.Current.SetLabel("theWatchLabel", string(taskCtx.Current.ID))

			taskCtx.Current.SetAnnotation("called", `true`)

			if err := taskCtx.Engine.Do(
				flowstate.Commit(
					flowstate.Transit(taskCtx, `callTID`),
					flowstate.Transit(nextTaskCtx, `calledTID`),
				),
			); err != nil {
				return nil, err
			}

			return flowstate.Nop(taskCtx), nil
		}

		cmd := flowstate.Watch(taskCtx.Committed.Rev, map[string]string{
			`theWatchLabel`: string(taskCtx.Current.ID),
		})
		if err := taskCtx.Engine.Do(cmd); err != nil {
			return nil, err
		}

		w := cmd.Watcher
		defer w.Close()

		for {
			select {
			// todo: case <-taskCtx.Done()
			case nextTaskCtx := <-w.Watch():
				if !flowstate.Ended(nextTaskCtx) {
					continue
				}

				delete(taskCtx.Current.Annotations, `called`)

				return flowstate.Commit(
					flowstate.Transit(taskCtx, `callEndTID`),
				), nil
			}
		}

	}))
	br.SetBehavior("called", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(taskCtx, trkr)
		return flowstate.Transit(taskCtx, `calledEndTID`), nil
	}))
	br.SetBehavior("end", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(taskCtx, trkr)

		return flowstate.Commit(
			flowstate.End(taskCtx),
		), nil
	}))

	d := &memdriver.Driver{}
	e := flowstate.NewEngine(d, br)

	err := e.Execute(taskCtx)
	require.NoError(t, err)

	time.Sleep(time.Second * 5)

	require.Equal(t, []flowstate.TransitionID{
		`callTID`,
		`callTID`,
		`calledTID`,
		`calledEndTID`,
		`callEndTID`,
	}, trkr.Visited())
}
