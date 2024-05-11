package tests

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
)

func TestCallProcessWithCommit(t *testing.T) {
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

		if flowstate.Resumed(taskCtx) {
			return flowstate.Transit(taskCtx, `callEndTID`), nil
		}

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

		if err := taskCtx.Engine.Do(
			flowstate.Commit(
				flowstate.Pause(taskCtx, taskCtx.Current.Transition.ID),
				flowstate.Stack(taskCtx, nextTaskCtx),
				flowstate.Transit(nextTaskCtx, `calledTID`),
				flowstate.Execute(nextTaskCtx),
			),
		); err != nil {
			return nil, err
		}

		return flowstate.Nop(taskCtx), nil
	}))
	br.SetBehavior("called", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(taskCtx, trkr)

		if err := taskCtx.Engine.Do(
			flowstate.Transit(taskCtx, `calledEndTID`),
			flowstate.Execute(taskCtx),
		); err != nil {
			return nil, err
		}

		return flowstate.Nop(taskCtx), nil
	}))
	br.SetBehavior("end", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(taskCtx, trkr)
		if flowstate.Stacked(taskCtx) {
			callTaskCtx := &flowstate.TaskCtx{}

			if err := taskCtx.Engine.Do(
				flowstate.Commit(
					flowstate.Unstack(taskCtx, callTaskCtx),
					flowstate.Resume(callTaskCtx),
					flowstate.Execute(callTaskCtx),
					flowstate.End(taskCtx),
				),
			); err != nil {
				return nil, err
			}

			return flowstate.Nop(taskCtx), nil
		}

		return flowstate.End(taskCtx), nil
	}))

	d := &memdriver.Driver{}
	e := flowstate.NewEngine(d, br)

	err := e.Execute(taskCtx)
	require.NoError(t, err)

	time.Sleep(time.Second * 5)

	require.Equal(t, []flowstate.TransitionID{
		`callTID`,
		`calledTID`,
		`calledEndTID`,
		`callTID`,
		`callEndTID`,
	}, trkr.Visited())
}
