package tests

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
)

func TestForkJoin_FirstWins(t *testing.T) {
	var forkedTaskCtx *flowstate.TaskCtx
	var forkedTwoTaskCtx *flowstate.TaskCtx
	taskCtx := &flowstate.TaskCtx{
		Current: flowstate.Task{
			ID: "aTID",
		},
	}

	trkr := &tracker2{}

	br := &flowstate.MapBehaviorRegistry{}
	br.SetBehavior("fork", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track2(taskCtx, trkr)

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

			flowstate.Transit(taskCtx, `forked`),
			flowstate.Transit(forkedTaskCtx, `forked`),
			flowstate.Transit(forkedTwoTaskCtx, `forked`),

			flowstate.Execute(taskCtx),
			flowstate.Execute(forkedTaskCtx),
			flowstate.Execute(forkedTwoTaskCtx),
		)); err != nil {
			return nil, err
		}

		return flowstate.Nop(taskCtx), nil
	}))
	br.SetBehavior("join", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track2(taskCtx, trkr)

		if taskCtx.Committed.Transition.ToID != `join` {
			if err := taskCtx.Engine.Do(flowstate.Commit(
				flowstate.Transit(taskCtx, `join`),
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

		cnt := 0
		for {
			select {
			case changedTaskCtx := <-w.Watch():
				if changedTaskCtx.Current.Transition.ToID != `join` {
					continue
				}
				cnt++

				if changedTaskCtx.Current.ID != taskCtx.Current.ID {
					continue
				}

				if cnt == 1 {
					return flowstate.Commit(
						flowstate.Transit(taskCtx, `joined`),
					), nil
				}

				return flowstate.Commit(
					flowstate.End(taskCtx),
				), nil

			}
		}
	}))

	br.SetBehavior("forked", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track2(taskCtx, trkr)
		return flowstate.Transit(taskCtx, `join`), nil
	}))

	br.SetBehavior("joined", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track2(taskCtx, trkr)
		return flowstate.End(taskCtx), nil
	}))

	d := &memdriver.Driver{}
	e := flowstate.NewEngine(d, br)

	require.NoError(t, e.Do(flowstate.Transit(taskCtx, `fork`)))
	require.NoError(t, e.Execute(taskCtx))

	time.Sleep(time.Millisecond * 100)

	require.Equal(t, []string{
		"fork",
		"forked",
		"forked",
		"forked",
		"join",
		"join",
		"join",
		"joined",
	}, trkr.VisitedSorted())
}
