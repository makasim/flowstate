package flowstate_test

import (
	"testing"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func TestSingleNode(t *testing.T) {
	p := flowstate.Process{
		ID:  "simplePID",
		Rev: 1,
		Nodes: []flowstate.Node{
			{
				ID:         "firstNID",
				BehaviorID: "first",
			},
		},
		Transitions: []flowstate.Transition{
			{
				ID:     "firstTID",
				FromID: "",
				ToID:   "firstNID",
			},
		},
	}

	br := &flowstate.MapBehaviorRegistry{}
	br.SetBehavior("first", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(t, taskCtx)
		return flowstate.End(taskCtx), nil
	}))

	e := flowstate.NewEngine(br)

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
	}

	err := e.Execute(taskCtx)

	require.NoError(t, err)

	visited, _ := taskCtx.Data.Get("visited")
	require.Equal(t, []interface{}{`firstTID`}, visited)
}

func TestTwoConsequentNodes(t *testing.T) {
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
				BehaviorID: "second",
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
		},
	}

	br := &flowstate.MapBehaviorRegistry{}
	br.SetBehavior("first", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(t, taskCtx)
		return flowstate.Transit(taskCtx, `secondTID`), nil
	}))
	br.SetBehavior("second", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(t, taskCtx)
		return flowstate.End(taskCtx), nil
	}))

	e := flowstate.NewEngine(br)

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
	}

	err := e.Execute(taskCtx)

	require.NoError(t, err)

	visited, _ := taskCtx.Data.Get(`visited`)
	require.Equal(t, []interface{}{`firstTID`, `secondTID`}, visited)
}

func track(t *testing.T, taskCtx *flowstate.TaskCtx) {
	var visited []interface{}
	visited0, found := taskCtx.Data.Get("visited")
	if found {
		visited = visited0.([]interface{})
	}
	visited = append(visited, string(taskCtx.Transition.ID))

	err := taskCtx.Data.Set("visited", visited)
	require.NoError(t, err)
}
