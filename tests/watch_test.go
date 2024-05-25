package tests

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
)

func TestWatch(t *testing.T) {
	br := &flowstate.MapBehaviorRegistry{}
	br.SetBehavior("first", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		return flowstate.Commit(
			flowstate.Transit(taskCtx, `second`),
		), nil
	}))
	br.SetBehavior("second", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		return flowstate.Commit(
			flowstate.Transit(taskCtx, `third`),
		), nil
	}))
	br.SetBehavior("third", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		return flowstate.Commit(
			flowstate.End(taskCtx),
		), nil
	}))

	d := &memdriver.Driver{}
	e := flowstate.NewEngine(d, br)

	w, err := e.Watch(0, map[string]string{
		`theWatchLabel`: `theValue`,
	})
	require.NoError(t, err)
	defer w.Close()

	taskCtx := &flowstate.TaskCtx{
		Current: flowstate.Task{
			ID: "aTID",
		},
	}
	taskCtx.Current.SetLabel("theWatchLabel", `theValue`)

	err = e.Do(flowstate.Commit(
		flowstate.Transit(taskCtx, `first`),
	))
	require.NoError(t, err)

	err = e.Execute(taskCtx)
	require.NoError(t, err)

	var visited []flowstate.BehaviorID
	timeoutT := time.NewTimer(time.Second * 3)

loop:
	for {
		select {
		case taskCtx := <-w.Watch():
			visited = append(visited, taskCtx.Current.Transition.ToID)

			if len(visited) >= 3 {
				break loop
			}
		case <-timeoutT.C:
			break loop
		}
	}

	require.Equal(t, []flowstate.BehaviorID{`first`, `second`, `third`}, visited)

}
