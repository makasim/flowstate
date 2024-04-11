package tests

import (
	"testing"

	"github.com/makasim/flowstate"
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

	br := &flowstate.MapBehaviorRegistry{}
	br.SetBehavior("first", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(t, taskCtx)

		cond0, _ := taskCtx.Data.Get("condition")
		cond := cond0.(bool)

		if cond {
			return flowstate.Transit(taskCtx, `secondTID`), nil
		} else {
			return flowstate.Transit(taskCtx, `thirdTID`), nil
		}
	}))
	br.SetBehavior("end", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(t, taskCtx)
		return flowstate.End(taskCtx), nil
	}))

	e := flowstate.NewEngine(br)

	// condition true
	taskCtx := &flowstate.TaskCtx{
		Task: flowstate.Task{
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
	visited, _ := taskCtx.Data.Get(`visited`)
	require.Equal(t, []interface{}{`firstTID`, `secondTID`}, visited)

	// condition false
	taskCtx = &flowstate.TaskCtx{
		Task: flowstate.Task{
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
	visited, _ = taskCtx.Data.Get(`visited`)
	require.Equal(t, []interface{}{`firstTID`, `thirdTID`}, visited)
}
