package testcases

import (
	"testing"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func TwoConsequentNodes(t *testing.T, e flowstate.Engine, d flowstate.Driver) {
	trkr := &Tracker{}

	mustSetFlow(d, "first", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.Transit(stateCtx, `second`), nil
	}))
	mustSetFlow(d, "second", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.End(stateCtx), nil
	}))

	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID:  "aTID",
			Rev: 0,
		},
	}

	require.NoError(t, e.Do(flowstate.Transit(stateCtx, `first`)))
	err := e.Execute(stateCtx)

	require.NoError(t, err)
	require.Equal(t, []string{`first`, `second`}, trkr.Visited())
}
