package usecase

import (
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func Watch(t TestingT, d flowstate.Doer, fr flowRegistry) {
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
		case stateCtx := <-w.Watch():
			visited = append(visited, stateCtx.Current.Transition.ToID)

			if len(visited) >= 3 {
				break loop
			}
		case <-timeoutT.C:
			break loop
		}
	}

	require.Equal(t, []flowstate.FlowID{`first`, `second`, `third`}, visited)

}
