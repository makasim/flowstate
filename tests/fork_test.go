package tests

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
)

func TestFork(t *testing.T) {
	p := flowstate.Process{
		ID:  "simplePID",
		Rev: 1,
		Nodes: []flowstate.Node{
			{
				ID:         "firstNID",
				BehaviorID: "fork",
			},
			{
				ID:         "forkedNID",
				BehaviorID: "end",
			},
			{
				ID:         "originNID",
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
				ID:     "forkedTID",
				FromID: "firstNID",
				ToID:   "forkedNID",
			},
			{
				ID:     "originTID",
				FromID: "firstNID",
				ToID:   "originNID",
			},
		},
	}

	var forkedTaskCtx *flowstate.TaskCtx
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

	trkr := &tracker{}

	br := &flowstate.MapBehaviorRegistry{}
	br.SetBehavior("fork", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(taskCtx, trkr)

		forkedTaskCtx = &flowstate.TaskCtx{}
		taskCtx.CopyTo(forkedTaskCtx)
		forkedTaskCtx.Current.ID = "forkedTID"
		forkedTaskCtx.Current.Rev = 0
		forkedTaskCtx.Current.CopyTo(&forkedTaskCtx.Committed)

		if err := taskCtx.Engine.Do(
			flowstate.Transit(taskCtx, `originTID`),
			flowstate.Transit(forkedTaskCtx, `forkedTID`),
		); err != nil {
			return nil, err
		}

		return flowstate.Nop(taskCtx), nil
	}))
	br.SetBehavior("end", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(taskCtx, trkr)
		return flowstate.End(taskCtx), nil
	}))

	d := &memdriver.Driver{}
	e := flowstate.NewEngine(d, br)

	err := e.Execute(taskCtx)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 100)

	require.Equal(t, []flowstate.TransitionID{
		`firstTID`,
		`forkedTID`,
		`originTID`,
	}, trkr.VisitedSorted())
}
