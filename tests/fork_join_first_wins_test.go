package tests

import (
	"log"
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
	taskCtx.Current.CopyTo(&taskCtx.Committed)

	trkr := &tracker{}

	br := &flowstate.MapBehaviorRegistry{}
	br.SetBehavior("fork", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(taskCtx, trkr)

		taskCtx.Current.SetLabel(`theForkJoinLabel`, string(taskCtx.Current.ID))

		forkedTaskCtx = &flowstate.TaskCtx{}
		forkedTaskCtx.Current.ID = "forkedTID"
		forkedTaskCtx.Committed.ID = "forkedTID"

		forkedTwoTaskCtx = &flowstate.TaskCtx{}
		forkedTwoTaskCtx.Current.ID = "forkedTwoTID"
		forkedTwoTaskCtx.Committed.ID = "forkedTwoTID"

		if err := taskCtx.Engine.Do(flowstate.Commit(
			flowstate.Fork(taskCtx, forkedTaskCtx),
			flowstate.Fork(taskCtx, forkedTwoTaskCtx),

			flowstate.Transit(taskCtx, `forkedTID`),
			flowstate.Transit(forkedTaskCtx, `forkedTID`),
			flowstate.Transit(forkedTwoTaskCtx, `forkedTID`),

			flowstate.Execute(taskCtx),
			flowstate.Execute(forkedTaskCtx),
			flowstate.Execute(forkedTwoTaskCtx),
		)); err != nil {
			return nil, err
		}

		return flowstate.Nop(taskCtx), nil
	}))
	br.SetBehavior("join", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(taskCtx, trkr)

		if taskCtx.Current.Transition.ID != taskCtx.Committed.Transition.ID {
			log.Println(123, taskCtx.Current.ID)
			if err := taskCtx.Engine.Do(flowstate.Commit(
				flowstate.Transit(taskCtx, `joinTID`),
			)); err != nil {
				return nil, err
			}
		}

		w, err := taskCtx.Engine.Watch(0, map[string]string{
			`theForkJoinLabel`: taskCtx.Current.Labels[`theForkJoinLabel`],
		})
		if err != nil {
			return nil, err
		}
		defer w.Close()

		for {
			select {
			case changedTaskCtx := <-w.Watch():
				if changedTaskCtx.Current.ID != taskCtx.Current.ID {
					return flowstate.Commit(
						flowstate.End(taskCtx),
					), nil
				}

				return flowstate.Commit(
					flowstate.Transit(taskCtx, `joinedTID`),
				), nil
			}
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
