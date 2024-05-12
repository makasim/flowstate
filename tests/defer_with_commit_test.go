package tests

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
)

func TestDefer_TransitWin_WithCommit(t *testing.T) {
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

		if flowstate.Deferred(taskCtx) {
			return flowstate.Commit(
				flowstate.Transit(taskCtx, `secondTID`),
			), nil
		}

		if err := taskCtx.Engine.Do(
			flowstate.Defer(taskCtx, time.Millisecond*200),
		); err != nil {
			return nil, err
		}

		return flowstate.Commit(
			flowstate.Transit(taskCtx, `thirdTID`),
		), nil
	}))
	br.SetBehavior("end", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
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
	taskCtx.Current.CopyTo(&taskCtx.Committed)

	err := e.Execute(taskCtx)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 500)

	// no secondTID in list
	require.Equal(t, []flowstate.TransitionID{`firstTID`, `thirdTID`, `firstTID`}, trkr.Visited())
}

func TestDefer_DeferWin_WithCommit(t *testing.T) {
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

		if flowstate.Deferred(taskCtx) {
			return flowstate.Commit(
				flowstate.Transit(taskCtx, `secondTID`),
			), nil
		}

		if err := taskCtx.Engine.Do(
			flowstate.Defer(taskCtx, time.Millisecond*200),
		); err != nil {
			return nil, err
		}

		time.Sleep(time.Millisecond * 300)

		return flowstate.Commit(
			flowstate.Transit(taskCtx, `thirdTID`),
		), nil
	}))
	br.SetBehavior("end", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
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
	taskCtx.Current.CopyTo(&taskCtx.Committed)

	err := e.Execute(taskCtx)
	require.EqualError(t, err, `driver: commit: conflict; cmd: *flowstate.TransitCommand tid: aTID; err: rev mismatch;`)

	time.Sleep(time.Millisecond * 500)

	require.Equal(t, []flowstate.TransitionID{`firstTID`, `firstTID`, `secondTID`}, trkr.Visited())
}
