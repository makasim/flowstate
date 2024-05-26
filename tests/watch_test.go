package tests

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
)

func TestWatch(t *testing.T) {
	d := memdriver.New()
	d.SetFlow("first", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx) (flowstate.Command, error) {
		return flowstate.Commit(
			flowstate.Transit(stateCtx, `second`),
		), nil
	}))
	d.SetFlow("second", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx) (flowstate.Command, error) {
		return flowstate.Commit(
			flowstate.Transit(stateCtx, `third`),
		), nil
	}))
	d.SetFlow("third", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx) (flowstate.Command, error) {
		return flowstate.Commit(
			flowstate.End(stateCtx),
		), nil
	}))

	e := flowstate.NewEngine(d)

	wCmd := flowstate.Watch(0, map[string]string{
		`theWatchLabel`: `theValue`,
	})
	require.NoError(t, e.Do(wCmd))
	w := wCmd.Listener
	defer w.Close()

	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTID",
		},
	}
	stateCtx.Current.SetLabel("theWatchLabel", `theValue`)

	err := e.Do(flowstate.Commit(
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
