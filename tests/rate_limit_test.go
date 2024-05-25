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
	// todo: workaround till taskCtx does not implement context.Context
	closeCh := make(chan struct{})

	trkr := &tracker2{
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

				if !(limitedTaskCtx.Current.Transition.ToID == `limited` && flowstate.Paused(limitedTaskCtx)) {
					continue
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
		track2(taskCtx, trkr)

		if flowstate.Resumed(taskCtx) {
			return flowstate.Commit(
				flowstate.End(taskCtx),
			), nil
		}

		taskCtx.Current.SetLabel("limiter", "theName")
		return flowstate.Commit(
			flowstate.Pause(taskCtx, taskCtx.Current.Transition.ToID),
		), nil
	}))

	limiterTaskCtx := &flowstate.TaskCtx{
		Current: flowstate.Task{
			ID:  "aLimiterTID",
			Rev: 0,
		},
	}

	var tasks []*flowstate.TaskCtx
	for i := 0; i < 10; i++ {
		taskCtx := &flowstate.TaskCtx{
			Current: flowstate.Task{
				ID:  flowstate.TaskID(fmt.Sprintf("aTID%d", i)),
				Rev: 0,
			},
		}
		tasks = append(tasks, taskCtx)
	}

	d := &memdriver.Driver{}
	e := flowstate.NewEngine(d, br)

	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Transit(limiterTaskCtx, `limiter`),
		flowstate.Execute(limiterTaskCtx),
	)))

	for _, taskCtx := range tasks {
		require.NoError(t, e.Do(flowstate.Commit(
			flowstate.Transit(taskCtx, `limited`),
			flowstate.Execute(taskCtx),
		)))
	}

	var visited []string
	require.Eventually(t, func() bool {
		visited = trkr.Visited()
		return len(visited) >= 15
	}, time.Second, time.Millisecond*10)

	close(closeCh)

	require.Equal(t, []string{
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
