package tests

import (
	"testing"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
)

func TestTwoConsequentNodesWithCommit(t *testing.T) {
	trkr := &tracker2{}

	br := &flowstate.MapFlowRegistry{}
	br.SetFlow("first", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		track2(stateCtx, trkr)
		return flowstate.Commit(
			flowstate.Transit(stateCtx, `second`),
		), nil
	}))
	br.SetFlow("second", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		track2(stateCtx, trkr)
		return flowstate.End(stateCtx), nil
	}))

	d := &memdriver.Driver{}
	e := flowstate.NewEngine(d, br)

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
