package testcases

import (
	"testing"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func TwoConsequentNodesWithCommit(t *testing.T, e flowstate.Engine, fr flowstate.FlowRegistry, d flowstate.Driver) {
	trkr := &Tracker{}

	mustSetFlow(fr, "first", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.Commit(
			flowstate.Transit(stateCtx, `second`),
		), nil
	}))
	mustSetFlow(fr, "second", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.Park(stateCtx), nil
	}))

	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID:  "aTID",
			Rev: 0,
		},
	}

	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Transit(stateCtx, `first`),
	)))
	require.NoError(t, e.Execute(stateCtx))

	require.Equal(t, []string{`first`, `second`}, trkr.Visited())
}
