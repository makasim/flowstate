package tests

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func TestDefer(t *testing.T) {
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

	trkr := &tracker{}

	br := &flowstate.MapBehaviorRegistry{}
	br.SetBehavior("first", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(taskCtx, trkr)
		if taskCtx.Current.Transition.Annotations[flowstate.DeferAtAnnotation] != `` {
			return flowstate.Transit(taskCtx, `secondTID`), nil
		}

		return flowstate.Defer(taskCtx, time.Millisecond*200), nil
	}))
	br.SetBehavior("end", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(taskCtx, trkr)
		return flowstate.End(taskCtx), nil
	}))

	d := &nopDriver{}
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

	time.Sleep(time.Millisecond * 500)

	require.Equal(t, []flowstate.TransitionID{`firstTID`, `firstTID`, `secondTID`}, trkr.visited)
}
