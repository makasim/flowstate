package testcases

import (
	"context"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func RecoveryAlwaysFail(t TestingT, d flowstate.Doer, fr flowRegistry) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	trkr := &Tracker{}

	fr.SetFlow("always_fail", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		// simulate a flow always failing
		return flowstate.Noop(stateCtx), nil
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
		flowstate.Transit(stateCtx, `always_fail`))),
	)
	require.NoError(t, e.Execute(stateCtx))

	wCmd := flowstate.Watch(map[string]string{
		`theRecovery`: `aTID`,
	})
	require.NoError(t, e.Do(wCmd))

	w := wCmd.Listener
	defer w.Close()

	var visited []string
loop:
	for {
		select {
		case latestState := <-w.Listen():
			if flowstate.Ended(latestState) {
				visited = append(visited, "ended")
				break loop
			}
			visited = append(visited, string(latestState.Transition.ToID))
		case <-time.NewTimer(time.Second * 2).C:
			t.Fatalf("expected to receive a state")
		}
	}

	require.Equal(t, []string{"always_fail", "always_fail", "always_fail", "ended"}, visited)
}
