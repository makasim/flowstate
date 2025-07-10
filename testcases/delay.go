package testcases

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func Delay(t *testing.T, e flowstate.Engine, d flowstate.Driver) {
	trkr := &Tracker{}

	mustSetFlow(d, "first", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		if flowstate.Delayed(stateCtx.Current) {
			return flowstate.Transit(stateCtx, `second`), nil
		}

		return flowstate.Delay(stateCtx, time.Millisecond*200), nil
	}))
	mustSetFlow(d, "second", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.End(stateCtx), nil
	}))

	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTID",
		},
	}

	require.NoError(t, e.Do(flowstate.Transit(stateCtx, `first`)))
	require.NoError(t, e.Execute(stateCtx))

	trkr.WaitVisitedEqual(t, []string{`first`, `first`, `second`}, time.Second*20)
}
