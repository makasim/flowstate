package testcases

import (
	"context"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func RecoveryFirstAttemptFail(t TestingT, d flowstate.Doer, fr flowRegistry) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	trkr := &Tracker{IncludeState: true}

	fr.SetFlow("first_attempt_fail", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		if flowstate.RecoveryAttempt(stateCtx.Current) < 1 {
			// simulate a fail on first attempt
			return flowstate.Noop(stateCtx), nil
		}

		return flowstate.Commit(
			flowstate.End(stateCtx),
		), nil
	}))

	e, err := flowstate.NewEngine(d)
	require.NoError(t, err)
	defer func() {
		sCtx, sCtxCancel := context.WithTimeout(context.Background(), time.Second*5)
		defer sCtxCancel()

		require.NoError(t, e.Shutdown(sCtx))
	}()

	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID:  "aTID",
			Rev: 0,
			Labels: map[string]string{
				`theRecovery`: `aTID`,
			},
		},
	}

	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Transit(stateCtx, `first_attempt_fail`))),
	)
	require.NoError(t, e.Execute(stateCtx))

	wCmd := flowstate.GetWatcher(0, map[string]string{
		`theRecovery`: `aTID`,
	})
	require.NoError(t, e.Do(wCmd))

	w := wCmd.Watcher
	defer w.Close()

	var visited []string
loop:
	for {
		select {
		case latestState := <-w.Watch():
			if flowstate.Ended(latestState) {
				visited = append(visited, "ended")
				break loop
			}
			visited = append(visited, string(latestState.Transition.ToID))
		case <-time.NewTimer(time.Second * 2).C:
			t.Fatalf("expected to receive a state")
		}
	}

	require.Equal(t, []string{"first_attempt_fail", "first_attempt_fail", "ended"}, visited)
}
