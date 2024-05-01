package tests

import (
	"testing"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
)

func TestThreeConsequentNodes(t *testing.T) {
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
			{
				ID:         "thirdNID",
				BehaviorID: "third",
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
				FromID: "secondNID",
				ToID:   "thirdNID",
			},
		},
	}

	trkr := &tracker{}

	br := &flowstate.MapBehaviorRegistry{}
	br.SetBehavior("first", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(taskCtx, trkr)
		return flowstate.Transit(taskCtx, `secondTID`), nil
	}))
	br.SetBehavior("second", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(taskCtx, trkr)
		return flowstate.Transit(taskCtx, `thirdTID`), nil
	}))
	br.SetBehavior("third", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(taskCtx, trkr)
		return flowstate.End(taskCtx), nil
	}))

	d := &memdriver.Driver{}
	e := flowstate.NewEngine(d, br)

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

	err := e.Execute(taskCtx)

	require.NoError(t, err)

	require.Equal(t, []flowstate.TransitionID{`firstTID`, `secondTID`, `thirdTID`}, trkr.Visited())
}
