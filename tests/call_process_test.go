package tests

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
)

func TestCallProcess(t *testing.T) {
	var nextTaskCtx *flowstate.TaskCtx
	taskCtx := &flowstate.TaskCtx{
		Current: flowstate.Task{
			ID: "aTID",
		},
	}

	endedCh := make(chan struct{})
	trkr := &tracker2{}

	br := &flowstate.MapBehaviorRegistry{}
	br.SetBehavior("call", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track2(taskCtx, trkr)

		if flowstate.Resumed(taskCtx) {
			return flowstate.Transit(taskCtx, `callEnd`), nil
		}

		nextTaskCtx = &flowstate.TaskCtx{
			Current: flowstate.Task{
				ID: "aTID",
			},
		}

		if err := taskCtx.Engine.Do(
			flowstate.Pause(taskCtx, taskCtx.Current.Transition.ToID),
			flowstate.Stack(taskCtx, nextTaskCtx),
			flowstate.Transit(nextTaskCtx, `called`),
			flowstate.Execute(nextTaskCtx),
		); err != nil {
			return nil, err
		}

		return flowstate.Nop(taskCtx), nil
	}))
	br.SetBehavior("called", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track2(taskCtx, trkr)
		return flowstate.Transit(taskCtx, `calledEnd`), nil
	}))
	br.SetBehavior("calledEnd", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track2(taskCtx, trkr)

		if flowstate.Stacked(taskCtx) {
			callTaskCtx := &flowstate.TaskCtx{}

			if err := taskCtx.Engine.Do(
				flowstate.Unstack(taskCtx, callTaskCtx),
				flowstate.Resume(callTaskCtx),
				flowstate.Execute(callTaskCtx),
				flowstate.End(taskCtx),
			); err != nil {
				return nil, err
			}

			return flowstate.Nop(taskCtx), nil
		}

		return flowstate.End(taskCtx), nil
	}))

	br.SetBehavior("callEnd", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track2(taskCtx, trkr)

		close(endedCh)

		return flowstate.End(taskCtx), nil
	}))

	d := &memdriver.Driver{}
	e := flowstate.NewEngine(d, br)

	require.NoError(t, e.Do(flowstate.Transit(taskCtx, `call`)))
	require.NoError(t, e.Execute(taskCtx))

	require.Eventually(t, func() bool {
		select {
		case <-endedCh:
			return true
		default:
			return false
		}
	}, time.Second*5, time.Millisecond*50)

	require.Equal(t, []string{
		`call`,
		`called`,
		`calledEnd`,
		`call`,
		`callEnd`,
	}, trkr.Visited())
}
