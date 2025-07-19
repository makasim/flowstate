package testcases

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func RateLimit(t *testing.T, e flowstate.Engine, fr flowstate.FlowRegistry, d flowstate.Driver) {
	trkr := &Tracker{
		IncludeState: true,
	}

	mustSetFlow(fr, "limiter", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		// The zero value of Sometimes behaves like sync.Once, though less efficiently.
		l := rate.NewLimiter(rate.Every(time.Millisecond*100), 1)

		w := flowstate.NewWatcher(e, flowstate.GetStatesByLabels(map[string]string{
			"limiter": "theName",
		}))
		defer w.Close()

		for {
			select {
			case limitedState := <-w.Next():
				limitedStateCtx := limitedState.CopyToCtx(&flowstate.StateCtx{})

				ctx, _ := context.WithTimeout(context.Background(), time.Second)
				if err := l.Wait(ctx); err != nil {
					return nil, err
				}

				if !flowstate.Paused(limitedStateCtx.Current) {
					continue
				}

				delete(limitedStateCtx.Current.Labels, "limiter")
				if err := e.Do(
					flowstate.Commit(
						flowstate.Resume(limitedStateCtx),
					),
					flowstate.Execute(limitedStateCtx),
				); err != nil {
					return nil, err
				}
			case <-stateCtx.Done():
				return flowstate.Noop(stateCtx), nil
			}
		}
	}))
	mustSetFlow(fr, "limited", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		if flowstate.Resumed(stateCtx.Current) {
			return flowstate.Commit(
				flowstate.End(stateCtx),
			), nil
		}

		stateCtx.Current.SetLabel("limiter", "theName")
		return flowstate.Commit(
			flowstate.Pause(stateCtx),
		), nil
	}))

	limiterStateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID:  "aLimiterTID",
			Rev: 0,
		},
	}

	var states []*flowstate.StateCtx
	for i := 0; i < 10; i++ {
		stateCtx := &flowstate.StateCtx{
			Current: flowstate.State{
				ID:  flowstate.StateID(fmt.Sprintf("aTID%d", i)),
				Rev: 0,
			},
		}
		states = append(states, stateCtx)
	}

	require.NoError(t, e.Do(
		flowstate.Commit(
			flowstate.Transit(limiterStateCtx, `limiter`),
		),
		flowstate.Execute(limiterStateCtx),
	))

	for _, stateCtx := range states {
		require.NoError(t, e.Do(
			flowstate.Commit(
				flowstate.Transit(stateCtx, `limited`),
			),
			flowstate.Execute(stateCtx),
		))
	}

	var visited []string
	assert.Eventually(t, func() bool {
		visited = trkr.VisitedSorted()
		return len(visited) >= 15
	}, time.Second, time.Millisecond*50)
	require.Len(t, visited, 15, "visited %d: %v; need 15", len(visited), visited)

	sCtx, sCtxCancel := context.WithTimeout(context.Background(), time.Second*5)
	defer sCtxCancel()

	require.NoError(t, e.Shutdown(sCtx))

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
	}, visited)
}
