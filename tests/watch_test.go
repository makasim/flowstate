package tests

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
)

func TestWatch(t *testing.T) {
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
				BehaviorID: "second",
			},
			{
				ID:         "thirdNID",
				BehaviorID: "third",
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
				FromID: "secondNID",
				ToID:   "thirdNID",
			},
		},
	}

	br := &flowstate.MapBehaviorRegistry{}
	br.SetBehavior("first", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		return flowstate.Commit(
			flowstate.Transit(taskCtx, `secondTID`),
		), nil
	}))
	br.SetBehavior("second", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		return flowstate.Commit(
			flowstate.Transit(taskCtx, `thirdTID`),
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
			ID:         "aTID",
			Rev:        0,
			ProcessID:  p.ID,
			ProcessRev: p.Rev,
		},
		Process: p,
	}
	taskCtx.Current.SetLabel("theWatchLabel", `theValue`)

	err = e.Do(flowstate.Commit(
		flowstate.Transit(taskCtx, `firstTID`),
	))
	require.NoError(t, err)

	err = e.Execute(taskCtx)
	require.NoError(t, err)

	var visited []flowstate.TransitionID
	timeoutT := time.NewTimer(time.Second * 3)

loop:
	for {
		select {
		case taskCtx := <-w.Watch():
			visited = append(visited, taskCtx.Current.Transition.ID)

			if len(visited) >= 3 {
				break loop
			}
		case <-timeoutT.C:
			break loop
		}
	}

	require.Equal(t, []flowstate.TransitionID{`firstTID`, `secondTID`, `thirdTID`}, visited)

}
