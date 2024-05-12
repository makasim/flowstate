package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestRateLimit(t *testing.T) {
	pLimiter := flowstate.Process{
		ID:  "limiterPID",
		Rev: 123,
		Nodes: []flowstate.Node{
			{
				ID:         "limiterNID",
				BehaviorID: "limiter",
			},
		},
		Transitions: []flowstate.Transition{
			{
				ID:   "limiter",
				ToID: "limiterNID",
			},
		},
	}

	p := flowstate.Process{
		ID:  "simplePID",
		Rev: 1,
		Nodes: []flowstate.Node{
			{
				ID:         "limitedNID",
				BehaviorID: "limited",
			},
		},
		Transitions: []flowstate.Transition{
			{
				ID:   "limited",
				ToID: `limitedNID`,
			},
		},
	}

	// todo: workaround till taskCtx does not implement context.Context
	closeCh := make(chan struct{})

	trkr := &tracker{
		IncludeState: true,
	}

	br := &flowstate.MapBehaviorRegistry{}
	br.SetBehavior("limiter", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		// The zero value of Sometimes behaves like sync.Once, though less efficiently.
		l := rate.NewLimiter(rate.Every(time.Millisecond*100), 1)

		w, err := taskCtx.Engine.Watch(0, map[string]string{"limiter": "theName"})
		if err != nil {
			return nil, err
		}
		defer w.Close()

		for {
			select {
			case limitedTaskCtx := <-w.Watch():
				ctx, _ := context.WithTimeout(context.Background(), time.Second)
				if err := l.Wait(ctx); err != nil {
					return nil, err
				}

				delete(limitedTaskCtx.Current.Labels, "limiter")
				if err := taskCtx.Engine.Do(flowstate.Commit(
					flowstate.Resume(limitedTaskCtx),
					flowstate.Execute(limitedTaskCtx),
				)); err != nil {
					return nil, err
				}
			case <-closeCh:
				return flowstate.Nop(taskCtx), nil
			}
		}
	}))
	br.SetBehavior("limited", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(taskCtx, trkr)

		if flowstate.Resumed(taskCtx) {
			return flowstate.Commit(
				flowstate.End(taskCtx),
			), nil
		}

		taskCtx.Current.SetLabel("limiter", "theName")

		return flowstate.Commit(
			flowstate.Pause(taskCtx, taskCtx.Current.Transition.ID),
		), nil
	}))

	limiterTaskCtx := &flowstate.TaskCtx{
		Current: flowstate.Task{
			ID:         "aLimiterTID",
			Rev:        0,
			ProcessID:  pLimiter.ID,
			ProcessRev: pLimiter.Rev,
		},
		Process: pLimiter,
	}

	var tasks []*flowstate.TaskCtx
	for i := 0; i < 10; i++ {
		taskCtx := &flowstate.TaskCtx{
			Current: flowstate.Task{
				ID:         flowstate.TaskID(fmt.Sprintf("aTID%d", i)),
				Rev:        0,
				ProcessID:  p.ID,
				ProcessRev: p.Rev,
			},
			Process: p,
		}
		tasks = append(tasks, taskCtx)
	}

	d := &memdriver.Driver{}
	e := flowstate.NewEngine(d, br)

	err := e.Do(flowstate.Commit(
		flowstate.Transit(limiterTaskCtx, `limiter`),
		flowstate.Execute(limiterTaskCtx),
	))
	require.NoError(t, err)

	for _, taskCtx := range tasks {
		err = e.Do(flowstate.Commit(
			flowstate.Transit(taskCtx, `limited`),
			flowstate.Execute(taskCtx),
		))
		require.NoError(t, err)
	}

	time.Sleep(time.Millisecond * 450)

	close(closeCh)

	require.Equal(t, []flowstate.TransitionID{
		"limited",
		"limited",
		"limited",
		"limited",
		"limited",
		"limited",
		"limited",
		"limited",
		"limited",
		"limited",

		// rate limited
		"limited:resumed",
		"limited:resumed",
		"limited:resumed",
		"limited:resumed",
		"limited:resumed",
	}, trkr.Visited())
}
