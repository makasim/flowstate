package tests

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
)

func TestFork(t *testing.T) {
	var forkedTaskCtx *flowstate.TaskCtx
	taskCtx := &flowstate.TaskCtx{
		Current: flowstate.Task{
			ID: "aTID",
		},
	}

	trkr := &tracker2{}

	br := &flowstate.MapBehaviorRegistry{}
	br.SetBehavior("fork", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track2(taskCtx, trkr)

		forkedTaskCtx = &flowstate.TaskCtx{}
		taskCtx.CopyTo(forkedTaskCtx)
		forkedTaskCtx.Current.ID = "forkedTID"
		forkedTaskCtx.Current.Rev = 0
		forkedTaskCtx.Current.CopyTo(&forkedTaskCtx.Committed)

		if err := taskCtx.Engine.Do(
			flowstate.Transit(taskCtx, `origin`),
			flowstate.Transit(forkedTaskCtx, `forked`),

			flowstate.Execute(taskCtx),
			flowstate.Execute(forkedTaskCtx),
		); err != nil {
			return nil, err
		}

		return flowstate.Nop(taskCtx), nil
	}))
	br.SetBehavior("origin", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track2(taskCtx, trkr)
		return flowstate.End(taskCtx), nil
	}))
	br.SetBehavior("forked", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track2(taskCtx, trkr)
		return flowstate.End(taskCtx), nil
	}))

	d := &memdriver.Driver{}
	e := flowstate.NewEngine(d, br)

	require.NoError(t, e.Do(flowstate.Transit(taskCtx, `fork`)))
	require.NoError(t, e.Execute(taskCtx))

	time.Sleep(time.Millisecond * 100)

	require.Equal(t, []string{
		`fork`,
		`forked`,
		`origin`,
	}, trkr.VisitedSorted())
}
