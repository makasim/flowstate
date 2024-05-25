package tests

import (
	"testing"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
)

func TestTwoConsequentNodes(t *testing.T) {
	//p := flowstate.Process{
	//	ID:  "simplePID",
	//	Rev: 1,
	//	Nodes: []flowstate.Node{
	//		{
	//			ID:         "firstNID",
	//			BehaviorID: "first",
	//		},
	//		{
	//			ID:         "secondNID",
	//			BehaviorID: "second",
	//		},
	//	},
	//	Transitions: []flowstate.Transition{
	//		{
	//			ID:     "firstTID",
	//			FromID: "",
	//			ToID:   "firstNID",
	//		},
	//		{
	//			ID:     "secondTID",
	//			FromID: "firstNID",
	//			ToID:   "secondNID",
	//		},
	//	},
	//}

	trkr := &tracker2{}

	br := &flowstate.MapBehaviorRegistry{}
	br.SetBehavior("first", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track2(taskCtx, trkr)
		return flowstate.Transit(taskCtx, `second`), nil
	}))
	br.SetBehavior("second", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track2(taskCtx, trkr)
		return flowstate.End(taskCtx), nil
	}))

	d := &memdriver.Driver{}
	e := flowstate.NewEngine(d, br)

	taskCtx := &flowstate.TaskCtx{
		Current: flowstate.Task{
			ID:  "aTID",
			Rev: 0,
		},
	}

	require.NoError(t, e.Do(flowstate.Transit(taskCtx, `first`)))
	err := e.Execute(taskCtx)

	require.NoError(t, err)
	require.Equal(t, []string{`first`, `second`}, trkr.Visited())
}
