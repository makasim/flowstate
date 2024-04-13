package tests

import (
	"testing"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
)

func TestCondition(t *testing.T) {
	p := flowstate.Process{
		ID:  "simplePID",
		Rev: 1,
		Nodes: []flowstate.Node{
			{
				ID:         "firstNID",
				BehaviorID: "first",
			},
			{
				ID:         "secondNID",
				BehaviorID: "end",
			},
			{
				ID:         "thirdNID",
				BehaviorID: "end",
			},
		},
		Transitions: []flowstate.Transition{
			{
				ID:     "firstTID",
				FromID: "",
				ToID:   "firstNID",
			},
			{
				ID:     "secondTID",
				FromID: "firstNID",
				ToID:   "secondNID",
			},
			{
				ID:     "thirdTID",
				FromID: "firstNID",
				ToID:   "thirdNID",
			},
		},
	}

	trkr := &tracker{}

	br := &flowstate.MapBehaviorRegistry{}
	br.SetBehavior("first", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(taskCtx, trkr)

		cond0, _ := taskCtx.Data.Get("condition")
		cond := cond0.(bool)

		if cond {
			return flowstate.Transit(taskCtx, `secondTID`), nil
		} else {
			return flowstate.Transit(taskCtx, `thirdTID`), nil
		}
	}))
	br.SetBehavior("end", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(taskCtx, trkr)
		return flowstate.End(taskCtx), nil
	}))

	d := &memdriver.Driver{}
	e := flowstate.NewEngine(d, br)

	// condition true
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
		Data: flowstate.Data{
			Bytes: []byte(`{"condition": true}`),
		},
	}

	err := e.Execute(taskCtx)
	require.NoError(t, err)
	require.Equal(t, []flowstate.TransitionID{`firstTID`, `secondTID`}, trkr.visited)

	// condition false
	trkr.visited = nil

	taskCtx = &flowstate.TaskCtx{
		Current: flowstate.Task{
			ID:         "aTID",
			Rev:        0,
			ProcessID:  p.ID,
			ProcessRev: p.Rev,

			Transition: p.Transitions[0],
		},
		Process: p,
		Node:    p.Nodes[0],
		Data: flowstate.Data{
			Bytes: []byte(`{"condition": false}`),
		},
	}

	err = e.Execute(taskCtx)
	require.NoError(t, err)
	require.Equal(t, []flowstate.TransitionID{`firstTID`, `thirdTID`}, trkr.visited)
}
