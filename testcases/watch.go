package testcases

import (
	"context"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func Watch(t TestingT, d flowstate.Doer, fr flowRegistry) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	fr.SetFlow("first", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		return flowstate.Commit(
			flowstate.Transit(stateCtx, `second`),
		), nil
	}))
	fr.SetFlow("second", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		return flowstate.Commit(
			flowstate.Transit(stateCtx, `third`),
		), nil
	}))
	fr.SetFlow("third", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		return flowstate.Commit(
			flowstate.End(stateCtx),
		), nil
	}))

	e, err := flowstate.NewEngine(d)
	defer func() {
		sCtx, sCtxCancel := context.WithTimeout(context.Background(), time.Second*5)
		defer sCtxCancel()

		require.NoError(t, e.Shutdown(sCtx))
	}()

	w, err := e.Watch(0, map[string]string{
		`theWatchLabel`: `theValue`,
	})
	require.NoError(t, err)
	defer w.Close()

	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTID",
		},
	}
	stateCtx.Current.SetLabel("theWatchLabel", `theValue`)

	err = e.Do(flowstate.Commit(
		flowstate.Transit(stateCtx, `first`),
	))
	require.NoError(t, err)

	err = e.Execute(stateCtx)
	require.NoError(t, err)

	var visited []flowstate.FlowID
	timeoutT := time.NewTimer(time.Second * 3)

loop:
	for {
		select {
		case state := <-w.Watch():
			visited = append(visited, state.Transition.ToID)

			if len(visited) >= 3 {
				break loop
			}
		case <-timeoutT.C:
			break loop
		}
	}

	require.Equal(t, []flowstate.FlowID{`first`, `second`, `third`}, visited)

}
