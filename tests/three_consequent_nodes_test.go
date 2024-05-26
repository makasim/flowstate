package tests

import (
	"testing"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
)

func TestThreeConsequentNodes(t *testing.T) {
	trkr := &tracker2{}

	fr := memdriver.NewFlowRegistry()
	fr.SetFlow("first", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		track2(stateCtx, trkr)
		return flowstate.Transit(stateCtx, `second`), nil
	}))
	fr.SetFlow("second", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		track2(stateCtx, trkr)
		return flowstate.Transit(stateCtx, `third`), nil
	}))
	fr.SetFlow("third", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		track2(stateCtx, trkr)
		return flowstate.End(stateCtx), nil
	}))

	d := &memdriver.Driver{}
	e := flowstate.NewEngine(d, fr)

	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID:  "aTID",
			Rev: 0,
		},
	}

	require.NoError(t, e.Do(flowstate.Transit(stateCtx, `first`)))
	require.NoError(t, e.Execute(stateCtx))

	require.Equal(t, []string{`first`, `second`, `third`}, trkr.Visited())
}
