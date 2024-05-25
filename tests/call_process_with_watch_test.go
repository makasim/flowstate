package tests

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
)

func TestCallProcessWithWatch(t *testing.T) {
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

		if taskCtx.Current.Annotations[`called`] == `` {
			nextTaskCtx = &flowstate.TaskCtx{
				Current: flowstate.Task{
					ID: "aNextTID",
				},
			}
			nextTaskCtx.Committed = nextTaskCtx.Current
			nextTaskCtx.Current.SetLabel("theWatchLabel", string(taskCtx.Current.ID))

			taskCtx.Current.SetAnnotation("called", `true`)

			if err := taskCtx.Engine.Do(
				flowstate.Commit(
					flowstate.Transit(taskCtx, `call`),
					flowstate.Transit(nextTaskCtx, `called`),
					flowstate.Execute(nextTaskCtx),
				),
			); err != nil {
				return nil, err
			}
		}

		w, err := taskCtx.Engine.Watch(taskCtx.Committed.Rev, map[string]string{
			`theWatchLabel`: string(taskCtx.Current.ID),
		})
		if err != nil {
			return nil, err
		}
		defer w.Close()

		for {
			select {
			// todo: case <-taskCtx.Done()
			case nextTaskCtx := <-w.Watch():
				if !flowstate.Ended(nextTaskCtx) {
					continue
				}

				delete(taskCtx.Current.Annotations, `called`)

				return flowstate.Commit(
					flowstate.Transit(taskCtx, `callEnd`),
				), nil
			}
		}

	}))
	br.SetBehavior("called", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track2(taskCtx, trkr)
		return flowstate.Transit(taskCtx, `calledEnd`), nil
	}))
	br.SetBehavior("calledEnd", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track2(taskCtx, trkr)

		return flowstate.Commit(
			flowstate.End(taskCtx),
		), nil
	}))
	br.SetBehavior("callEnd", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track2(taskCtx, trkr)

		close(endedCh)

		return flowstate.Commit(
			flowstate.End(taskCtx),
		), nil
	}))

	d := &memdriver.Driver{}
	e := flowstate.NewEngine(d, br)

	err := e.Do(flowstate.Commit(
		flowstate.Transit(taskCtx, `call`),
	))
	require.NoError(t, err)

	err = e.Execute(taskCtx)
	require.NoError(t, err)

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
		`callEnd`,
	}, trkr.Visited())
}
