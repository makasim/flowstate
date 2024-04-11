package tests

import (
	"testing"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func TestTwoConsequentNodesWithCommit(t *testing.T) {
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
		return flowstate.Commit(flowstate.Transit(taskCtx, `secondTID`)), nil
	}))
	br.SetBehavior("second", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(t, taskCtx)
		return flowstate.End(taskCtx), nil
	}))

	d := &nopDriver{}
	e := flowstate.NewEngine(d, br)

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
	require.Equal(t, 1, d.calls)
}
