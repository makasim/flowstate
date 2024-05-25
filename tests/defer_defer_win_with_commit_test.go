package tests

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
)

func TestDefer_DeferWin_WithCommit(t *testing.T) {
	trkr := &tracker2{}

	br := &flowstate.MapBehaviorRegistry{}
	br.SetBehavior("first", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track2(taskCtx, trkr)

		if flowstate.Deferred(taskCtx) {
			return flowstate.Commit(
				flowstate.Transit(taskCtx, `second`),
			), nil
		}

		if err := taskCtx.Engine.Do(
			flowstate.Defer(taskCtx, time.Millisecond*200),
		); err != nil {
			return nil, err
		}

		time.Sleep(time.Millisecond * 300)

		return flowstate.Commit(
			flowstate.Transit(taskCtx, `third`),
		), nil
	}))
	br.SetBehavior("second", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track2(taskCtx, trkr)
		return flowstate.End(taskCtx), nil
	}))
	br.SetBehavior("third", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track2(taskCtx, trkr)
		return flowstate.End(taskCtx), nil
	}))

	d := &memdriver.Driver{}
	e := flowstate.NewEngine(d, br)

	taskCtx := &flowstate.TaskCtx{
		Current: flowstate.Task{
			ID: "aTID",
		},
	}
	taskCtx.Current.CopyTo(&taskCtx.Committed)

	require.NoError(t, e.Do(flowstate.Transit(taskCtx, `first`)))
	require.NoError(t, e.Execute(taskCtx))

	time.Sleep(time.Millisecond * 500)

	// no third in list
	require.Equal(t, []string{`first`, `first`, `second`}, trkr.Visited())
}
