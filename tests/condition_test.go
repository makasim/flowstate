package tests

import (
	"testing"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
)

func TestCondition(t *testing.T) {
	trkr := &tracker2{}

	br := &flowstate.MapBehaviorRegistry{}
	br.SetBehavior("first", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track2(taskCtx, trkr)

		bID := flowstate.BehaviorID(`third`)
		if taskCtx.Current.Annotations["condition"] == "true" {
			bID = `second`
		}

		return flowstate.Transit(taskCtx, bID), nil
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

	// condition true
	taskCtx := &flowstate.TaskCtx{
		Current: flowstate.Task{
			ID: "aTrueTID",
			Annotations: map[string]string{
				"condition": "true",
			},
		},
	}

	require.NoError(t, e.Do(flowstate.Transit(taskCtx, `first`)))
	require.NoError(t, e.Execute(taskCtx))
	require.Equal(t, []string{`first`, `second`}, trkr.Visited())

	// condition false
	trkr.visited = nil

	taskCtx1 := &flowstate.TaskCtx{
		Current: flowstate.Task{
			ID: "aFalseTID",
			Annotations: map[string]string{
				"condition": "false",
			},
		},
	}

	require.NoError(t, e.Do(flowstate.Transit(taskCtx1, `first`)))
	require.NoError(t, e.Execute(taskCtx1))
	require.Equal(t, []string{`first`, `third`}, trkr.Visited())
}
