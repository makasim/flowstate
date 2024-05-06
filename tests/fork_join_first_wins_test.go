package tests

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
)

func TestForkJoin_FirstWins(t *testing.T) {
	p := flowstate.Process{
		ID:  "simplePID",
		Rev: 1,
		Nodes: []flowstate.Node{
			{
				ID:         "forkNID",
				BehaviorID: "fork",
			},
			{
				ID:         "forkedNID",
				BehaviorID: "forked",
			},
			{
				ID:         "joinNID",
				BehaviorID: "join",
			},
			{
				ID:         "joinedNID",
				BehaviorID: "joined",
			},
		},
		Transitions: []flowstate.Transition{
			{
				ID:     "forkTID",
				FromID: "",
				ToID:   "forkNID",
			},
			{
				ID:     "forkedTID",
				FromID: "forkNID",
				ToID:   "forkedNID",
			},
			{
				ID:     "joinTID",
				FromID: "forkedNID",
				ToID:   "joinNID",
			},
			{
				ID:     "joinedTID",
				FromID: "joinNID",
				ToID:   "joinedNID",
			},
		},
	}

	var forkedTaskCtx *flowstate.TaskCtx
	var forkedTwoTaskCtx *flowstate.TaskCtx
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
		forkedTaskCtx.Current.ID = "forkedTID"

		forkedTwoTaskCtx = &flowstate.TaskCtx{}
		forkedTwoTaskCtx.Current.ID = "forkedTwoTID"

		if err := taskCtx.Engine.Do(
			flowstate.Fork(taskCtx, forkedTaskCtx),
			flowstate.Fork(taskCtx, forkedTwoTaskCtx),

			flowstate.AddAtomicInt64(`theAtomicKey`, 1, forkedTwoTaskCtx),
			flowstate.AddAtomicInt64(`theAtomicKey`, 1, forkedTaskCtx),
			flowstate.AddAtomicInt64(`theAtomicKey`, 1, taskCtx),

			flowstate.Transit(taskCtx, `forkedTID`),
			flowstate.Transit(forkedTaskCtx, `forkedTID`),
			flowstate.Transit(forkedTwoTaskCtx, `forkedTID`),
		); err != nil {
			return nil, err
		}

		return flowstate.Nop(taskCtx), nil
	}))
	br.SetBehavior("join", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(taskCtx, trkr)

		if taskCtx.Current.Transition.ID != `joinTID` {
			if err := taskCtx.Engine.Do(
				flowstate.AddAtomicInt64(`theAtomicKey`, -1, taskCtx),
				flowstate.Transit(taskCtx, `joinTID`),
			); err != nil {
				return nil, err
			}

			return flowstate.Nop(taskCtx), nil
		}

		val, err := flowstate.GetAtomicInt64(`theAtomicKey`, taskCtx)
		if err != nil {
			return nil, err
		}

		if val == 1 {
			return flowstate.Transit(taskCtx, `joinedTID`), nil
		} else {
			return flowstate.End(taskCtx), nil
		}
	}))

	br.SetBehavior("forked", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(taskCtx, trkr)
		return flowstate.Transit(taskCtx, `joinTID`), nil
	}))

	br.SetBehavior("joined", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(taskCtx, trkr)
		return flowstate.End(taskCtx), nil
	}))

	d := &memdriver.Driver{}
	e := flowstate.NewEngine(d, br)

	err := e.Execute(taskCtx)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 100)

	require.Equal(t, []flowstate.TransitionID{
		"forkTID",
		"forkedTID",
		"forkedTID",
		"forkedTID",
		"joinTID",
		"joinTID",
		"joinTID",
		"joinedTID",
	}, trkr.VisitedSorted())
}
