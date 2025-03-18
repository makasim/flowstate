package testcases

import (
	"context"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func RecoveryFirstAttemptFail(t TestingT, d flowstate.Doer, fr FlowRegistry) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	trkr := &Tracker{IncludeState: true}

	fr.SetFlow("first_attempt_fail", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		if flowstate.RecoveryAttempt(stateCtx.Current) < 1 {
			// simulate a fail on first attempt
			return flowstate.Noop(stateCtx), nil
		}

		return flowstate.Commit(
			flowstate.End(stateCtx),
		), nil
	}))

	l, _ := NewTestLogger(t)
	e, err := flowstate.NewEngine(d, l)
	require.NoError(t, err)
	defer func() {
		sCtx, sCtxCancel := context.WithTimeout(context.Background(), time.Second*5)
		defer sCtxCancel()

		require.NoError(t, e.Shutdown(sCtx))
	}()

	r := flowstate.Recoverer(time.Millisecond * 500)
	require.NoError(t, r.Init(e))
	defer func() {
		sCtx, sCtxCancel := context.WithTimeout(context.Background(), time.Second*5)
		defer sCtxCancel()

		require.NoError(t, r.Shutdown(sCtx))
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

	w := flowstate.NewWatcher(e, flowstate.GetManyByLabels(map[string]string{
		`theRecovery`: `aTID`,
	}))
	defer w.Close()

	var visited []string
loop:
	for {
		select {
		case latestState := <-w.Next():
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
