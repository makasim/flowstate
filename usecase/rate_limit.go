package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"golang.org/x/time/rate"
)

func RateLimit(t TestingT, d flowstate.Doer, fr flowRegistry) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	trkr := &Tracker{
		IncludeState: true,
	}

	fr.SetFlow("limiter", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		// The zero value of Sometimes behaves like sync.Once, though less efficiently.
		l := rate.NewLimiter(rate.Every(time.Millisecond*100), 1)

		w, err := e.Watch(0, map[string]string{"limiter": "theName"})
		if err != nil {
			return nil, err
		}
		defer w.Close()

		for {
			select {
			case limitedState := <-w.Watch():
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
	fr.SetFlow("limited", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		if flowstate.Resumed(stateCtx.Current) {
			return flowstate.Commit(
				flowstate.End(stateCtx),
			), nil
		}

		stateCtx.Current.SetLabel("limiter", "theName")
		return flowstate.Commit(
			flowstate.Pause(stateCtx, stateCtx.Current.Transition.ToID),
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

	e, err := flowstate.NewEngine(d)
	require.NoError(t, err)
	defer func() {
		sCtx, sCtxCancel := context.WithTimeout(context.Background(), time.Second*5)
		defer sCtxCancel()

		require.NoError(t, e.Shutdown(sCtx))
	}()

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
